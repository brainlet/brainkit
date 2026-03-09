// Ported from: assemblyscript/src/compiler.ts compileGlobalLazy (lines 1143-1154),
// compileGlobal (lines 1157-1414).
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// CompileGlobalLazy compiles a lazily-compiled global. Ensures it is compiled exactly once.
// Ported from: assemblyscript/src/compiler.ts compileGlobalLazy (lines 1143-1154).
func (c *Compiler) CompileGlobalLazy(global *program.Global) bool {
	if global.Is(common.CommonFlagsCompiled) {
		return !global.Is(common.CommonFlagsErrored)
	}
	return c.CompileGlobal(global)
}

// CompileGlobal compiles a global variable. Returns true if successful.
// Ported from: assemblyscript/src/compiler.ts compileGlobal (lines 1157-1414).
func (c *Compiler) CompileGlobal(global *program.Global) bool {
	if global.Is(common.CommonFlagsCompiled) {
		return !global.Is(common.CommonFlagsErrored)
	}
	global.Set(common.CommonFlagsCompiled)

	mod := c.Module()
	options := c.Options()
	resolver := c.Resolver()
	isWasm64 := options.IsWasm64()

	// Check for pending (circular) compilation
	if _, pending := c.PendingElements[global]; pending {
		c.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			global.IdentifierNode().GetRange(),
			"Circular global initializer", "", "",
		)
		global.Set(common.CommonFlagsErrored)
		return false
	}
	c.PendingElements[global] = struct{}{}
	defer delete(c.PendingElements, global)

	// Resolve the type
	typeNode := global.TypeNode()
	initializerNode := global.InitializerNode()
	var resolvedType *types.Type

	if typeNode != nil {
		resolvedType = resolver.ResolveType(
			typeNode,
			c.CurrentFlow,
			global.GetParent(),
			nil, // no contextual types
			program.ReportModeReport,
		)
		if resolvedType == nil {
			global.Set(common.CommonFlagsErrored)
			return false
		}
	} else if initializerNode != nil {
		resolvedType = resolver.ResolveExpression(
			initializerNode,
			c.CurrentFlow,
			types.TypeVoid,
			program.ReportModeReport,
		)
		if resolvedType == nil {
			global.Set(common.CommonFlagsErrored)
			return false
		}
	} else {
		// No type, no initializer → error
		c.Error(
			diagnostics.DiagnosticCodeTypeExpected,
			global.IdentifierNode().GetRange(),
			"", "", "",
		)
		global.Set(common.CommonFlagsErrored)
		return false
	}

	// Check type support
	if !c.Program.CheckTypeSupported(resolvedType, global.GetDeclaration()) {
		global.Set(common.CommonFlagsErrored)
		return false
	}

	global.SetType(resolvedType)
	typeRef := resolvedType.ToRef()
	isDeclaredConst := global.Is(common.CommonFlagsConst)
	isDeclaredInline := global.HasDecorator(program.DecoratorFlagsInline)
	isAmbient := global.Is(common.CommonFlagsAmbient)

	// Handle builtin globals (onAccess callbacks)
	internalName := global.GetInternalName()
	if program.BuiltinVariablesOnAccess != nil {
		if _, ok := program.BuiltinVariablesOnAccess[internalName]; ok {
			// Builtins with on-access handlers are compiled on-demand, not here.
			return true
		}
	}

	// Handle ambient (imported) globals
	if isAmbient {
		if isDeclaredInline && initializerNode == nil {
			c.Error(
				diagnostics.DiagnosticCodeDecoratorInlineMustHaveAnInitializer,
				global.IdentifierNode().GetRange(),
				"", "", "",
			)
			global.Set(common.CommonFlagsErrored)
			return false
		}

		// Evaluate constant initializer for ambient globals
		if initializerNode != nil {
			previousFlow := c.CurrentFlow
			c.CurrentFlow = global.File().StartFunction.Flow
			initExpr := c.CompileExpression(initializerNode, resolvedType, ConstraintsConvImplicit)
			c.CurrentFlow = previousFlow

			// Try to precompute to a constant
			precomp := mod.RunExpression(initExpr, module.ExpressionRunnerFlagsDefault, 8, 1)
			if precomp != 0 {
				initExpr = precomp
			}

			if mod.IsConstExpression(initExpr) {
				// Extract constant value and inline
				c.extractConstantValue(global, initExpr, resolvedType, isWasm64)
				if isDeclaredInline || isDeclaredConst {
					return true // fully inlined, no wasm global needed
				}
			}

			// Create a non-mutable global with the constant init
			if mod.IsConstExpression(initExpr) {
				mod.AddGlobal(internalName, typeRef, false, initExpr)
				return true
			}
		}

		// Imported ambient global: add import declaration
		moduleName, elementName := mangleImportName(global, global.GetDeclaration())
		isMutable := !isDeclaredConst

		// Workaround: nullable externref imports need to be initialized with ref.null
		if resolvedType.IsExternalReference() && resolvedType.IsNullableReference() {
			mod.AddGlobal(internalName, typeRef, true, mod.RefNull(typeRef))
		} else {
			mod.AddGlobalImport(internalName, moduleName, elementName, typeRef, isMutable)
		}
		return true
	}

	// Non-ambient global: compile initializer
	if initializerNode != nil {
		previousFlow := c.CurrentFlow
		c.CurrentFlow = global.File().StartFunction.Flow
		initExpr := c.CompileExpression(initializerNode, resolvedType, ConstraintsConvImplicit)
		c.CurrentFlow = previousFlow

		// Try to precompute constant
		precomp := mod.RunExpression(initExpr, module.ExpressionRunnerFlagsDefault, 8, 1)
		if precomp != 0 {
			initExpr = precomp
		}

		if mod.IsConstExpression(initExpr) {
			// Extract constant value for inlining
			c.extractConstantValue(global, initExpr, resolvedType, isWasm64)

			if isDeclaredInline {
				// Fully inlined, no wasm global needed
				return true
			}

			// Create immutable global with constant init
			mod.AddGlobal(internalName, typeRef, false, initExpr)
		} else {
			// Non-constant: create mutable global initialized to zero,
			// set the actual value in the start function.
			mod.AddGlobal(internalName, typeRef, true, c.makeZeroOfType(resolvedType))
			c.CurrentBody = append(c.CurrentBody,
				mod.GlobalSet(internalName, initExpr),
			)
		}
	} else {
		// No initializer: create mutable global initialized to zero
		mod.AddGlobal(internalName, typeRef, true, c.makeZeroOfType(resolvedType))
	}

	return true
}

// extractConstantValue extracts the constant value from a const expression
// and stores it on the global element for later inlining.
func (c *Compiler) extractConstantValue(global *program.Global, expr module.ExpressionRef, resolvedType *types.Type, isWasm64 bool) {
	typeRef := resolvedType.ToRef()
	switch typeRef {
	case module.TypeRefI32:
		global.SetConstantIntegerValue(int64(module.GetConstValueI32(expr)), resolvedType)
	case module.TypeRefI64:
		global.SetConstantIntegerValue(module.GetConstValueI64(expr), resolvedType)
	case module.TypeRefF32:
		global.SetConstantFloatValue(float64(module.GetConstValueF32(expr)), resolvedType)
	case module.TypeRefF64:
		global.SetConstantFloatValue(module.GetConstValueF64(expr), resolvedType)
	}
}

// makeZeroOfType creates a zero/default constant expression for the given type.
// Ported from: assemblyscript/src/compiler.ts makeZero (lines 10082-10119).
func (c *Compiler) makeZeroOfType(typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
		return mod.I32(0)
	case types.TypeKindIsize, types.TypeKindUsize:
		if typ.Size == 64 {
			return mod.I64(0)
		}
		return mod.I32(0)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.I64(0)
	case types.TypeKindF32:
		return mod.F32(0)
	case types.TypeKindF64:
		return mod.F64(0)
	case types.TypeKindV128:
		return mod.V128([16]byte{})
	case types.TypeKindI31:
		if typ.IsNullableReference() {
			return mod.RefNull(typ.ToRef())
		}
		return mod.RefI31(mod.I32(0))
	case types.TypeKindFunc, types.TypeKindExtern, types.TypeKindAny,
		types.TypeKindEq, types.TypeKindStruct, types.TypeKindArray,
		types.TypeKindString, types.TypeKindStringviewWTF8,
		types.TypeKindStringviewWTF16, types.TypeKindStringviewIter:
		if typ.IsNullableReference() {
			return mod.RefNull(typ.ToRef())
		}
		return mod.Unreachable()
	default:
		return mod.RefNull(typ.ToRef())
	}
}

// makeOneOfType creates a constant one of the specified type.
// Ported from: assemblyscript/src/compiler.ts makeOne (lines 10122-10141).
func (c *Compiler) makeOneOfType(typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindBool, types.TypeKindI8, types.TypeKindI16,
		types.TypeKindI32, types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
		return mod.I32(1)
	case types.TypeKindIsize, types.TypeKindUsize:
		if typ.Size == 64 {
			return mod.I64(1)
		}
		return mod.I32(1)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.I64(1)
	case types.TypeKindF32:
		return mod.F32(1)
	case types.TypeKindF64:
		return mod.F64(1)
	case types.TypeKindI31:
		return mod.RefI31(mod.I32(1))
	default:
		return mod.Unreachable()
	}
}

// makeNegOneOfType creates a constant negative one of the specified type.
// Ported from: assemblyscript/src/compiler.ts makeNegOne (lines 10144-10163).
func (c *Compiler) makeNegOneOfType(typ *types.Type) module.ExpressionRef {
	mod := c.Module()
	switch typ.Kind {
	case types.TypeKindI8, types.TypeKindI16, types.TypeKindI32,
		types.TypeKindU8, types.TypeKindU16, types.TypeKindU32:
		return mod.I32(-1)
	case types.TypeKindIsize, types.TypeKindUsize:
		if typ.Size == 64 {
			return mod.I64(-1)
		}
		return mod.I32(-1)
	case types.TypeKindI64, types.TypeKindU64:
		return mod.I64(-1)
	case types.TypeKindF32:
		return mod.F32(-1)
	case types.TypeKindF64:
		return mod.F64(-1)
	case types.TypeKindV128:
		return mod.V128([16]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	case types.TypeKindI31:
		return mod.RefI31(mod.I32(-1))
	default:
		return mod.Unreachable()
	}
}

// compileInlineConstant compiles an inlined constant value.
// Ported from: assemblyscript/src/compiler.ts compileInlineConstant (lines 4794-4876).
func (c *Compiler) compileInlineConstant(element program.Element, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	mod := c.Module()

	switch el := element.(type) {
	case *program.Global:
		globalType := el.GetResolvedType()
		switch el.GetConstantValueKind() {
		case program.ConstantValueKindInteger:
			intValue := el.GetConstantIntegerValue()
			switch contextualType.ToRef() {
			case module.TypeRefI32:
				c.CurrentType = contextualType
				return mod.I32(int32(intValue))
			case module.TypeRefI64:
				c.CurrentType = contextualType
				return mod.I64(intValue)
			case module.TypeRefF32:
				c.CurrentType = types.TypeF32
				return mod.F32(float32(intValue))
			case module.TypeRefF64:
				c.CurrentType = types.TypeF64
				return mod.F64(float64(intValue))
			default:
				switch globalType.ToRef() {
				case module.TypeRefI32:
					c.CurrentType = globalType
					return mod.I32(int32(intValue))
				case module.TypeRefI64:
					c.CurrentType = globalType
					return mod.I64(intValue)
				default:
					c.CurrentType = types.TypeI32
					return mod.I32(int32(intValue))
				}
			}

		case program.ConstantValueKindFloat:
			floatValue := el.GetConstantFloatValue()
			switch contextualType.ToRef() {
			case module.TypeRefF32:
				c.CurrentType = types.TypeF32
				return mod.F32(float32(floatValue))
			case module.TypeRefF64:
				c.CurrentType = types.TypeF64
				return mod.F64(floatValue)
			case module.TypeRefI32:
				c.CurrentType = types.TypeI32
				return mod.I32(int32(floatValue))
			case module.TypeRefI64:
				c.CurrentType = types.TypeI64
				return mod.I64(int64(floatValue))
			default:
				switch globalType.ToRef() {
				case module.TypeRefF32:
					c.CurrentType = globalType
					return mod.F32(float32(floatValue))
				case module.TypeRefF64:
					c.CurrentType = globalType
					return mod.F64(floatValue)
				default:
					c.CurrentType = types.TypeF64
					return mod.F64(floatValue)
				}
			}
		}
		// Not inlined, fall through to global.get
		c.CurrentType = globalType
		return mod.GlobalGet(el.GetInternalName(), globalType.ToRef())

	case *program.EnumValue:
		intValue := el.GetConstantIntegerValue()
		switch contextualType.ToRef() {
		case module.TypeRefI32:
			c.CurrentType = types.TypeI32
			return mod.I32(int32(intValue))
		case module.TypeRefI64:
			c.CurrentType = types.TypeI64
			return mod.I64(intValue)
		case module.TypeRefF32:
			c.CurrentType = types.TypeF32
			return mod.F32(float32(intValue))
		case module.TypeRefF64:
			c.CurrentType = types.TypeF64
			return mod.F64(float64(intValue))
		default:
			c.CurrentType = types.TypeI32
			return mod.I32(int32(intValue))
		}
	}

	c.CurrentType = contextualType
	return mod.Unreachable()
}

// ensureRuntimeFunction ensures a function is in the function table and returns its table index.
// Ported from: assemblyscript/src/compiler.ts ensureRuntimeFunction (lines 10202-10230).
func (c *Compiler) ensureRuntimeFunction(instance *program.Function) int32 {
	// Check if already in the function table
	for i, fn := range c.FunctionTable {
		if fn == instance {
			return int32(i)
		}
	}
	// Add to function table
	index := int32(len(c.FunctionTable))
	c.FunctionTable = append(c.FunctionTable, instance)
	return index
}

// evaluateCondition tries to determine whether a condition expression is
// statically known to be true, false, or unknown.
// Ported from: assemblyscript/src/compiler.ts evaluateCondition (lines 10063-10077).
func (c *Compiler) evaluateCondition(expr module.ExpressionRef) flow.ConditionKind {
	typ := module.GetExpressionType(expr)
	if typ == module.TypeRefUnreachable {
		return flow.ConditionKindUnknown
	}
	mod := c.Module()
	evaled := mod.RunExpression(expr, module.ExpressionRunnerFlagsDefault, 1, 0)
	if evaled != 0 {
		if module.GetConstValueI32(evaled) != 0 {
			return flow.ConditionKindTrue
		}
		return flow.ConditionKindFalse
	}
	return flow.ConditionKindUnknown
}

// Ported from: assemblyscript/src/compiler.ts compileGlobalLazy (lines 1143-1154),
// compileGlobal (lines 1157-1414).
package compiler

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// CompileGlobalLazy compiles a lazily-compiled global. If the global is not @lazy,
// not @builtin, and not ambient, it means it is used before its declaration, which is an error.
// Ported from: assemblyscript/src/compiler.ts compileGlobalLazy (lines 1143-1154).
func (c *Compiler) CompileGlobalLazy(global *program.Global, reportNode ast.Node) bool {
	if global.Is(common.CommonFlagsCompiled) {
		return !global.Is(common.CommonFlagsErrored)
	}
	if global.HasAnyDecorator(program.DecoratorFlagsLazy|program.DecoratorFlagsBuiltin) || global.Is(common.CommonFlagsAmbient) {
		return c.CompileGlobal(global) // compile now
	}
	// Otherwise the global is used before its initializer executes
	c.ErrorRelated(
		diagnostics.DiagnosticCodeVariable0UsedBeforeItsDeclaration,
		reportNode.GetRange(), global.IdentifierNode().GetRange(),
		global.GetInternalName(), "", "",
	)
	return false
}

// CompileGlobal compiles a global variable. Returns true if successful.
// Ported from: assemblyscript/src/compiler.ts compileGlobal (lines 1157-1414).
func (c *Compiler) CompileGlobal(global *program.Global) bool {
	if global.Is(common.CommonFlagsCompiled) {
		return !global.Is(common.CommonFlagsErrored)
	}
	global.Set(common.CommonFlagsCompiled)

	pendingElements := c.PendingElements
	pendingElements[global] = struct{}{}

	mod := c.Module()
	var initExpr module.ExpressionRef
	typeNode := global.TypeNode()
	initializerNode := global.InitializerNode()

	if !global.Is(common.CommonFlagsResolved) {

		// Resolve type if annotated
		if typeNode != nil {
			resolver := c.Resolver()
			resolvedType := resolver.ResolveType(
				typeNode,
				nil, // null flow, matching TS: this.resolver.resolveType(typeNode, null, global.parent)
				global.GetParent(),
				nil, // no contextual types
				program.ReportModeReport,
			)
			if resolvedType == nil {
				global.Set(common.CommonFlagsErrored)
				delete(pendingElements, global)
				return false
			}
			if resolvedType == types.TypeVoid {
				c.Error(
					diagnostics.DiagnosticCodeTypeExpected,
					typeNode.GetRange(),
					"", "", "",
				)
				global.Set(common.CommonFlagsErrored)
				delete(pendingElements, global)
				return false
			}
			global.SetType(resolvedType)
			c.Program.CheckTypeSupported(resolvedType, typeNode)

			// Otherwise infer type from initializer
		} else if initializerNode != nil {
			previousFlow := c.CurrentFlow
			if global.HasDecorator(program.DecoratorFlagsLazy) {
				c.CurrentFlow = global.File().StartFunction.Flow
			}
			initExpr = c.CompileExpression(initializerNode, types.TypeAuto, // reports
				ConstraintsMustWrap|ConstraintsPreferStatic,
			)
			c.CurrentFlow = previousFlow
			if c.CurrentType == types.TypeVoid {
				c.Error(
					diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
					initializerNode.GetRange(),
					c.CurrentType.String(), "<auto>", "",
				)
				global.Set(common.CommonFlagsErrored)
				delete(pendingElements, global)
				return false
			}
			global.SetType(c.CurrentType)

			// Error if there's neither a type nor an initializer
		} else {
			c.Error(
				diagnostics.DiagnosticCodeTypeExpected,
				global.IdentifierNode().GetRange().AtEnd(),
				"", "", "",
			)
			global.Set(common.CommonFlagsErrored)
			delete(pendingElements, global)
			return false
		}
	}

	// Handle builtins like '__heap_base' that need to be resolved but are added explicitly
	if global.HasDecorator(program.DecoratorFlagsBuiltin) {
		internalName := global.GetInternalName()
		if fn, ok := BuiltinVariablesOnCompile[internalName]; ok {
			fn(&BuiltinVariableContext{
				Compiler: c,
				Element:  global,
			})
		}
		delete(pendingElements, global)
		return true
	}

	typ := global.GetResolvedType()

	// Enforce either an initializer, a definitive assignment or a nullable type
	// to guarantee soundness when globals are accessed.
	if initializerNode == nil && !global.Is(common.CommonFlagsDefinitelyAssigned) &&
		typ.IsReference() && !typ.IsNullableReference() {
		c.Error(
			diagnostics.DiagnosticCodeInitializerDefinitiveAssignmentOrNullableTypeExpected,
			global.IdentifierNode().GetRange(),
			"", "", "",
		)
	}

	typeRef := typ.ToRef()
	isDeclaredConstant := global.Is(common.CommonFlagsConst) ||
		global.Is(common.CommonFlagsStatic|common.CommonFlagsReadonly)
	isDeclaredInline := global.HasDecorator(program.DecoratorFlagsInline)

	// Handle imports
	if global.Is(common.CommonFlagsAmbient) {
		options := c.Options()
		// Constant global or mutable globals enabled
		if isDeclaredConstant || options.HasFeature(common.FeatureMutableGlobals) {
			moduleName, elementName := mangleImportName(global, global.GetDeclaration())
			c.Program.MarkModuleImport(moduleName, elementName, global)
			mod.AddGlobalImport(
				global.GetInternalName(),
				moduleName,
				elementName,
				typeRef,
				!isDeclaredConstant,
			)
			delete(pendingElements, global)
			if !c.DesiresExportRuntime && lowerRequiresExportRuntime(typ) {
				c.DesiresExportRuntime = true
			}
			return true
		}

		// Importing mutable globals is not supported in the MVP
		c.Error(
			diagnostics.DiagnosticCodeFeature0IsNotEnabled,
			global.GetDeclaration().GetRange(),
			"mutable-globals", "", "",
		)
		global.Set(common.CommonFlagsErrored)
		delete(pendingElements, global)
		return false
	}

	// The MVP does not yet support initializer expressions other than constants and gets of
	// imported immutable globals, hence such initializations must be performed in the start.
	initializeInStart := false

	// Evaluate initializer if present
	if initializerNode != nil {
		if initExpr == 0 {
			previousFlow := c.CurrentFlow
			if global.HasDecorator(program.DecoratorFlagsLazy) {
				c.CurrentFlow = global.File().StartFunction.Flow
			}
			initExpr = c.CompileExpression(initializerNode, typ,
				ConstraintsConvImplicit|ConstraintsMustWrap|ConstraintsPreferStatic,
			)
			c.CurrentFlow = previousFlow
		}

		// If not a constant expression, attempt to precompute
		if !mod.IsConstExpression(initExpr) {
			if isDeclaredConstant {
				precomp := mod.RunExpression(initExpr, module.ExpressionRunnerFlagsPreserveSideeffects, 50, 1)
				if precomp != 0 {
					initExpr = precomp
				} else {
					initializeInStart = true
				}
			} else {
				initializeInStart = true
			}
		}

		// Handle special case of initializing from imported immutable global
		if initializeInStart && module.GetExpressionId(initExpr) == module.ExpressionIdGlobalGet {
			fromName := module.GetGlobalGetName(initExpr)
			if fromName == "" {
				panic("compileGlobal: GlobalGet expression has empty name")
			}
			{
				globalRef := mod.GetGlobal(fromName)
				if globalRef != 0 && !module.IsGlobalMutable(globalRef) {
					elementsByName := c.Program.ElementsByNameMap
					if elem, ok := elementsByName[fromName]; ok {
						if elem.Is(common.CommonFlagsAmbient) {
							initializeInStart = false
						}
					}
				}
			}
		}

		// Explicitly inline if annotated
		if isDeclaredInline {
			if initializeInStart {
				c.Warning(
					diagnostics.DiagnosticCodeMutableValueCannotBeInlined,
					initializerNode.GetRange(),
					"", "", "",
				)
			} else {
				exprType := module.GetExpressionType(initExpr)
				switch exprType {
				case module.TypeRefI32:
					global.SetConstantIntegerValue(int64(module.GetConstValueI32(initExpr)), typ)
				case module.TypeRefI64:
					global.SetConstantIntegerValue(module.GetConstValueI64(initExpr), typ)
				case module.TypeRefF32:
					global.SetConstantFloatValue(float64(module.GetConstValueF32(initExpr)), typ)
				case module.TypeRefF64:
					global.SetConstantFloatValue(module.GetConstValueF64(initExpr), typ)
				default:
					global.Set(common.CommonFlagsErrored)
					delete(pendingElements, global)
					return false
				}
				global.Set(common.CommonFlagsInlined) // inline the value from now on
			}
		}

		// Initialize to zero if there's no initializer
	} else {
		if global.Is(common.CommonFlagsInlined) {
			initExpr = c.compileInlineConstant(global, typ, ConstraintsPreferStatic)
		} else {
			initExpr = c.makeZeroOfType(typ)
		}
	}

	internalName := global.GetInternalName()

	if initializeInStart { // initialize to mutable zero and set the actual value in start
		if isDeclaredInline {
			c.Error(
				diagnostics.DiagnosticCodeDecorator0IsNotValidHere,
				ast.FindDecorator(ast.DecoratorKindInline, global.DecoratorNodes()).GetRange(),
				"inline", "", "",
			)
		}
		internalType := typ
		if typ.IsExternalReference() && !typ.Is(types.TypeFlagNullable) {
			// There is no default value for non-nullable external references, so
			// make the global nullable internally and use `null`.
			global.Set(common.CommonFlagsInternallyNullable)
			internalType = typ.AsNullable()
		}
		mod.AddGlobal(internalName, internalType.ToRef(), true, c.makeZeroOfType(internalType))
		c.CurrentBody = append(c.CurrentBody,
			mod.GlobalSet(internalName, initExpr),
		)
	} else if !isDeclaredInline { // compile normally
		mod.AddGlobal(internalName, typeRef, !isDeclaredConstant, initExpr)
	}
	delete(pendingElements, global)
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
// Ported from: assemblyscript/src/compiler.ts compileInlineConstant (lines 3350-3429).
func (c *Compiler) compileInlineConstant(element program.VariableLikeElement, contextualType *types.Type, constraints Constraints) module.ExpressionRef {
	// assert(element.is(CommonFlags.Inlined | CommonFlags.Resolved))
	if !element.Is(common.CommonFlagsInlined | common.CommonFlagsResolved) {
		panic("compileInlineConstant: element must be inlined and resolved")
	}
	mod := c.Module()
	typ := element.GetResolvedType()
	c.CurrentType = typ

	switch typ.Kind {
	case types.TypeKindBool:
		if element.GetConstantValueKind() == program.ConstantValueKindInteger {
			val := int32(0)
			if element.GetConstantIntegerValue() != 0 {
				val = 1
			}
			return mod.I32(val)
		}
		return mod.I32(0)

	case types.TypeKindI8, types.TypeKindI16:
		// Signed small integers: sign-extend via shift
		shift := typ.ComputeSmallIntegerShift(types.TypeI32)
		if element.GetConstantValueKind() == program.ConstantValueKindInteger {
			return mod.I32(int32(element.GetConstantIntegerValue()) << shift >> shift)
		}
		return mod.I32(0) // recognized by canOverflow

	case types.TypeKindU8, types.TypeKindU16:
		// Unsigned small integers: mask
		mask := typ.ComputeSmallIntegerMask(types.TypeI32)
		if element.GetConstantValueKind() == program.ConstantValueKindInteger {
			return mod.I32(int32(element.GetConstantIntegerValue()) & mask)
		}
		return mod.I32(0) // recognized by canOverflow

	case types.TypeKindI32, types.TypeKindU32:
		if element.GetConstantValueKind() == program.ConstantValueKindInteger {
			return mod.I32(int32(element.GetConstantIntegerValue()))
		}
		return mod.I32(0)

	case types.TypeKindIsize, types.TypeKindUsize:
		if !c.Options().IsWasm64() {
			if element.GetConstantValueKind() == program.ConstantValueKindInteger {
				return mod.I32(int32(element.GetConstantIntegerValue()))
			}
			return mod.I32(0)
		}
		// fall-through to I64/U64
		fallthrough

	case types.TypeKindI64, types.TypeKindU64:
		if element.GetConstantValueKind() == program.ConstantValueKindInteger {
			return mod.I64(element.GetConstantIntegerValue())
		}
		return mod.I64(0)

	case types.TypeKindF64:
		// monkey-patch for converting built-in floats to f32 implicitly
		if !(element.HasDecorator(program.DecoratorFlagsBuiltin) && contextualType == types.TypeF32) {
			return mod.F64(element.GetConstantFloatValue())
		}
		// otherwise fall-through: basically precomputes f32.demote/f64 of NaN / Infinity
		c.CurrentType = types.TypeF32
		fallthrough

	case types.TypeKindF32:
		return mod.F32(float32(element.GetConstantFloatValue()))

	default:
		return mod.Unreachable()
	}
}

// ensureRuntimeFunction ensures that a runtime counterpart of the specified function exists
// and returns its memory address (i64).
// Ported from: assemblyscript/src/compiler.ts ensureRuntimeFunction (lines 2124-2145).
func (c *Compiler) ensureRuntimeFunction(instance *program.Function) int64 {
	if !instance.Is(common.CommonFlagsCompiled) || instance.Is(common.CommonFlagsStub) {
		panic("ensureRuntimeFunction: instance must be compiled and not a stub")
	}

	prog := c.Program
	memorySegment := instance.MemorySegment
	if memorySegment == nil {
		// Add to the function table
		functionTable := c.FunctionTable
		tableBase := c.Options().TableBase
		if tableBase == 0 {
			tableBase = 1 // leave first elem blank
		}
		index := int32(tableBase) + int32(len(functionTable))
		c.FunctionTable = append(c.FunctionTable, instance)

		// Create runtime Function class instance buffer
		// Ported from: compiler.ts:2138-2142
		// TODO: Full port requires Class.CreateBuffer() and Class.WriteField() infrastructure.
		// TS does:
		//   let rtInstance = assert(this.resolver.resolveClass(program.functionPrototype, [instance.type]));
		//   let buf = rtInstance.createBuffer();
		//   assert(rtInstance.writeField("_index", index, buf));
		//   assert(rtInstance.writeField("_env", 0, buf));
		//   instance.memorySegment = memorySegment = this.addRuntimeMemorySegment(buf);
		//
		// createBuffer (program.ts:4683-4695) allocates a byte buffer with:
		//   - blockOverhead + computeBlockSize(nextMemoryOffset) bytes
		//   - Writes OBJECT header fields: mmInfo, gcInfo, gcInfo2, rtId, rtSize
		// writeField (program.ts:4698-4740) writes a value into the buffer at:
		//   - property.memoryOffset + totalOverhead offset
		//   - Using the property's type to determine byte width

		// Resolve the runtime Function<T> class
		funcType := instance.GetResolvedType()
		rtInstance := c.Resolver().ResolveClass(
			prog.FunctionPrototype(),
			[]*types.Type{funcType},
			nil,
			program.ReportModeReport,
		)
		if rtInstance == nil {
			panic("ensureRuntimeFunction: failed to resolve Function class")
		}

		// Allocate buffer: blockOverhead + enough space for the Function instance
		totalOverhead := prog.TotalOverhead()
		blockOverhead := prog.BlockOverhead()
		payloadSize := int32(rtInstance.NextMemoryOffset)
		blockSize := prog.ComputeBlockSize(payloadSize, true)
		buf := make([]byte, blockOverhead+blockSize)

		// Write OBJECT header fields
		objectInstance := prog.ObjectInstance()
		writeRuntimeField(objectInstance, "mmInfo", int64(blockSize), buf, 0)
		writeRuntimeField(objectInstance, "gcInfo", 0, buf, 0)
		writeRuntimeField(objectInstance, "gcInfo2", 0, buf, 0)
		writeRuntimeField(objectInstance, "rtId", int64(rtInstance.Id()), buf, 0)
		writeRuntimeField(objectInstance, "rtSize", int64(payloadSize), buf, 0)

		// Write Function instance fields
		writeRuntimeField(rtInstance, "_index", int64(index), buf, totalOverhead)
		writeRuntimeField(rtInstance, "_env", 0, buf, totalOverhead)

		memorySegment = c.addRuntimeMemorySegment(buf)
		instance.MemorySegment = memorySegment
	}

	// Return memory address: segment offset + totalOverhead
	// The segment Offset is an ExpressionRef (i32.const or i64.const), we need the raw value.
	segmentOffset := module.GetConstValueI64(memorySegment.Offset)
	return segmentOffset + int64(prog.TotalOverhead())
}

// writeRuntimeField writes a field value into a byte buffer at the field's memory offset.
// Simplified port of Class.writeField from program.ts:4698-4767.
// Only handles i32 and usize fields, which covers OBJECT header and Function fields.
func writeRuntimeField(clazz *program.Class, fieldName string, value int64, buf []byte, baseOffset int32) {
	member := clazz.GetMember(fieldName)
	if member == nil {
		return
	}
	propProto, ok := member.(*program.PropertyPrototype)
	if !ok {
		return
	}
	prop := propProto.PropertyInstance
	if prop == nil || !prop.IsField() || prop.MemoryOffset < 0 {
		return
	}
	offset := int(baseOffset) + int(prop.MemoryOffset)
	typeKind := prop.GetType().Kind
	switch typeKind {
	case types.TypeKindI32, types.TypeKindU32:
		if offset+4 <= len(buf) {
			buf[offset] = byte(value)
			buf[offset+1] = byte(value >> 8)
			buf[offset+2] = byte(value >> 16)
			buf[offset+3] = byte(value >> 24)
		}
	case types.TypeKindIsize, types.TypeKindUsize:
		// For wasm32, write 4 bytes; for wasm64, write 8 bytes.
		// Since we only handle static buffers during compilation,
		// and the target is determined at compile time, we write 4 bytes (wasm32 default).
		if offset+4 <= len(buf) {
			buf[offset] = byte(value)
			buf[offset+1] = byte(value >> 8)
			buf[offset+2] = byte(value >> 16)
			buf[offset+3] = byte(value >> 24)
		}
	case types.TypeKindI64, types.TypeKindU64:
		if offset+8 <= len(buf) {
			buf[offset] = byte(value)
			buf[offset+1] = byte(value >> 8)
			buf[offset+2] = byte(value >> 16)
			buf[offset+3] = byte(value >> 24)
			buf[offset+4] = byte(value >> 32)
			buf[offset+5] = byte(value >> 40)
			buf[offset+6] = byte(value >> 48)
			buf[offset+7] = byte(value >> 56)
		}
	}
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
	evaled := mod.RunExpression(expr, module.ExpressionRunnerFlagsDefault, 50, 1)
	if evaled != 0 {
		if module.GetConstValueI32(evaled) != 0 {
			return flow.ConditionKindTrue
		}
		return flow.ConditionKindFalse
	}
	return flow.ConditionKindUnknown
}

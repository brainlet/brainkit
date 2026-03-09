// Ported from: assemblyscript/src/compiler.ts compileFunction (lines 1571-1757),
// compileFunctionBody (lines 1760-1873), mangleImportName (lines 10632-10687),
// checkSignatureSupported (lines 10038-10060).
package compiler

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// CompileFunction compiles a function. Returns true if successful.
// Ported from: assemblyscript/src/compiler.ts compileFunction (lines 1571-1757).
func (c *Compiler) CompileFunction(instance *program.Function) bool {
	return c.compileFunctionImpl(instance, false)
}

// CompileFunctionForced compiles a function even if lazy. Returns true if successful.
func (c *Compiler) CompileFunctionForced(instance *program.Function) bool {
	return c.compileFunctionImpl(instance, true)
}

// compileFunctionImpl implements function compilation with optional force flag.
func (c *Compiler) compileFunctionImpl(instance *program.Function, forceStdAlternative bool) bool {
	if instance.Is(common.CommonFlagsCompiled) {
		return !instance.Is(common.CommonFlagsErrored)
	}

	if !forceStdAlternative {
		if instance.HasDecorator(program.DecoratorFlagsBuiltin) {
			return true
		}
		if instance.HasDecorator(program.DecoratorFlagsLazy) {
			c.LazyFunctions[instance] = struct{}{}
			return true
		}
	}

	// ensure the function has no duplicate parameters
	prototype := instance.Prototype
	fnTypeNode := prototype.FunctionTypeNode()
	if fnTypeNode != nil {
		parameters := fnTypeNode.Parameters
		numParameters := len(parameters)
		if numParameters >= 2 {
			visited := make(map[string]struct{})
			visited[parameters[0].Name.Text] = struct{}{}
			for i := 1; i < numParameters; i++ {
				paramIdentifier := parameters[i].Name
				paramName := paramIdentifier.Text
				if _, exists := visited[paramName]; !exists {
					visited[paramName] = struct{}{}
				} else {
					c.Error(
						diagnostics.DiagnosticCodeDuplicateIdentifier0,
						paramIdentifier.GetRange(),
						paramName, "", "",
					)
				}
			}
		}
	}

	instance.Set(common.CommonFlagsCompiled)
	pendingElements := c.PendingElements
	pendingElements[instance] = struct{}{}

	previousType := c.CurrentType
	mod := c.Module()
	signature := instance.Signature
	bodyNode := prototype.BodyNode()
	declarationNode := instance.GetDeclaration()

	// assert: must be FunctionDeclaration or MethodDeclaration
	kind := declarationNode.GetKind()
	_ = kind // validated by program

	// Check signature types are supported
	if fnTypeNode != nil {
		c.checkSignatureSupported(signature, fnTypeNode)
	}

	var funcRef module.FunctionRef

	// concrete function
	if bodyNode != nil {
		// must not be ambient
		if instance.Is(common.CommonFlagsAmbient) {
			c.Error(
				diagnostics.DiagnosticCodeAnImplementationCannotBeDeclaredInAmbientContexts,
				instance.IdentifierNode().GetRange(),
				"", "", "",
			)
		}

		// cannot have an annotated external name or code
		if instance.HasAnyDecorator(program.DecoratorFlagsExternal | program.DecoratorFlagsExternalJs) {
			decoratorNodes := instance.DecoratorNodes()
			if decorator := ast.FindDecorator(ast.DecoratorKindExternal, decoratorNodes); decorator != nil {
				c.Error(
					diagnostics.DiagnosticCodeDecorator0IsNotValidHere,
					decorator.GetRange(),
					"external", "", "",
				)
			}
			if decorator := ast.FindDecorator(ast.DecoratorKindExternalJs, decoratorNodes); decorator != nil {
				c.Error(
					diagnostics.DiagnosticCodeDecorator0IsNotValidHere,
					decorator.GetRange(),
					"external.js", "", "",
				)
			}
		}

		// compile body in this function's context
		previousFlow := c.CurrentFlow
		funcFlow := instance.Flow
		c.CurrentFlow = funcFlow
		stmts := make([]module.ExpressionRef, 0)

		if !c.compileFunctionBody(instance, &stmts) {
			stmts = append(stmts, mod.Unreachable())
		}

		c.CurrentFlow = previousFlow

		// create the function
		funcRef = mod.AddFunction(
			instance.GetInternalName(),
			signature.ParamRefs(),
			signature.ResultRefs(),
			typesToRefs(instance.GetNonParameterLocalTypes()),
			mod.Flatten(stmts, signature.ReturnType.ToRef()),
		)

	} else if instance.Is(common.CommonFlagsAmbient) {
		// imported function
		moduleName, elementName := mangleImportName(instance, declarationNode)
		c.Program.MarkModuleImport(moduleName, elementName, instance)
		mod.AddFunctionImport(
			instance.GetInternalName(),
			moduleName,
			elementName,
			signature.ParamRefs(),
			signature.ResultRefs(),
		)
		funcRef = mod.GetFunction(instance.GetInternalName())
		if !c.DesiresExportRuntime {
			thisType := signature.ThisType
			if (thisType != nil && liftRequiresExportRuntime(thisType)) ||
				lowerRequiresExportRuntime(signature.ReturnType) {
				c.DesiresExportRuntime = true
			} else {
				parameterTypes := signature.ParameterTypes
				for _, pt := range parameterTypes {
					if liftRequiresExportRuntime(pt) {
						c.DesiresExportRuntime = true
						break
					}
				}
			}
		}

	} else if instance.Is(common.CommonFlagsAbstract) ||
		(instance.GetParent() != nil && instance.GetParent().GetElementKind() == program.ElementKindInterface) {
		// abstract or interface function
		funcRef = mod.AddFunction(
			instance.GetInternalName(),
			signature.ParamRefs(),
			signature.ResultRefs(),
			nil,
			mod.Unreachable(),
		)

	} else {
		// built-in field accessor?
		if instance.IsAny(common.CommonFlagsGet | common.CommonFlagsSet) {
			decl := instance.GetDeclaration().(*ast.FunctionDeclaration)
			propertyName := decl.Name.Text
			propertyParent := instance.GetParent().GetMember(propertyName)
			if propertyParent != nil && propertyParent.GetElementKind() == program.ElementKindPropertyPrototype {
				propertyProto := propertyParent.(*program.PropertyPrototype)
				propertyInstance := propertyProto.PropertyInstance
				if propertyInstance != nil && propertyInstance.IsField() {
					if instance.Is(common.CommonFlagsGet) {
						funcRef = c.makeBuiltinFieldGetter(propertyInstance)
					} else {
						funcRef = c.makeBuiltinFieldSetter(propertyInstance)
					}
				}
			}
		}
		if funcRef == 0 {
			c.Error(
				diagnostics.DiagnosticCodeFunctionImplementationIsMissingOrNotImmediatelyFollowingTheDeclaration,
				instance.IdentifierNode().GetRange(),
				"", "", "",
			)
			instance.Set(common.CommonFlagsErrored)
		}
	}

	if instance.Is(common.CommonFlagsAmbient) || instance.Is(common.CommonFlagsExport) {
		// Verify and print warn if signature has v128 type for imported or exported functions
		if signature.HasVectorValueOperands() {
			var rng *diagnostics.Range
			if fnTypeNode != nil {
				if signature.ReturnType == types.TypeV128 {
					rng = fnTypeNode.ReturnType.GetRange()
				} else {
					indices := signature.GetVectorValueOperandIndices()
					if len(indices) > 0 {
						firstIndex := indices[0]
						if int(firstIndex) < len(fnTypeNode.Parameters) {
							rng = fnTypeNode.Parameters[firstIndex].GetRange()
						} else {
							rng = fnTypeNode.GetRange()
						}
					}
				}
			}
			c.Warning(
				diagnostics.DiagnosticCodeExchangeOf0ValuesIsNotSupportedByAllEmbeddings,
				rng,
				"v128", "", "",
			)
		}
	}

	instance.Finalize(mod, funcRef)
	c.CurrentType = previousType
	delete(pendingElements, instance)
	return true
}

// compileFunctionBody compiles the body of a function within the specified flow.
// Ported from: assemblyscript/src/compiler.ts compileFunctionBody (lines 1760-1873).
func (c *Compiler) compileFunctionBody(instance *program.Function, stmts *[]module.ExpressionRef) bool {
	mod := c.Module()
	bodyNode := instance.Prototype.BodyNode()
	returnType := instance.Signature.ReturnType
	funcFlow := c.CurrentFlow
	var thisLocal *program.Local
	if instance.Signature.ThisType != nil {
		localRef := funcFlow.LookupLocal(common.CommonNameThis)
		if localRef != nil {
			thisLocal = localRef.(*program.Local)
		}
	}
	bodyStartIndex := len(*stmts)

	// compile statements
	if bodyNode.GetKind() == ast.NodeKindBlock {
		blockStmt := bodyNode.(*ast.BlockStatement)
		*stmts = c.CompileStatements(blockStmt.Statements, *stmts)
	} else {
		// must be an expression statement if not a block (arrow function)
		exprStmt := bodyNode.(*ast.ExpressionStatement)
		expr := c.CompileExpression(exprStmt.Expression, returnType, ConstraintsConvImplicit)
		if !funcFlow.CanOverflow(expr, returnType) {
			funcFlow.SetFlag(flow.FlowFlagReturnsWrapped)
		}
		if funcFlow.IsNonnull(expr, returnType) {
			funcFlow.SetFlag(flow.FlowFlagReturnsNonNull)
		}
		*stmts = append(*stmts, expr)

		if !funcFlow.Is(flow.FlowFlagTerminates) {
			if !funcFlow.CanOverflow(expr, returnType) {
				funcFlow.SetFlag(flow.FlowFlagReturnsWrapped)
			}
			if funcFlow.IsNonnull(expr, returnType) {
				funcFlow.SetFlag(flow.FlowFlagReturnsNonNull)
			}
			funcFlow.SetFlag(flow.FlowFlagReturns | flow.FlowFlagTerminates)
		}
	}

	// Make constructors return their instance pointer, and prepend a conditional
	// allocation if any code path accesses `this`.
	if instance.Is(common.CommonFlagsConstructor) {
		parent := instance.GetParent()
		classInstance := parent.(*program.Class)

		if funcFlow.IsAny(flow.FlowFlagAccessesThis|flow.FlowFlagConditionallyAccessesThis) ||
			!funcFlow.Is(flow.FlowFlagTerminates) {

			// Allocate `this` if not a super call, and initialize fields
			allocStmts := make([]module.ExpressionRef, 0)
			allocStmts = append(allocStmts,
				c.makeConditionalAllocation(classInstance, thisLocal.Index),
			)
			c.makeFieldInitializationInConstructor(classInstance, &allocStmts)

			// Insert right before the body
			body := *stmts
			body = append(body, 0) // grow by one
			for i := len(body) - 1; i > bodyStartIndex; i-- {
				body[i] = body[i-1]
			}
			body[bodyStartIndex] = mod.Flatten(allocStmts, module.TypeRefNone)
			*stmts = body

			// Just prepended allocation is dropped when returning non-'this'
			if funcFlow.Is(flow.FlowFlagMayReturnNonThis) {
				if c.Options().Pedantic {
					c.Pedantic(
						diagnostics.DiagnosticCodeExplicitlyReturningConstructorDropsThisAllocation,
						instance.IdentifierNode().GetRange(),
						"", "", "",
					)
				}
			}
		}

		// Returning something else than 'this' would break 'super()' calls
		if funcFlow.Is(flow.FlowFlagMayReturnNonThis) && !classInstance.HasDecorator(program.DecoratorFlagsFinal) {
			c.Error(
				diagnostics.DiagnosticCodeAClassWithAConstructorExplicitlyReturningSomethingElseThanThisMustBeFinal,
				classInstance.IdentifierNode().GetRange(),
				"", "", "",
			)
		}

		// Implicitly return `this` if the flow falls through
		if !funcFlow.Is(flow.FlowFlagTerminates) {
			*stmts = append(*stmts,
				mod.LocalGet(thisLocal.Index, thisLocal.GetType().ToRef()),
			)
			funcFlow.SetFlag(flow.FlowFlagReturns | flow.FlowFlagReturnsNonNull | flow.FlowFlagTerminates)
		}

		// check that super has been called if this is a derived class
		if classInstance.Base != nil && !classInstance.Prototype.ImplicitlyExtendsObject && !funcFlow.Is(flow.FlowFlagCallsSuper) {
			c.Error(
				diagnostics.DiagnosticCodeConstructorsForDerivedClassesMustContainASuperCall,
				instance.Prototype.GetDeclaration().GetRange(),
				"", "", "",
			)
		}

	} else if returnType != types.TypeVoid && !funcFlow.Is(flow.FlowFlagTerminates) {
		// if this is a normal function, make sure that all branches terminate
		if fnTypeNode := instance.Prototype.FunctionTypeNode(); fnTypeNode != nil {
			c.Error(
				diagnostics.DiagnosticCodeAFunctionWhoseDeclaredTypeIsNotVoidMustReturnAValue,
				fnTypeNode.ReturnType.GetRange(),
				"", "", "",
			)
		}
		return false // not recoverable
	}

	return true
}

// checkSignatureSupported checks that all types in a signature are supported.
// Ported from: assemblyscript/src/compiler.ts checkSignatureSupported (lines 10038-10060).
func (c *Compiler) checkSignatureSupported(signature *types.Signature, reportNode *ast.FunctionTypeNode) bool {
	supported := true
	explicitThisType := reportNode.ExplicitThisType
	if explicitThisType != nil {
		if signature.ThisType != nil {
			if !c.Program.CheckTypeSupported(signature.ThisType, explicitThisType) {
				supported = false
			}
		}
	}
	parameterTypes := signature.ParameterTypes
	parameterNodes := reportNode.Parameters
	for i, pt := range parameterTypes {
		var parameterReportNode ast.Node
		if i < len(parameterNodes) {
			parameterReportNode = parameterNodes[i]
		} else {
			parameterReportNode = reportNode
		}
		if !c.Program.CheckTypeSupported(pt, parameterReportNode) {
			supported = false
		}
	}
	if !c.Program.CheckTypeSupported(signature.ReturnType, reportNode.ReturnType) {
		supported = false
	}
	return supported
}

// mangleImportName computes the import module and element names for an imported function.
// Returns (moduleName, elementName).
// Ported from: assemblyscript/src/compiler.ts mangleImportName (lines 10632-10687).
func mangleImportName(element program.Element, declaration ast.Node) (string, string) {
	// by default, use the file name as the module name
	rng := declaration.GetRange()
	moduleName := ""
	if rng != nil {
		if src, ok := rng.Source.(*ast.Source); ok {
			moduleName = src.SimplePath
		}
	}
	// and the internal name of the element within that file as the element name
	elementName := program.MangleInternalName(
		element.GetName(), element.GetParent(), element.Is(common.CommonFlagsInstance), true,
	)

	// override module name if a `module` statement is present
	if decl, ok := declaration.(*ast.FunctionDeclaration); ok {
		if decl.HasOverriddenModule {
			moduleName = decl.OverriddenModuleName
		}
	}

	if !element.HasDecorator(program.DecoratorFlagsExternal) {
		return moduleName, elementName
	}

	prog := element.GetProgram()
	decoratorNodes := element.(program.DeclaredElement).DecoratorNodes()
	decorator := ast.FindDecorator(ast.DecoratorKindExternal, decoratorNodes)
	if decorator == nil {
		return moduleName, elementName
	}

	args := decorator.Args
	if len(args) > 0 {
		arg := args[0]
		if lit, ok := arg.(*ast.StringLiteralExpression); ok {
			elementName = lit.Value
			if len(args) >= 2 {
				arg2 := args[1]
				if lit2, ok := arg2.(*ast.StringLiteralExpression); ok {
					moduleName = elementName
					elementName = lit2.Value
					if len(args) > 2 {
						prog.Error(
							diagnostics.DiagnosticCodeExpected0ArgumentsButGot1,
							decorator.GetRange(),
							"2", intToString(len(args)),
						)
					}
				} else {
					prog.Error(
						diagnostics.DiagnosticCodeStringLiteralExpected,
						arg2.GetRange(),
					)
				}
			}
		} else {
			prog.Error(
				diagnostics.DiagnosticCodeStringLiteralExpected,
				arg.GetRange(),
			)
		}
	} else {
		prog.Error(
			diagnostics.DiagnosticCodeExpectedAtLeast0ArgumentsButGot1,
			decorator.GetRange(),
			"1", "0",
		)
	}

	return moduleName, elementName
}

// liftRequiresExportRuntime tests if lifting a type requires the exported runtime.
// Ported from: assemblyscript/src/bindings/js.ts liftRequiresExportRuntime.
func liftRequiresExportRuntime(typ *types.Type) bool {
	if !typ.IsInternalReference() {
		return false
	}
	// Internal references (managed objects, functions) generally require the runtime.
	// A full implementation would check specific class hierarchies (ArrayBuffer, String, etc.)
	// for exceptions. For now, conservatively return true for all internal references.
	return true
}

// lowerRequiresExportRuntime tests if lowering a type requires the exported runtime.
// Ported from: assemblyscript/src/bindings/js.ts lowerRequiresExportRuntime.
func lowerRequiresExportRuntime(typ *types.Type) bool {
	if !typ.IsInternalReference() {
		return false
	}
	clazz := typ.ClassRef
	if clazz == nil {
		// Function signatures lower by reference, no runtime needed.
		return false
	}
	// A full implementation would check specific class hierarchies.
	// For now, conservatively return true for class references.
	return true
}

// --- Stub methods for compilation phases not yet ported ---

// CompileStatements is now in compile_statement.go
// CompileExpression is now in compile_expression.go

// makeBuiltinFieldGetter makes a built-in getter for a field property.
// Creates a function that loads the field value from the object at the field's memory offset.
// Ported from: assemblyscript/src/compiler.ts makeBuiltinFieldGetter (lines 1874-1920).
func (c *Compiler) makeBuiltinFieldGetter(property *program.Property) module.FunctionRef {
	mod := c.Module()
	getterInstance := property.GetterInstance
	if getterInstance == nil {
		return 0
	}

	fieldType := property.GetType()
	if fieldType == nil {
		return 0
	}
	fieldTypeRef := fieldType.ToRef()

	// The getter takes `this` (i32/i64 pointer) and returns the field value.
	// body: load(this + offset)
	thisTypeRef := module.TypeRefI32
	if c.Options().IsWasm64() {
		thisTypeRef = module.TypeRefI64
	}
	offset := uint32(property.MemoryOffset)

	memName := common.CommonNameDefaultMemory
	var loadExpr module.ExpressionRef
	switch fieldTypeRef {
	case module.TypeRefI32:
		loadExpr = mod.Load(4, true, mod.LocalGet(0, thisTypeRef), fieldTypeRef, offset, 4, memName)
	case module.TypeRefI64:
		loadExpr = mod.Load(8, true, mod.LocalGet(0, thisTypeRef), fieldTypeRef, offset, 8, memName)
	case module.TypeRefF32:
		loadExpr = mod.Load(4, true, mod.LocalGet(0, thisTypeRef), fieldTypeRef, offset, 4, memName)
	case module.TypeRefF64:
		loadExpr = mod.Load(8, true, mod.LocalGet(0, thisTypeRef), fieldTypeRef, offset, 8, memName)
	default:
		// For reference types, load as pointer
		loadExpr = mod.Load(4, false, mod.LocalGet(0, thisTypeRef), thisTypeRef, offset, 4, memName)
	}

	return mod.AddFunction(
		getterInstance.GetInternalName(),
		getterInstance.Signature.ParamRefs(),
		getterInstance.Signature.ResultRefs(),
		nil,
		loadExpr,
	)
}

// makeBuiltinFieldSetter makes a built-in setter for a field property.
// Creates a function that stores a value to the field at the field's memory offset.
// Ported from: assemblyscript/src/compiler.ts makeBuiltinFieldSetter (lines 1922-1960).
func (c *Compiler) makeBuiltinFieldSetter(property *program.Property) module.FunctionRef {
	mod := c.Module()
	setterInstance := property.SetterInstance
	if setterInstance == nil {
		return 0
	}

	fieldType := property.GetType()
	if fieldType == nil {
		return 0
	}
	fieldTypeRef := fieldType.ToRef()

	// The setter takes `this` (i32/i64 pointer) and the value, returns void.
	// body: store(this + offset, value)
	thisTypeRef := module.TypeRefI32
	if c.Options().IsWasm64() {
		thisTypeRef = module.TypeRefI64
	}
	offset := uint32(property.MemoryOffset)

	memName := common.CommonNameDefaultMemory
	var storeExpr module.ExpressionRef
	switch fieldTypeRef {
	case module.TypeRefI32:
		storeExpr = mod.Store(4, mod.LocalGet(0, thisTypeRef), mod.LocalGet(1, fieldTypeRef), fieldTypeRef, offset, 4, memName)
	case module.TypeRefI64:
		storeExpr = mod.Store(8, mod.LocalGet(0, thisTypeRef), mod.LocalGet(1, fieldTypeRef), fieldTypeRef, offset, 8, memName)
	case module.TypeRefF32:
		storeExpr = mod.Store(4, mod.LocalGet(0, thisTypeRef), mod.LocalGet(1, fieldTypeRef), fieldTypeRef, offset, 4, memName)
	case module.TypeRefF64:
		storeExpr = mod.Store(8, mod.LocalGet(0, thisTypeRef), mod.LocalGet(1, fieldTypeRef), fieldTypeRef, offset, 8, memName)
	default:
		storeExpr = mod.Store(4, mod.LocalGet(0, thisTypeRef), mod.LocalGet(1, fieldTypeRef), thisTypeRef, offset, 4, memName)
	}

	return mod.AddFunction(
		setterInstance.GetInternalName(),
		setterInstance.Signature.ParamRefs(),
		setterInstance.Signature.ResultRefs(),
		nil,
		storeExpr,
	)
}

// makeConditionalAllocation creates a conditional this allocation for constructors.
// If `this` is null (not yet allocated by a super call), allocate the object.
// Ported from: assemblyscript/src/compiler.ts makeConditionalAllocation (lines 9847-9882).
func (c *Compiler) makeConditionalAllocation(classInstance *program.Class, thisLocalIndex int32) module.ExpressionRef {
	mod := c.Module()
	thisTypeRef := module.TypeRefI32
	if c.Options().IsWasm64() {
		thisTypeRef = module.TypeRefI64
	}

	// if (!this) this = __new(size, id);
	// The __new builtin allocates an object. For now, emit a call to __new.
	size := int32(classInstance.NextMemoryOffset)
	classId := int32(classInstance.Id())
	allocExpr := mod.Call(
		"~lib/rt/__new",
		[]module.ExpressionRef{
			mod.I32(size),
			mod.I32(classId),
		},
		thisTypeRef,
	)

	return mod.If(
		mod.Unary(module.UnaryOpEqzI32,
			mod.LocalGet(thisLocalIndex, thisTypeRef),
		),
		mod.LocalSet(thisLocalIndex, allocExpr, false),
		0,
	)
}

// makeFieldInitializationInConstructor initializes fields in a constructor.
// Compiles default field initializers and assigns them to `this`.
// Ported from: assemblyscript/src/compiler.ts makeFieldInitializationInConstructor (lines 9884-9961).
func (c *Compiler) makeFieldInitializationInConstructor(classInstance *program.Class, stmts *[]module.ExpressionRef) {
	mod := c.Module()
	thisTypeRef := module.TypeRefI32
	if c.Options().IsWasm64() {
		thisTypeRef = module.TypeRefI64
	}
	memName := common.CommonNameDefaultMemory

	// Iterate instance members and initialize fields
	members := classInstance.GetMembers()
	if members == nil {
		return
	}

	for _, member := range members {
		if member.GetElementKind() != program.ElementKindPropertyPrototype {
			continue
		}
		propProto := member.(*program.PropertyPrototype)
		if !propProto.IsField() {
			continue
		}
		propInstance := propProto.PropertyInstance
		if propInstance == nil {
			continue
		}
		if propInstance.Is(common.CommonFlagsStatic) {
			continue
		}

		fieldType := propInstance.GetType()
		if fieldType == nil {
			continue
		}
		offset := uint32(propInstance.MemoryOffset)

		// Check for initializer
		initNode := propInstance.InitializerNode()
		var initExpr module.ExpressionRef
		if initNode != nil {
			initExpr = c.CompileExpression(initNode, fieldType, ConstraintsConvImplicit)
		} else {
			initExpr = c.makeZeroOfType(fieldType)
		}

		// Store the value: store(this, value, offset)
		fieldTypeRef := fieldType.ToRef()
		var bytes uint32
		switch fieldTypeRef {
		case module.TypeRefI64, module.TypeRefF64:
			bytes = 8
		default:
			bytes = 4
		}
		thisExpr := mod.LocalGet(0, thisTypeRef)
		*stmts = append(*stmts,
			mod.Store(bytes, thisExpr, initExpr, fieldTypeRef, offset, bytes, memName),
		)
	}
}

// ensureConstructor ensures a class has a constructor compiled, creating a default one if needed.
// Ported from: assemblyscript/src/compiler.ts ensureConstructor (lines 10060-10080).
func (c *Compiler) ensureConstructor(classInstance *program.Class, reportNode ast.Node) *program.Function {
	ctorInstance := classInstance.ConstructorInstance
	if ctorInstance != nil {
		c.CompileFunction(ctorInstance)
		return ctorInstance
	}
	// If no explicit constructor, try to resolve the default constructor
	if classInstance.Prototype.ConstructorPrototype != nil {
		resolver := c.Resolver()
		ctorInstance = resolver.ResolveFunction(classInstance.Prototype.ConstructorPrototype, nil, nil, program.ReportModeReport)
		if ctorInstance != nil {
			classInstance.ConstructorInstance = ctorInstance
			c.CompileFunction(ctorInstance)
			return ctorInstance
		}
	}
	return nil
}

// intToString converts an int to its string representation.
func intToString(n int) string {
	return fmt.Sprintf("%d", n)
}

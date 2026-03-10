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
	// TS: this.checkSignatureSupported(instance.signature, (<FunctionDeclaration>declarationNode).signature)
	// Called unconditionally in TS — declarationNode is always FunctionDeclaration or MethodDeclaration
	if fnTypeNode != nil {
		c.checkSignatureSupported(signature, fnTypeNode)
	}
	// Note: fnTypeNode nil guard retained since Go's FunctionTypeNode() may return nil
	// for compiler-generated functions that lack declaration AST nodes

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
// Ported from: assemblyscript/src/compiler.ts makeBuiltinFieldGetter (lines 1874-1899).
func (c *Compiler) makeBuiltinFieldGetter(property *program.Property) module.FunctionRef {
	getterInstance := property.GetterInstance
	if getterInstance == nil {
		return 0
	}
	mod := c.Module()
	valueType := property.GetType()
	if valueType == nil {
		return 0
	}
	valueTypeRef := valueType.ToRef()
	thisTypeRef := c.Options().SizeTypeRef()
	getterInstance.Set(common.CommonFlagsCompiled)
	byteSize := uint32(valueType.ByteSize())
	body := mod.Load(
		byteSize, valueType.IsSignedIntegerValue(),
		mod.LocalGet(0, thisTypeRef),
		valueTypeRef, uint32(property.MemoryOffset), byteSize,
		module.DefaultMemory,
	)
	flowBefore := c.CurrentFlow
	fl := getterInstance.Flow
	c.CurrentFlow = fl
	if property.Is(common.CommonFlagsDefinitelyAssigned) && valueType.IsReference() && !valueType.IsNullableReference() {
		body = c.makeRuntimeNonNullCheck(body, valueType, getterInstance.IdentifierNode())
	}
	c.CurrentFlow = flowBefore
	return mod.AddFunction(
		getterInstance.GetInternalName(),
		thisTypeRef,
		valueTypeRef,
		typesToRefs(getterInstance.GetNonParameterLocalTypes()),
		body,
	)
}

// makeBuiltinFieldSetter makes a built-in setter for a field property.
// Creates a function that stores a value to the field at the field's memory offset.
// Ported from: assemblyscript/src/compiler.ts makeBuiltinFieldSetter (lines 1902-1938).
func (c *Compiler) makeBuiltinFieldSetter(property *program.Property) module.FunctionRef {
	setterInstance := property.SetterInstance
	if setterInstance == nil {
		return 0
	}
	mod := c.Module()
	valueType := property.GetType()
	if valueType == nil {
		return 0
	}
	thisTypeRef := c.Options().SizeTypeRef()
	valueTypeRef := valueType.ToRef()
	// void(this.field = value)
	byteSize := uint32(valueType.ByteSize())
	bodyExpr := mod.Store(
		byteSize,
		mod.LocalGet(0, thisTypeRef),
		mod.LocalGet(1, valueTypeRef),
		valueTypeRef, uint32(property.MemoryOffset), byteSize,
		module.DefaultMemory,
	)
	if valueType.IsManaged() {
		parent := setterInstance.GetParent()
		if parent.GetElementKind() == program.ElementKindClass {
			parentClass := parent.(*program.Class)
			if parentClass.GetType().IsManaged() {
				linkInstance := c.Program.LinkInstance()
				c.CompileFunction(linkInstance)
				bodyExpr = mod.Block("", []module.ExpressionRef{
					bodyExpr,
					mod.Call(linkInstance.GetInternalName(), []module.ExpressionRef{
						mod.LocalGet(0, thisTypeRef),
						mod.LocalGet(1, valueTypeRef),
						mod.I32(0),
					}, module.TypeRefNone),
				}, module.TypeRefNone)
			}
		}
	}
	setterInstance.Set(common.CommonFlagsCompiled)
	return mod.AddFunction(
		setterInstance.GetInternalName(),
		module.CreateType([]module.TypeRef{thisTypeRef, valueTypeRef}),
		module.TypeRefNone,
		nil,
		bodyExpr,
	)
}

// makeAllocation makes an allocation suitable to hold the data of an instance of the given class.
// For @unmanaged classes, calls __alloc(size). For managed classes, calls __new(size, classId).
// Ported from: assemblyscript/src/compiler.ts makeAllocation (lines 10351-10377).
func (c *Compiler) makeAllocation(classInstance *program.Class) module.ExpressionRef {
	prog := c.Program
	mod := c.Module()
	options := c.Options()
	c.CurrentType = classInstance.GetType()
	if classInstance.HasDecorator(program.DecoratorFlagsUnmanaged) {
		allocInstance := prog.AllocInstance()
		c.CompileFunction(allocInstance)
		sizeArg := module.ExpressionRef(0)
		if options.IsWasm64() {
			sizeArg = mod.I64(int64(classInstance.NextMemoryOffset))
		} else {
			sizeArg = mod.I32(int32(classInstance.NextMemoryOffset))
		}
		return mod.Call(allocInstance.GetInternalName(), []module.ExpressionRef{
			sizeArg,
		}, options.SizeTypeRef())
	} else {
		newInstance := prog.NewInstance()
		c.CompileFunction(newInstance)
		sizeArg := module.ExpressionRef(0)
		if options.IsWasm64() {
			sizeArg = mod.I64(int64(classInstance.NextMemoryOffset))
		} else {
			sizeArg = mod.I32(int32(classInstance.NextMemoryOffset))
		}
		return mod.Call(newInstance.GetInternalName(), []module.ExpressionRef{
			sizeArg,
			mod.I32(int32(classInstance.Id())),
		}, options.SizeTypeRef())
	}
}

// makeConditionalAllocation creates a conditional this allocation for constructors.
// If `this` is null (not yet allocated by a super call), allocate the object.
// Ported from: assemblyscript/src/compiler.ts makeConditionalAllocation (lines 9847-9882).
func (c *Compiler) makeConditionalAllocation(classInstance *program.Class, thisLocalIndex int32) module.ExpressionRef {
	mod := c.Module()
	classType := classInstance.GetType()
	classTypeRef := classType.ToRef()

	eqzOp := module.UnaryOpEqzI32
	if classTypeRef == module.TypeRefI64 {
		eqzOp = module.UnaryOpEqzI64
	}

	return mod.If(
		mod.Unary(eqzOp,
			mod.LocalGet(thisLocalIndex, classTypeRef),
		),
		mod.LocalSet(thisLocalIndex,
			c.makeAllocation(classInstance),
			classInstance.GetType().IsManaged(),
		),
		0,
	)
}

// makeFieldInitializationInConstructor initializes fields in a constructor.
// Compiles default field initializers and assigns them to `this`.
// Ported from: assemblyscript/src/compiler.ts makeFieldInitializationInConstructor (lines 9884-9961).
func (c *Compiler) makeFieldInitializationInConstructor(classInstance *program.Class, stmts *[]module.ExpressionRef) {
	members := classInstance.OrderedOwnMembers()
	if len(members) == 0 {
		return
	}

	mod := c.Module()
	fl := c.CurrentFlow
	isInline := fl != nil && fl.IsInline()
	thisLocalIndex := int32(0)
	if isInline {
		if thisLocal := fl.LookupLocal(common.CommonNameThis); thisLocal != nil {
			thisLocalIndex = thisLocal.FlowIndex()
		}
	}
	sizeTypeRef := c.Options().SizeTypeRef()
	nonParameterFields := make([]*program.Property, 0)

	for _, member := range members {
		propertyPrototype, ok := member.(*program.PropertyPrototype)
		if !ok {
			continue
		}

		property := propertyPrototype.PropertyInstance
		if property == nil || !property.IsField() || property.GetBoundClassOrInterface() != classInstance {
			continue
		}
		if property.IsAny(common.CommonFlagsConst) {
			panic("const fields must not be initialized in the constructor")
		}

		parameterIndex := propertyPrototype.ParameterIndex()
		if parameterIndex < 0 {
			nonParameterFields = append(nonParameterFields, property)
			continue
		}

		setterInstance := property.SetterInstance
		if setterInstance == nil {
			continue
		}

		fieldType := property.GetResolvedType()
		if fieldType == nil {
			continue
		}

		parameterLocalIndex := int32(1 + parameterIndex)
		if isInline {
			if parameterLocal := fl.LookupLocal(property.GetName()); parameterLocal != nil {
				parameterLocalIndex = parameterLocal.FlowIndex()
			}
		}

		var reportNode ast.Node = property.IdentifierNode()
		if reportNode == nil {
			reportNode = property.GetDeclaration()
		}

		expr := c.makeCallDirect(setterInstance, []module.ExpressionRef{
			mod.LocalGet(thisLocalIndex, sizeTypeRef),
			mod.LocalGet(parameterLocalIndex, fieldType.ToRef()),
		}, reportNode, true)
		if c.CurrentType != types.TypeVoid {
			expr = mod.Drop(expr)
		}
		*stmts = append(*stmts, expr)
	}

	for _, field := range nonParameterFields {
		fieldType := field.GetResolvedType()
		if fieldType == nil {
			continue
		}

		setterInstance := field.SetterInstance
		if setterInstance == nil {
			continue
		}

		initializerNode := field.Prototype.PropertyInitializerNode()
		fieldValue := c.makeZeroOfType(fieldType)
		if initializerNode != nil {
			fieldValue = c.CompileExpression(initializerNode, fieldType, ConstraintsConvImplicit)
		}

		var reportNode ast.Node = field.IdentifierNode()
		if reportNode == nil {
			reportNode = field.GetDeclaration()
		}

		expr := c.makeCallDirect(setterInstance, []module.ExpressionRef{
			mod.LocalGet(thisLocalIndex, sizeTypeRef),
			fieldValue,
		}, reportNode, true)
		if c.CurrentType != types.TypeVoid {
			expr = mod.Drop(expr)
		}
		*stmts = append(*stmts, expr)
	}

	c.CurrentType = types.TypeVoid
}

// ensureConstructor ensures a class has a constructor compiled, creating a default one if needed.
// Ported from: assemblyscript/src/compiler.ts ensureConstructor (lines 8868-8995).
func (c *Compiler) ensureConstructor(classInstance *program.Class, reportNode ast.Node) *program.Function {
	instance := classInstance.ConstructorInstance
	if instance != nil {
		// shortcut if already compiled
		if instance.Is(common.CommonFlagsCompiled) {
			return instance
		}
		// do not attempt to compile if inlined anyway
		if !instance.HasDecorator(program.DecoratorFlagsInline) {
			c.CompileFunction(instance)
		}
	} else {
		// clone base constructor if a derived class. note that we cannot just
		// call the base ctor since the derived class may have additional fields.
		baseClass := classInstance.Base
		contextualTypeArguments := cloneTypeArgMap(classInstance.ContextualTypeArguments)
		if baseClass != nil {
			baseCtor := c.ensureConstructor(baseClass, reportNode)
			c.checkFieldInitialization(baseClass, reportNode)
			baseCtorDecl := baseCtor.GetDeclaration().(*ast.FunctionDeclaration).Clone()
			instance = program.NewFunction(
				common.CommonNameConstructor,
				program.NewFunctionPrototype(
					common.CommonNameConstructor,
					classInstance,
					// declaration is important, i.e. to access optional parameter initializers
					baseCtorDecl,
					program.DecoratorFlagsNone,
				),
				nil, // typeArguments
				types.CreateSignature(
					classInstance.GetProgram(),
					baseCtor.Signature.ParameterTypes,
					classInstance.GetType(),
					classInstance.GetType(),
					baseCtor.Signature.RequiredParameters,
					baseCtor.Signature.HasRest,
				),
				contextualTypeArguments,
			)

		// otherwise make a default constructor
		} else {
			instance = program.NewFunction(
				common.CommonNameConstructor,
				program.NewFunctionPrototype(
					common.CommonNameConstructor,
					classInstance, // bound
					c.Program.MakeNativeFunctionDeclaration(common.CommonNameConstructor,
						common.CommonFlagsInstance|common.CommonFlagsConstructor,
					),
					program.DecoratorFlagsNone,
				),
				nil, // typeArguments
				types.CreateSignature(classInstance.GetProgram(), nil, classInstance.GetType(), classInstance.GetType(), -1, false),
				contextualTypeArguments,
			)
		}

		instance.Set(common.CommonFlagsCompiled)
		instance.Prototype.SetResolvedInstance("", instance)
		if classInstance.Is(common.CommonFlagsModuleExport) {
			instance.Set(common.CommonFlagsModuleExport)
		}
		classInstance.ConstructorInstance = instance
		members := classInstance.GetMembers()
		if members == nil {
			members = make(map[string]program.DeclaredElement)
			classInstance.SetMembers(members)
		}
		members[common.CommonNameConstructor] = instance.Prototype

		previousFlow := c.CurrentFlow
		fl := instance.Flow
		c.CurrentFlow = fl

		// generate body
		signature := instance.Signature
		mod := c.Module()
		sizeTypeRef := module.TypeRef(c.Options().SizeTypeRef())
		stmts := make([]module.ExpressionRef, 0)

		// {
		//   this = <COND_ALLOC>
		//   IF_DERIVED: this = super(this, ...args)
		//   this.a = X
		//   this.b = Y
		//   return this
		// }
		stmts = append(stmts,
			c.makeConditionalAllocation(classInstance, 0),
		)
		if baseClass != nil {
			parameterTypes := signature.ParameterTypes
			numParameters := len(parameterTypes)
			operands := make([]module.ExpressionRef, 1+numParameters)
			operands[0] = mod.LocalGet(0, sizeTypeRef)
			for i := 1; i <= numParameters; i++ {
				operands[i] = mod.LocalGet(int32(i), parameterTypes[i-1].ToRef())
			}
			baseCtorInstance := baseClass.ConstructorInstance
			if baseCtorInstance == nil {
				panic("ensureConstructor: baseClass.ConstructorInstance is nil after ensureConstructor")
			}
			stmts = append(stmts,
				mod.LocalSet(0,
					c.makeCallDirect(baseCtorInstance, operands, reportNode, false),
					baseClass.GetType().IsManaged(),
				),
			)
		}
		c.makeFieldInitializationInConstructor(classInstance, &stmts)
		stmts = append(stmts,
			mod.LocalGet(0, sizeTypeRef),
		)
		c.CurrentFlow = previousFlow

		// make the function
		locals := instance.LocalsByIndex
		varTypes := make([]module.TypeRef, 0) // of temp. vars added while compiling initializers
		numOperands := 1 + len(signature.ParameterTypes)
		numLocals := len(locals)
		if numLocals > numOperands {
			for i := numOperands; i < numLocals; i++ {
				varTypes = append(varTypes, locals[i].GetType().ToRef())
			}
		}
		funcRef := mod.AddFunction(
			instance.GetInternalName(),
			signature.ParamRefs(),
			signature.ResultRefs(),
			varTypes,
			mod.Flatten(stmts, sizeTypeRef),
		)
		instance.Finalize(c.Module(), funcRef)
	}

	return instance
}

// cloneTypeArgMap shallow-clones a map of contextual type arguments.
// Ported from: assemblyscript/src/util/collections.ts cloneMap.
func cloneTypeArgMap(src map[string]*types.Type) map[string]*types.Type {
	if src == nil {
		return nil
	}
	dst := make(map[string]*types.Type, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// intToString converts an int to its string representation.
func intToString(n int) string {
	return fmt.Sprintf("%d", n)
}

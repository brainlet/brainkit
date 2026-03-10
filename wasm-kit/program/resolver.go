package program

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/types"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// ReportMode indicates whether errors are reported or swallowed.
type ReportMode int32

const (
	// ReportModeReport reports errors.
	ReportModeReport ReportMode = iota
	// ReportModeSwallow swallows errors silently.
	ReportModeSwallow
)

// Resolver provides tools to resolve types and expressions.
// It is a 1:1 port of the TypeScript Resolver class.
type Resolver struct {
	diagnostics.DiagnosticEmitter

	// program is the program this resolver belongs to.
	program *Program

	// CurrentThisExpression is the target expression of the previously resolved
	// property or element access.
	CurrentThisExpression ast.Node

	// CurrentElementExpression is the element expression of the previously
	// resolved element access.
	CurrentElementExpression ast.Node

	// DiscoveredOverride indicates whether a new override has been discovered.
	DiscoveredOverride bool

	// resolveClassPending tracks classes currently being resolved to detect cycles.
	resolveClassPending map[*Class]struct{}
}

// NewResolver creates a new resolver for the given program.
func NewResolver(program *Program) *Resolver {
	r := &Resolver{
		program:             program,
		resolveClassPending: make(map[*Class]struct{}),
	}
	r.DiagnosticEmitter = diagnostics.NewDiagnosticEmitter(program.Diagnostics)
	return r
}

// GetProgram returns the program this resolver belongs to.
func (r *Resolver) GetProgram() *Program {
	return r.program
}

// =========================================================================
// Type resolution
// =========================================================================

// ResolveType resolves a TypeNode to a concrete Type.
func (r *Resolver) ResolveType(
	node ast.Node,
	f *flow.Flow,
	ctxElement Element,
	ctxTypes map[string]*types.Type,
	reportMode ReportMode,
) *types.Type {
	// Type-switch to access CurrentlyResolving on concrete types
	switch n := node.(type) {
	case *ast.NamedTypeNode:
		if n.CurrentlyResolving {
			if reportMode == ReportModeReport {
				r.program.Error(
					diagnostics.DiagnosticCodeNotImplemented0,
					node.GetRange(),
					"Recursive types",
				)
			}
			return nil
		}
		n.CurrentlyResolving = true
		resolved := r.resolveNamedType(n, f, ctxElement, ctxTypes, reportMode)
		n.CurrentlyResolving = false
		return resolved
	case *ast.FunctionTypeNode:
		if n.CurrentlyResolving {
			if reportMode == ReportModeReport {
				r.program.Error(
					diagnostics.DiagnosticCodeNotImplemented0,
					node.GetRange(),
					"Recursive types",
				)
			}
			return nil
		}
		n.CurrentlyResolving = true
		resolved := r.resolveFunctionType(n, f, ctxElement, ctxTypes, reportMode)
		n.CurrentlyResolving = false
		return resolved
	default:
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeNotImplemented0,
				node.GetRange(),
				"Unsupported type node",
			)
		}
		return nil
	}
}

// resolveNamedType resolves a NamedTypeNode to a concrete Type.
func (r *Resolver) resolveNamedType(
	node *ast.NamedTypeNode,
	f *flow.Flow,
	ctxElement Element,
	ctxTypes map[string]*types.Type,
	reportMode ReportMode,
) *types.Type {
	// Look up by type name
	element := r.ResolveTypeName(node.Name, f, ctxElement, reportMode)
	if element == nil {
		return nil
	}

	switch element.GetElementKind() {
	case ElementKindClassPrototype:
		prototype := element.(*ClassPrototype)
		// Resolve type arguments if generic
		if prototype.Is(common.CommonFlagsGeneric) {
			var resolvedTypeArguments []*types.Type
			typeParameterNodes := prototype.TypeParameterNodes()
			if typeParameterNodes != nil {
				resolvedTypeArguments = r.ResolveTypeArguments(
					typeParameterNodes,
					node.TypeArguments,
					f,
					ctxElement,
					ctxTypes,
					node,
					reportMode,
				)
				if resolvedTypeArguments == nil {
					return nil
				}
			}
			instance := r.ResolveClass(prototype, resolvedTypeArguments, ctxTypes, reportMode)
			if instance == nil {
				return nil
			}
			return instance.resolvedType
		}
		// Non-generic class
		instance := r.ResolveClass(prototype, nil, ctxTypes, reportMode)
		if instance == nil {
			return nil
		}
		return instance.resolvedType

	case ElementKindTypeDefinition:
		td := element.(*TypeDefinition)
		if td.GetResolvedType() != nil && td.GetResolvedType() != types.TypeVoid {
			return td.GetResolvedType()
		}
		// Resolve the type definition's type node
		typeNode := td.TypeNode()
		if typeNode == nil {
			return nil
		}
		resolved := r.ResolveType(typeNode, f, td.GetParent(), ctxTypes, reportMode)
		if resolved != nil {
			td.SetType(resolved)
		}
		return resolved

	default:
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeTypeExpected,
				node.GetRange(),
			)
		}
		return nil
	}
}

// resolveFunctionType resolves a FunctionTypeNode to a concrete Type.
func (r *Resolver) resolveFunctionType(
	node *ast.FunctionTypeNode,
	f *flow.Flow,
	ctxElement Element,
	ctxTypes map[string]*types.Type,
	reportMode ReportMode,
) *types.Type {
	params := node.Parameters
	numParams := len(params)
	parameterTypes := make([]*types.Type, numParams)
	requiredParameters := int32(0)
	hasRest := false

	for i := 0; i < numParams; i++ {
		param := params[i]
		if param.ParameterKind == ast.ParameterKindDefault {
			requiredParameters = int32(i + 1)
		} else if param.ParameterKind == ast.ParameterKindRest {
			hasRest = true
		}
		paramType := r.ResolveType(param.Type, f, ctxElement, ctxTypes, reportMode)
		if paramType == nil {
			return nil
		}
		parameterTypes[i] = paramType
	}

	returnType := r.ResolveType(node.ReturnType, f, ctxElement, ctxTypes, reportMode)
	if returnType == nil {
		return nil
	}

	// Resolve explicit this type if present
	var thisType *types.Type
	if node.ExplicitThisType != nil {
		thisType = r.ResolveType(node.ExplicitThisType, f, ctxElement, ctxTypes, reportMode)
		if thisType == nil {
			return nil
		}
	}

	sig := types.CreateSignature(r.program, parameterTypes, returnType, thisType, requiredParameters, hasRest)
	return sig.Type
}

// ResolveTypeName resolves a TypeName to the element it refers to.
func (r *Resolver) ResolveTypeName(
	node *ast.TypeName,
	f *flow.Flow,
	ctxElement Element,
	reportMode ReportMode,
) Element {
	var element Element

	// Check flow-scoped type aliases first
	if f != nil {
		alias := f.LookupScopedTypeAlias(node.Identifier.Text)
		if alias != nil {
			if elem, ok := alias.(Element); ok {
				element = elem
			}
		}
	}

	// Look up in context element
	if element == nil {
		element = ctxElement.Lookup(node.Identifier.Text, true)
	}

	if element == nil {
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeCannotFindName0,
				node.GetRange(),
				node.Identifier.Text,
			)
		}
		return nil
	}

	// Walk the dotted path: Foo.Bar.Baz
	next := node.Next
	for next != nil {
		member := element.GetMember(next.Identifier.Text)
		if member == nil {
			if reportMode == ReportModeReport {
				r.program.Error(
					diagnostics.DiagnosticCodeProperty0DoesNotExistOnType1,
					next.GetRange(),
					next.Identifier.Text,
					node.Identifier.Text,
				)
			}
			return nil
		}
		element = member
		next = next.Next
	}

	return element
}

// ResolveTypeArguments resolves an array of type arguments to concrete types.
func (r *Resolver) ResolveTypeArguments(
	typeParameters []*ast.TypeParameterNode,
	typeArgumentNodes []ast.Node,
	f *flow.Flow,
	ctxElement Element,
	ctxTypes map[string]*types.Type,
	alternativeReportNode ast.Node,
	reportMode ReportMode,
) []*types.Type {
	// Count min/max type parameters
	minParameterCount := 0
	maxParameterCount := len(typeParameters)
	for _, tp := range typeParameters {
		if tp.DefaultType == nil {
			minParameterCount++
		}
	}

	argumentCount := len(typeArgumentNodes)
	if argumentCount < minParameterCount || argumentCount > maxParameterCount {
		if reportMode == ReportModeReport {
			var rng *diagnostics.Range
			if argumentCount > 0 {
				rng = typeArgumentNodes[0].GetRange()
			} else if alternativeReportNode != nil {
				rng = alternativeReportNode.GetRange()
			}
			r.program.Error(
				diagnostics.DiagnosticCodeExpected0TypeArgumentsButGot1,
				rng,
				fmt.Sprintf("%d", func() int {
					if argumentCount < minParameterCount {
						return minParameterCount
					}
					return maxParameterCount
				}()),
				fmt.Sprintf("%d", argumentCount),
			)
		}
		return nil
	}

	// Save old contextual types
	oldCtxTypes := make(map[string]*types.Type, len(ctxTypes))
	for k, v := range ctxTypes {
		oldCtxTypes[k] = v
	}
	// Clear for fresh resolution
	for k := range ctxTypes {
		delete(ctxTypes, k)
	}

	typeArguments := make([]*types.Type, maxParameterCount)
	for i := 0; i < maxParameterCount; i++ {
		var typ *types.Type
		if i < argumentCount {
			typ = r.ResolveType(typeArgumentNodes[i], f, ctxElement, oldCtxTypes, reportMode)
		} else {
			// Use default type
			defaultType := typeParameters[i].DefaultType
			if defaultType == nil {
				return nil
			}
			// Clone ctxTypes for default resolution
			clonedCtx := make(map[string]*types.Type, len(ctxTypes))
			for k, v := range ctxTypes {
				clonedCtx[k] = v
			}
			typ = r.ResolveType(defaultType, f, ctxElement, clonedCtx, reportMode)
		}
		if typ == nil {
			return nil
		}
		ctxTypes[typeParameters[i].Name.Text] = typ
		typeArguments[i] = typ
	}

	return typeArguments
}

// =========================================================================
// Function resolution
// =========================================================================

// ResolveFunction resolves a function prototype to a concrete Function instance.
func (r *Resolver) ResolveFunction(
	prototype *FunctionPrototype,
	typeArguments []*types.Type,
	ctxTypes map[string]*types.Type,
	reportMode ReportMode,
) *Function {
	var classInstance *Class
	instanceKey := ""
	if typeArguments != nil {
		instanceKey = types.TypesToString(typeArguments)
	}

	if ctxTypes == nil {
		ctxTypes = make(map[string]*types.Type)
	}

	// Instance method prototypes are pre-bound to their concrete class
	if prototype.Is(common.CommonFlagsInstance) {
		classInstance = prototype.GetBoundClassOrInterface()
		if classInstance == nil {
			return nil
		}

		// Check if already resolved
		resolved := prototype.GetResolvedInstance(instanceKey)
		if resolved != nil {
			return resolved
		}

		// Inherit class-specific type arguments
		classTypeArguments := classInstance.TypeArguments
		if classTypeArguments != nil {
			typeParameterNodes := classInstance.Prototype.TypeParameterNodes()
			if typeParameterNodes != nil {
				for i, tp := range typeParameterNodes {
					if i < len(classTypeArguments) {
						ctxTypes[tp.Name.Text] = classTypeArguments[i]
					}
				}
			}
		}
	} else {
		resolved := prototype.GetResolvedInstance(instanceKey)
		if resolved != nil {
			return resolved
		}
	}

	// Override contextual types with actual function type arguments
	signatureNode := prototype.FunctionTypeNode()
	typeParameterNodes := prototype.TypeParameterNodes()
	if typeArguments != nil && len(typeArguments) > 0 {
		if typeParameterNodes == nil || len(typeArguments) != len(typeParameterNodes) {
			return nil
		}
		for i, tp := range typeParameterNodes {
			ctxTypes[tp.Name.Text] = typeArguments[i]
		}
	}

	// Resolve `this` type if applicable
	var thisType *types.Type
	if signatureNode != nil && signatureNode.ExplicitThisType != nil {
		thisType = r.ResolveType(
			signatureNode.ExplicitThisType,
			nil,
			prototype.GetParent(),
			ctxTypes,
			reportMode,
		)
		if thisType == nil {
			return nil
		}
		ctxTypes[common.CommonNameThis] = thisType
	} else if classInstance != nil {
		thisType = classInstance.resolvedType
		ctxTypes[common.CommonNameThis] = thisType
	}

	// Resolve parameter types
	var parameterTypes []*types.Type
	requiredParameters := int32(0)
	hasRest := false
	if signatureNode != nil {
		params := signatureNode.Parameters
		parameterTypes = make([]*types.Type, len(params))
		for i, param := range params {
			if param.ParameterKind == ast.ParameterKindDefault {
				requiredParameters = int32(i + 1)
			} else if param.ParameterKind == ast.ParameterKindRest {
				hasRest = true
			}
			typeNode := param.Type
			if ast.IsTypeOmitted(typeNode) {
				if reportMode == ReportModeReport {
					r.program.Error(
						diagnostics.DiagnosticCodeTypeExpected,
						typeNode.GetRange(),
					)
				}
				return nil
			}
			paramType := r.ResolveType(typeNode, nil, prototype.GetParent(), ctxTypes, reportMode)
			if paramType == nil {
				return nil
			}
			if paramType == types.TypeVoid {
				if reportMode == ReportModeReport {
					r.program.Error(
						diagnostics.DiagnosticCodeTypeExpected,
						typeNode.GetRange(),
					)
				}
				return nil
			}
			parameterTypes[i] = paramType
		}
	}

	// Resolve return type
	var returnType *types.Type
	if prototype.Is(common.CommonFlagsSet) {
		returnType = types.TypeVoid
	} else if prototype.Is(common.CommonFlagsConstructor) {
		if classInstance != nil {
			returnType = classInstance.resolvedType
		} else {
			returnType = types.TypeVoid
		}
	} else if signatureNode != nil {
		retTypeNode := signatureNode.ReturnType
		if ast.IsTypeOmitted(retTypeNode) {
			if reportMode == ReportModeReport {
				r.program.Error(
					diagnostics.DiagnosticCodeTypeExpected,
					retTypeNode.GetRange(),
				)
			}
			return nil
		}
		resolved := r.ResolveType(retTypeNode, nil, prototype.GetParent(), ctxTypes, reportMode)
		if resolved == nil {
			return nil
		}
		returnType = resolved
	} else {
		returnType = types.TypeVoid
	}

	signature := types.CreateSignature(r.program, parameterTypes, returnType, thisType, requiredParameters, hasRest)

	nameInclTypeParams := prototype.GetName()
	if instanceKey != "" {
		nameInclTypeParams += "<" + instanceKey + ">"
	}

	instance := NewFunction(
		nameInclTypeParams,
		prototype,
		typeArguments,
		signature,
		ctxTypes,
	)
	prototype.SetResolvedInstance(instanceKey, instance)

	// Check against overridden base member
	if classInstance != nil {
		r.checkOverrideCompatibility(instance, classInstance, typeArguments, reportMode)
	}

	return instance
}

// checkOverrideCompatibility verifies override compatibility with base class.
func (r *Resolver) checkOverrideCompatibility(
	instance *Function,
	classInstance *Class,
	typeArguments []*types.Type,
	reportMode ReportMode,
) {
	decl := instance.GetDeclaration()
	if decl == nil {
		return
	}
	funcDecl, ok := decl.(*ast.FunctionDeclaration)
	if !ok || funcDecl.Name == nil {
		return
	}
	methodName := funcDecl.Name.Text

	baseClass := classInstance.Base
	if baseClass == nil {
		return
	}

	baseMember := baseClass.GetMember(methodName)
	if baseMember == nil {
		return
	}

	// Note override discovery
	r.DiscoveredOverride = true

	incompatibleOverride := true
	if instance.IsAny(common.CommonFlagsGet | common.CommonFlagsSet) {
		if baseMember.GetElementKind() == ElementKindPropertyPrototype {
			pp := baseMember.(*PropertyPrototype)
			baseProperty := r.ResolveProperty(pp, reportMode)
			if baseProperty != nil {
				if instance.Is(common.CommonFlagsGet) {
					if baseProperty.GetterInstance != nil {
						if instance.Signature.IsAssignableTo(baseProperty.GetterInstance.Signature, true) {
							incompatibleOverride = false
						}
					}
				} else {
					if baseProperty.SetterInstance != nil {
						if instance.Signature.IsAssignableTo(baseProperty.SetterInstance.Signature, true) {
							incompatibleOverride = false
						}
					}
				}
			}
		}
	} else if instance.Is(common.CommonFlagsConstructor) {
		incompatibleOverride = false
	} else {
		if baseMember.GetElementKind() == ElementKindFunctionPrototype {
			basePrototype := baseMember.(*FunctionPrototype)
			baseFunction := r.ResolveFunction(basePrototype, typeArguments, make(map[string]*types.Type), ReportModeSwallow)
			if baseFunction != nil && instance.Signature.IsAssignableTo(baseFunction.Signature, true) {
				incompatibleOverride = false
			}
		}
	}

	if incompatibleOverride {
		if reportMode == ReportModeReport {
			ident := instance.IdentifierNode()
			baseIdent := baseMember.(DeclaredElement).IdentifierNode()
			if ident != nil && baseIdent != nil {
				r.program.ErrorRelated(
					diagnostics.DiagnosticCodeThisOverloadSignatureIsNotCompatibleWithItsImplementationSignature,
					ident.GetRange(),
					baseIdent.GetRange(),
				)
			}
		}
	}
}

// ResolveFunctionInclTypeArguments resolves a function prototype by first resolving type argument nodes.
func (r *Resolver) ResolveFunctionInclTypeArguments(
	prototype *FunctionPrototype,
	typeArgumentNodes []ast.Node,
	ctxElement Element,
	ctxTypes map[string]*types.Type,
	reportNode ast.Node,
	reportMode ReportMode,
) *Function {
	var resolvedTypeArguments []*types.Type
	if ctxTypes == nil {
		ctxTypes = make(map[string]*types.Type)
	}

	if prototype.Is(common.CommonFlagsGeneric) {
		if prototype.Is(common.CommonFlagsInstance) {
			classInstance := prototype.GetBoundClassOrInterface()
			if classInstance != nil && classInstance.TypeArguments != nil {
				typeParameterNodes := classInstance.Prototype.TypeParameterNodes()
				for i, typeArgument := range classInstance.TypeArguments {
					if i < len(typeParameterNodes) {
						ctxTypes[typeParameterNodes[i].Name.Text] = typeArgument
					}
				}
			}
		}

		resolvedTypeArguments = r.ResolveTypeArguments(
			prototype.TypeParameterNodes(),
			typeArgumentNodes,
			nil,
			ctxElement,
			ctxTypes,
			reportNode,
			reportMode,
		)
		if resolvedTypeArguments == nil {
			return nil
		}
	} else if len(typeArgumentNodes) > 0 {
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeType0IsNotGeneric,
				reportNode.GetRange(),
				prototype.GetInternalName(),
				"", "",
			)
		}
		return nil
	}

	return r.ResolveFunction(prototype, resolvedTypeArguments, ctxTypes, reportMode)
}

// MaybeInferCall resolves a function prototype from a call expression, handling
// explicit type arguments, type inference, and non-generic cases.
// Ported from: assemblyscript/src/resolver.ts maybeInferCall (lines 573-624).
func (r *Resolver) MaybeInferCall(
	node *ast.CallExpression,
	prototype *FunctionPrototype,
	ctxFlow *flow.Flow,
	reportMode ReportMode,
) *Function {
	typeArguments := node.TypeArguments

	// resolve generic call if type arguments have been provided
	if typeArguments != nil {
		if !prototype.Is(common.CommonFlagsGeneric) {
			if reportMode == ReportModeReport {
				r.program.Error(
					diagnostics.DiagnosticCodeType0IsNotGeneric,
					node.Expression.GetRange(),
					prototype.GetInternalName(), "", "",
				)
			}
			return nil
		}
		return r.ResolveFunctionInclTypeArguments(
			prototype,
			typeArguments,
			ctxFlow.SourceFunction().(Element),
			util.CloneMap(ctxFlow.ContextualTypeArguments()),
			node,
			reportMode,
		)
	}

	// infer generic call if type arguments have been omitted
	if prototype.Is(common.CommonFlagsGeneric) {
		resolvedTypeArguments := r.InferGenericTypeArguments(
			node,
			prototype,
			prototype.TypeParameterNodes(),
			ctxFlow,
			reportMode,
		)
		if resolvedTypeArguments == nil {
			return nil
		}
		return r.ResolveFunction(
			prototype,
			resolvedTypeArguments,
			util.CloneMap(ctxFlow.ContextualTypeArguments()),
			reportMode,
		)
	}

	// otherwise resolve the non-generic call as usual
	return r.ResolveFunction(prototype, nil, make(map[string]*types.Type), reportMode)
}

// InferGenericTypeArguments attempts to infer generic type arguments from a call expression.
// Ported from: assemblyscript/src/resolver.ts inferGenericTypeArguments (lines 626-780).
func (r *Resolver) InferGenericTypeArguments(
	node ast.Node,
	prototype *FunctionPrototype,
	typeParameterNodes []*ast.TypeParameterNode,
	ctxFlow *flow.Flow,
	reportMode ReportMode,
) []*types.Type {
	if typeParameterNodes == nil {
		return nil
	}

	contextualTypeArguments := util.CloneMap(ctxFlow.ContextualTypeArguments())
	if contextualTypeArguments == nil {
		contextualTypeArguments = make(map[string]*types.Type)
	}

	// fill up contextual types with auto for each generic component
	numTypeParameters := len(typeParameterNodes)
	typeParameterNames := make(map[string]struct{}, numTypeParameters)
	for i := 0; i < numTypeParameters; i++ {
		name := typeParameterNodes[i].Name.Text
		contextualTypeArguments[name] = types.TypeAuto
		typeParameterNames[name] = struct{}{}
	}

	parameterNodes := prototype.FunctionTypeNode().Parameters
	numParameters := len(parameterNodes)

	var argumentNodes []ast.Node
	var argumentsRange diagnostics.Range
	switch expr := node.(type) {
	case *ast.CallExpression:
		argumentNodes = expr.Args
		argumentsRange = expr.ArgumentsRange()
	case *ast.NewExpression:
		argumentNodes = expr.Args
		argumentsRange = expr.ArgumentsRange()
	default:
		return nil
	}
	numArguments := len(argumentNodes)

	for i := 0; i < numParameters; i++ {
		var argumentExpression ast.Node
		if i < numArguments {
			argumentExpression = argumentNodes[i]
		} else {
			argumentExpression = parameterNodes[i].Initializer
		}
		if argumentExpression == nil {
			if parameterNodes[i].ParameterKind == ast.ParameterKindOptional {
				continue
			}
			if reportMode == ReportModeReport {
				if parameterNodes[i].ParameterKind == ast.ParameterKindRest {
					r.program.Error(
						diagnostics.DiagnosticCodeTypeArgumentExpected,
						argumentsRange.AtEnd(),
						"", "", "",
					)
				} else {
					r.program.Error(
						diagnostics.DiagnosticCodeExpected0ArgumentsButGot1,
						node.GetRange(),
						fmt.Sprintf("%d", numParameters),
						fmt.Sprintf("%d", numArguments),
						"",
					)
				}
			}
			return nil
		}

		typeNode := parameterNodes[i].Type
		if parameterNodes[i].ParameterKind == ast.ParameterKindRest {
			if namedType, ok := typeNode.(*ast.NamedTypeNode); ok && len(namedType.TypeArguments) == 1 {
				typeNode = namedType.TypeArguments[0]
			}
		}
		if ast.HasGenericComponent(typeNode, typeParameterNodes) {
			inferredType := r.ResolveExpression(argumentExpression, ctxFlow, types.TypeAuto, ReportModeSwallow)
			if inferredType != nil {
				r.propagateInferredGenericTypes(typeNode, inferredType, prototype, contextualTypeArguments, typeParameterNames)
			}
		}
	}

	result := make([]*types.Type, numTypeParameters)
	for i := 0; i < numTypeParameters; i++ {
		typeParameterNode := typeParameterNodes[i]
		name := typeParameterNode.Name.Text
		if inferredType, ok := contextualTypeArguments[name]; ok && inferredType != types.TypeAuto {
			result[i] = inferredType
			continue
		}

		defaultType := typeParameterNode.DefaultType
		if defaultType != nil {
			var defaultTypeContextualTypeArguments map[string]*types.Type
			switch parent := prototype.GetParent().(type) {
			case *Class:
				defaultTypeContextualTypeArguments = parent.ContextualTypeArguments
			case *Function:
				defaultTypeContextualTypeArguments = parent.ContextualTypeArguments
			}
			resolvedDefaultType := r.ResolveType(
				defaultType,
				nil,
				prototype,
				defaultTypeContextualTypeArguments,
				reportMode,
			)
			if resolvedDefaultType == nil {
				return nil
			}
			result[i] = resolvedDefaultType
			continue
		}

		if reportMode == ReportModeReport {
			var rng *diagnostics.Range
			switch expr := node.(type) {
			case *ast.CallExpression:
				rng = expr.Expression.GetRange().AtEnd()
			case *ast.NewExpression:
				rng = expr.TypeName.GetRange().AtEnd()
			}
			r.program.Error(
				diagnostics.DiagnosticCodeTypeArgumentExpected,
				rng,
				"", "", "",
			)
		}
		return nil
	}
	return result
}

func (r *Resolver) propagateInferredGenericTypes(
	node ast.Node,
	inferredType *types.Type,
	ctxElement Element,
	ctxTypes map[string]*types.Type,
	typeParameterNames map[string]struct{},
) {
	if node == nil || inferredType == nil {
		return
	}

	switch typedNode := node.(type) {
	case *ast.NamedTypeNode:
		typeArgumentNodes := typedNode.TypeArguments
		if len(typeArgumentNodes) > 0 {
			classReference := inferredType.GetClass()
			if classReference == nil {
				return
			}
			classPrototype := r.ResolveTypeName(typedNode.Name, nil, ctxElement, ReportModeSwallow)
			if classPrototype == nil || classPrototype.GetElementKind() != ElementKindClassPrototype {
				return
			}
			classInstance := classReference.(*Class)
			if classInstance.Prototype != classPrototype.(*ClassPrototype) {
				return
			}
			typeArguments := classInstance.TypeArguments
			if len(typeArguments) != len(typeArgumentNodes) {
				return
			}
			for i, typeArgument := range typeArguments {
				r.propagateInferredGenericTypes(typeArgumentNodes[i], typeArgument, ctxElement, ctxTypes, typeParameterNames)
			}
			return
		}

		name := typedNode.Name.Identifier.Text
		currentType, ok := ctxTypes[name]
		if !ok {
			return
		}
		if currentType == types.TypeAuto {
			ctxTypes[name] = inferredType
			return
		}
		if _, ok := typeParameterNames[name]; ok && currentType.IsAssignableTo(inferredType, false) {
			ctxTypes[name] = inferredType
		}

	case *ast.FunctionTypeNode:
		signatureReference := inferredType.GetSignature()
		if signatureReference == nil {
			return
		}
		parameterTypes := signatureReference.ParameterTypes
		limit := len(parameterTypes)
		if len(typedNode.Parameters) < limit {
			limit = len(typedNode.Parameters)
		}
		for i := 0; i < limit; i++ {
			r.propagateInferredGenericTypes(
				typedNode.Parameters[i].Type,
				parameterTypes[i],
				ctxElement,
				ctxTypes,
				typeParameterNames,
			)
		}
		returnType := signatureReference.ReturnType
		if returnType != types.TypeVoid {
			r.propagateInferredGenericTypes(typedNode.ReturnType, returnType, ctxElement, ctxTypes, typeParameterNames)
		}
		if signatureReference.ThisType != nil && typedNode.ExplicitThisType != nil {
			r.propagateInferredGenericTypes(typedNode.ExplicitThisType, signatureReference.ThisType, ctxElement, ctxTypes, typeParameterNames)
		}
	}
}

func (r *Resolver) findConstructorPrototype(prototype *ClassPrototype) *FunctionPrototype {
	for current := prototype; current != nil; current = current.BasePrototype {
		if current.ConstructorPrototype != nil {
			return current.ConstructorPrototype
		}
		if current.InstanceMembers != nil {
			if ctor, ok := current.InstanceMembers[common.CommonNameConstructor].(*FunctionPrototype); ok {
				return ctor
			}
		}
		if members := current.GetMembers(); members != nil {
			if ctor, ok := members[common.CommonNameConstructor].(*FunctionPrototype); ok && ctor.Is(common.CommonFlagsInstance) {
				return ctor
			}
		}
	}
	return nil
}

// =========================================================================
// Class resolution
// =========================================================================

// ResolveClass resolves a class prototype to a concrete Class instance.
func (r *Resolver) ResolveClass(
	prototype *ClassPrototype,
	typeArguments []*types.Type,
	ctxTypes map[string]*types.Type,
	reportMode ReportMode,
) *Class {
	instanceKey := ""
	if typeArguments != nil {
		instanceKey = types.TypesToString(typeArguments)
	}

	// Check if already resolved
	existing := prototype.GetResolvedInstance(instanceKey)
	if existing != nil {
		return existing
	}

	if ctxTypes == nil {
		ctxTypes = make(map[string]*types.Type)
	}

	// Create instance
	nameInclTypeParams := prototype.GetName()
	if instanceKey != "" {
		nameInclTypeParams += "<" + instanceKey + ">"
	}

	var instance *Class
	if prototype.GetElementKind() == ElementKindInterfacePrototype {
		iface := &Interface{}
		iface.Class = *NewClass(nameInclTypeParams, prototype, typeArguments, true)
		iface.Class.SetInterfaceRef(iface)
		instance = &iface.Class
	} else {
		instance = NewClass(nameInclTypeParams, prototype, typeArguments, false)
	}
	prototype.SetResolvedInstance(instanceKey, instance)
	r.resolveClassPending[instance] = struct{}{}

	// Set contextual type arguments
	if typeArguments != nil {
		typeParameterNodes := prototype.TypeParameterNodes()
		if typeParameterNodes != nil {
			for i, tp := range typeParameterNodes {
				if i < len(typeArguments) {
					ctxTypes[tp.Name.Text] = typeArguments[i]
				}
			}
		}
	}
	instance.ContextualTypeArguments = ctxTypes

	anyPending := false

	// Resolve base class if applicable
	if prototype.BasePrototype != nil {
		// Check for circular inheritance
		current := prototype.BasePrototype
		for current != nil {
			if current == prototype {
				if reportMode == ReportModeReport {
					ident := prototype.IdentifierNode()
					if ident != nil {
						r.program.Error(
							diagnostics.DiagnosticCode0IsReferencedDirectlyOrIndirectlyInItsOwnBaseExpression,
							ident.GetRange(),
							prototype.GetInternalName(),
						)
					}
				}
				return nil
			}
			current = current.BasePrototype
		}

		// Resolve base class
		clonedCtx := make(map[string]*types.Type, len(ctxTypes))
		for k, v := range ctxTypes {
			clonedCtx[k] = v
		}
		base := r.resolveBaseClass(prototype, clonedCtx, reportMode)
		if base == nil {
			return nil
		}
		instance.SetBase(base)

		if _, pending := r.resolveClassPending[base]; pending {
			anyPending = true
		}
	} else if prototype.ImplicitlyExtendsObject {
		objectInstance := r.program.ObjectInstance()
		if objectInstance != nil {
			instance.SetBase(objectInstance)
		}
	}

	interfacePrototypes := prototype.InterfacePrototypes
	if interfacePrototypes != nil {
		for i, ifaceProto := range interfacePrototypes {
			current := &ifaceProto.ClassPrototype
			for current != nil {
				if current == prototype {
					if reportMode == ReportModeReport {
						ident := prototype.IdentifierNode()
						if ident != nil {
							r.program.Error(
								diagnostics.DiagnosticCode0IsReferencedDirectlyOrIndirectlyInItsOwnBaseExpression,
								ident.GetRange(),
								prototype.GetInternalName(),
							)
						}
					}
					return nil
				}
				current = current.BasePrototype
			}

			var typeArgumentNodes []ast.Node
			implementsNodes := prototype.ImplementsNodes()
			if i < len(implementsNodes) && implementsNodes[i] != nil {
				typeArgumentNodes = implementsNodes[i].TypeArguments
			}

			iface := r.ResolveClassInclTypeArguments(
				&ifaceProto.ClassPrototype,
				typeArgumentNodes,
				nil,
				prototype.GetParent(),
				util.CloneMap(ctxTypes),
				prototype.GetDeclaration(),
				reportMode,
			)
			if iface == nil {
				return nil
			}
			if ifaceRef := iface.AsInterface(); ifaceRef != nil {
				instance.AddInterface(ifaceRef)
				if _, pending := r.resolveClassPending[iface]; pending {
					anyPending = true
				}
			}
		}
	}

	// Finish resolving only once dependencies are fully resolved.
	if anyPending {
		return instance
	}

	r.finishResolveClass(instance, reportMode)
	return instance
}

// resolveBaseClass resolves the base class of a class prototype.
func (r *Resolver) resolveBaseClass(
	prototype *ClassPrototype,
	ctxTypes map[string]*types.Type,
	reportMode ReportMode,
) *Class {
	basePrototype := prototype.BasePrototype
	if basePrototype == nil {
		return nil
	}

	// Resolve base with its own type arguments
	extendsNode := prototype.ExtendsNode()
	if extendsNode != nil {
		return r.ResolveClassInclTypeArguments(
			basePrototype,
			extendsNode.TypeArguments,
			nil,
			prototype.GetParent(),
			ctxTypes,
			extendsNode,
			reportMode,
		)
	}

	// Non-generic base
	return r.ResolveClass(basePrototype, nil, ctxTypes, reportMode)
}

// ResolveClassInclTypeArguments resolves a class prototype by first resolving type argument nodes.
func (r *Resolver) ResolveClassInclTypeArguments(
	prototype *ClassPrototype,
	typeArgumentNodes []ast.Node,
	f *flow.Flow,
	ctxElement Element,
	ctxTypes map[string]*types.Type,
	reportNode ast.Node,
	reportMode ReportMode,
) *Class {
	var resolvedTypeArguments []*types.Type

	if prototype.Is(common.CommonFlagsGeneric) {
		typeParameterNodes := prototype.TypeParameterNodes()
		constructorPrototype := r.findConstructorPrototype(prototype)
		if len(typeArgumentNodes) == 0 && constructorPrototype != nil && f != nil && len(ctxTypes) == 0 {
			resolvedTypeArguments = r.InferGenericTypeArguments(
				reportNode,
				constructorPrototype,
				typeParameterNodes,
				f,
				reportMode,
			)
		} else if typeParameterNodes != nil {
			resolvedTypeArguments = r.ResolveTypeArguments(
				typeParameterNodes,
				typeArgumentNodes,
				f,
				ctxElement,
				ctxTypes,
				reportNode,
				reportMode,
			)
		}
		if resolvedTypeArguments == nil {
			return nil
		}
	} else if len(typeArgumentNodes) > 0 {
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeType0IsNotGeneric,
				reportNode.GetRange(),
				prototype.GetInternalName(),
				"", "",
			)
		}
		return nil
	}

	return r.ResolveClass(prototype, resolvedTypeArguments, ctxTypes, reportMode)
}

func (r *Resolver) checkOverrideVisibility(
	name string,
	thisMember DeclaredElement,
	thisClass *Class,
	baseMember DeclaredElement,
	baseClass *Class,
	reportMode ReportMode,
) bool {
	hasErrors := false

	thisIdent := thisMember.IdentifierNode()
	baseIdent := baseMember.IdentifierNode()
	thisRange := thisMember.GetDeclaration().GetRange()
	baseRange := baseMember.GetDeclaration().GetRange()
	if thisIdent != nil {
		thisRange = thisIdent.GetRange()
	}
	if baseIdent != nil {
		baseRange = baseIdent.GetRange()
	}

	if thisMember.Is(common.CommonFlagsConstructor) {
		if baseMember.Is(common.CommonFlagsPrivate) {
			if reportMode == ReportModeReport {
				r.program.ErrorRelated(
					diagnostics.DiagnosticCodeCannotExtendAClass0ClassConstructorIsMarkedAsPrivate,
					thisRange,
					baseRange,
					baseClass.GetInternalName(), "", "",
				)
			}
			hasErrors = true
		}
	} else if thisMember.Is(common.CommonFlagsPrivate) {
		if baseMember.Is(common.CommonFlagsPrivate) {
			if reportMode == ReportModeReport {
				r.program.ErrorRelated(
					diagnostics.DiagnosticCodeTypesHaveSeparateDeclarationsOfAPrivateProperty0,
					thisRange,
					baseRange,
					name, "", "",
				)
			}
			hasErrors = true
		} else {
			if reportMode == ReportModeReport {
				r.program.ErrorRelated(
					diagnostics.DiagnosticCodeProperty0IsPrivateInType1ButNotInType2,
					thisRange,
					baseRange,
					name,
					thisClass.GetInternalName(),
					baseClass.GetInternalName(),
				)
			}
			hasErrors = true
		}
	} else if thisMember.Is(common.CommonFlagsProtected) {
		if baseMember.Is(common.CommonFlagsPrivate) {
			if reportMode == ReportModeReport {
				r.program.ErrorRelated(
					diagnostics.DiagnosticCodeProperty0IsPrivateInType1ButNotInType2,
					thisRange,
					baseRange,
					name,
					baseClass.GetInternalName(),
					thisClass.GetInternalName(),
				)
			}
			hasErrors = true
		} else if baseMember.IsPublic() {
			if reportMode == ReportModeReport {
				r.program.ErrorRelated(
					diagnostics.DiagnosticCodeProperty0IsProtectedInType1ButPublicInType2,
					thisRange,
					baseRange,
					name,
					thisClass.GetInternalName(),
					baseClass.GetInternalName(),
				)
			}
			hasErrors = true
		}
	} else if thisMember.IsPublic() {
		if baseMember.Is(common.CommonFlagsPrivate) {
			if reportMode == ReportModeReport {
				r.program.ErrorRelated(
					diagnostics.DiagnosticCodeProperty0IsPrivateInType1ButNotInType2,
					thisRange,
					baseRange,
					name,
					baseClass.GetInternalName(),
					thisClass.GetInternalName(),
				)
			}
			hasErrors = true
		} else if baseMember.Is(common.CommonFlagsProtected) {
			if reportMode == ReportModeReport {
				r.program.ErrorRelated(
					diagnostics.DiagnosticCodeProperty0IsProtectedInType1ButPublicInType2,
					thisRange,
					baseRange,
					name,
					baseClass.GetInternalName(),
					thisClass.GetInternalName(),
				)
			}
			hasErrors = true
		}
	}

	return !hasErrors
}

func (r *Resolver) classInstanceMemberPrototypes(prototype *ClassPrototype) []DeclaredElement {
	if prototype.InstanceMembers != nil {
		if len(prototype.InstanceMemberOrder) == 0 {
			result := make([]DeclaredElement, 0, len(prototype.InstanceMembers))
			for _, member := range prototype.InstanceMembers {
				result = append(result, member)
			}
			if len(result) == 0 {
				return nil
			}
			return result
		}
		result := make([]DeclaredElement, 0, len(prototype.InstanceMemberOrder))
		for _, name := range prototype.InstanceMemberOrder {
			if member, ok := prototype.InstanceMembers[name]; ok {
				result = append(result, member)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	}
	members := prototype.GetMembers()
	if members == nil {
		return nil
	}
	result := make([]DeclaredElement, 0, len(members))
	for _, member := range members {
		if member.Is(common.CommonFlagsInstance) {
			result = append(result, member)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// finishResolveClass completes class resolution by resolving members and interface prototypes.
func (r *Resolver) finishResolveClass(instance *Class, reportMode ReportMode) {
	prototype := instance.Prototype
	members := instance.GetMembers()
	if members == nil {
		members = make(map[string]DeclaredElement)
		instance.SetMembers(members)
	}

	unimplemented := make(map[string]DeclaredElement)
	if instance.Interfaces != nil {
		for iface := range instance.Interfaces {
			if _, pending := r.resolveClassPending[&iface.Class]; pending {
				continue
			}
			ifaceMembers := iface.GetMembers()
			if ifaceMembers == nil {
				continue
			}
			for memberName, ifaceMember := range ifaceMembers {
				existingMember := instance.GetMember(memberName)
				if existingMember != nil && !r.checkOverrideVisibility(memberName, existingMember, instance, ifaceMember, &iface.Class, reportMode) {
					continue
				}
				members[memberName] = ifaceMember
				unimplemented[memberName] = ifaceMember
			}
		}
	}

	memoryOffset := uint32(0)
	base := instance.Base
	if base != nil {
		baseMembers := base.GetMembers()
		if baseMembers != nil {
			for memberName, baseMember := range baseMembers {
				if prototype.ImplicitlyExtendsObject && baseMember.Is(common.CommonFlagsStatic) {
					continue
				}
				existingMember := instance.GetMember(memberName)
				if existingMember != nil && !r.checkOverrideVisibility(memberName, existingMember, instance, baseMember, base, reportMode) {
					continue
				}
				members[memberName] = baseMember
				if baseMember.Is(common.CommonFlagsAbstract) {
					unimplemented[memberName] = baseMember
				} else {
					delete(unimplemented, memberName)
				}
			}
		}
		memoryOffset = base.NextMemoryOffset
	}

	instanceMemberPrototypes := r.classInstanceMemberPrototypes(prototype)
	if instanceMemberPrototypes != nil {
		for _, member := range instanceMemberPrototypes {
			memberName := member.GetName()
			if base != nil {
				baseMember := base.GetMember(memberName)
				if baseMember != nil {
					r.checkOverrideVisibility(memberName, member, instance, baseMember, base, reportMode)
				}
			}

			switch member.GetElementKind() {
			case ElementKindFunctionPrototype:
				boundPrototype := member.(*FunctionPrototype).ToBound(instance)
				instance.Add(boundPrototype.GetName(), boundPrototype, nil)

			case ElementKindPropertyPrototype:
				boundPrototype := member.(*PropertyPrototype).ToBound(instance)
				if boundPrototype.IsField() {
					boundInstance := r.ResolveProperty(boundPrototype, reportMode)
					if boundInstance == nil {
						break
					}
					fieldType := boundInstance.GetResolvedType()
					if fieldType == types.TypeVoid {
						break
					}
					if fieldType.IsExternalReference() {
						if reportMode == ReportModeReport {
							typeNode := boundPrototype.PropertyTypeNode()
							if typeNode != nil {
								r.program.Error(
									diagnostics.DiagnosticCodeNotImplemented0,
									typeNode.GetRange(),
									"Reference typed fields",
									"", "",
								)
							}
						}
						break
					}

					needsLayout := true
					if base != nil {
						existingMember := base.GetMember(boundPrototype.GetName())
						if existingPrototype, ok := existingMember.(*PropertyPrototype); ok {
							existingProperty := r.ResolveProperty(existingPrototype, reportMode)
							if existingProperty != nil && existingProperty.IsField() {
								if existingProperty.GetResolvedType() != boundInstance.GetResolvedType() {
									if reportMode == ReportModeReport {
										r.program.ErrorRelated(
											diagnostics.DiagnosticCodeProperty0InType1IsNotAssignableToTheSamePropertyInBaseType2,
											boundInstance.IdentifierNode().GetRange(),
											existingProperty.IdentifierNode().GetRange(),
											boundInstance.GetName(),
											instance.GetInternalName(),
											base.GetInternalName(),
										)
									}
									break
								}
								boundInstance.MemoryOffset = existingProperty.MemoryOffset
								needsLayout = false
							}
						}
					}

					if needsLayout {
						byteSize := uint32(fieldType.ByteSize())
						if !util.IsPowerOf2(int32(byteSize)) {
							panic("field size must be a power of two")
						}
						mask := byteSize - 1
						if memoryOffset&mask != 0 {
							memoryOffset = (memoryOffset | mask) + 1
						}
						boundInstance.MemoryOffset = int32(memoryOffset)
						memoryOffset += byteSize
					}

					boundPrototype.PropertyInstance = boundInstance
					instance.Add(boundPrototype.GetName(), boundPrototype, nil)
					if typeNode := boundPrototype.FieldDeclaration.Type; typeNode != nil {
						r.program.CheckTypeSupported(fieldType, typeNode)
					}
				} else {
					instance.Add(boundPrototype.GetName(), boundPrototype, nil)
				}
			}

			if !member.Is(common.CommonFlagsAbstract) {
				delete(unimplemented, memberName)
			}
		}
	}

	if !instance.IsInterface() {
		if !instance.Is(common.CommonFlagsAbstract) {
			for memberName, member := range unimplemented {
				if reportMode == ReportModeReport {
					r.program.ErrorRelated(
						diagnostics.DiagnosticCodeNonAbstractClass0DoesNotImplementInheritedAbstractMember1From2,
						instance.IdentifierNode().GetRange(),
						member.IdentifierNode().GetRange(),
						instance.GetInternalName(),
						memberName,
						member.GetParent().GetInternalName(),
					)
				}
			}
		}

		instance.NextMemoryOffset = memoryOffset

		ctorPrototype := instance.GetMember(common.CommonNameConstructor)
		if ctorPrototype != nil && ctorPrototype.GetParent() == instance {
			if ctor, ok := ctorPrototype.(*FunctionPrototype); ok {
				ctorInstance := r.ResolveFunction(ctor, nil, instance.ContextualTypeArguments, reportMode)
				if ctorInstance != nil {
					instance.ConstructorInstance = ctorInstance
				}
			}
		}
	}

	for overloadKind, overloadPrototype := range prototype.OperatorOverloadPrototypes {
		if overloadKind == OperatorKindInvalid || overloadPrototype.Is(common.CommonFlagsGeneric) {
			continue
		}

		var operatorInstance *Function
		if overloadPrototype.Is(common.CommonFlagsInstance) {
			boundPrototype := overloadPrototype.ToBound(instance)
			operatorInstance = r.ResolveFunction(boundPrototype, nil, map[string]*types.Type{}, reportMode)
		} else {
			operatorInstance = r.ResolveFunction(overloadPrototype, nil, map[string]*types.Type{}, reportMode)
		}
		if operatorInstance == nil {
			continue
		}

		if instance.OperatorOverloads == nil {
			instance.OperatorOverloads = make(map[OperatorKind]*Function)
		}

		if operatorInstance.Is(common.CommonFlagsInstance) {
			switch overloadKind {
			case OperatorKindPrefixInc, OperatorKindPrefixDec, OperatorKindPostfixInc, OperatorKindPostfixDec:
				returnType := operatorInstance.Signature.ReturnType
				if !returnType.IsAssignableTo(instance.GetResolvedType(), false) && reportMode == ReportModeReport {
					returnTypeNode := overloadPrototype.FunctionTypeNode().ReturnType
					if returnTypeNode != nil {
						r.program.Error(
							diagnostics.DiagnosticCodeType0IsNotAssignableToType1,
							returnTypeNode.GetRange(),
							returnType.String(),
							instance.GetResolvedType().String(),
							"",
						)
					}
				}
			}
		}

		if _, ok := instance.OperatorOverloads[overloadKind]; !ok {
			instance.OperatorOverloads[overloadKind] = operatorInstance
			if overloadKind == OperatorKindIndexedGet || overloadKind == OperatorKindIndexedSet {
				if instance.IndexSignature_ == nil {
					instance.IndexSignature_ = NewIndexSignature(instance)
				}
				if overloadKind == OperatorKindIndexedGet {
					instance.IndexSignature_.SetType(operatorInstance.Signature.ReturnType)
				}
			}
		} else if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeDuplicateDecorator,
				operatorInstance.GetDeclaration().GetRange(),
				"", "", "",
			)
		}
	}

	delete(r.resolveClassPending, instance)
	for pending := range r.resolveClassPending {
		dependsOnInstance := pending.Base == instance
		if pending.Interfaces != nil {
			anyPending := false
			for iface := range pending.Interfaces {
				if &iface.Class == instance {
					dependsOnInstance = true
				} else if _, ok := r.resolveClassPending[&iface.Class]; ok {
					anyPending = true
				}
			}
			if anyPending {
				continue
			}
		}
		if dependsOnInstance {
			r.finishResolveClass(pending, reportMode)
		}
	}
}

// =========================================================================
// Property resolution
// =========================================================================

// ResolveProperty resolves a property prototype to a concrete Property instance.
func (r *Resolver) ResolveProperty(
	prototype *PropertyPrototype,
	reportMode ReportMode,
) *Property {
	// Check if already resolved
	if prototype.PropertyInstance != nil {
		return prototype.PropertyInstance
	}

	property := NewProperty(prototype, prototype.GetParent())
	prototype.PropertyInstance = property

	// Resolve getter if present
	if prototype.GetterPrototype != nil {
		getter := r.ResolveFunction(prototype.GetterPrototype, nil, map[string]*types.Type{}, reportMode)
		if getter != nil {
			property.GetterInstance = getter
			property.SetType(getter.Signature.ReturnType)
		}
	}

	// Resolve setter if present
	if prototype.SetterPrototype != nil {
		setter := r.ResolveFunction(prototype.SetterPrototype, nil, map[string]*types.Type{}, reportMode)
		if setter != nil {
			property.SetterInstance = setter
			if !property.Is(common.CommonFlagsResolved) && len(setter.Signature.ParameterTypes) == 1 {
				property.SetType(setter.Signature.ParameterTypes[0])
			}
		}
	}

	property.CheckVisibility(&r.program.DiagnosticEmitter)
	return property
}

// =========================================================================
// Expression resolution
// =========================================================================

// LookupExpression looks up the program element an expression refers to.
// Ported from: assemblyscript/src/resolver.ts lookupExpression (lines 1950-2150).
func (r *Resolver) LookupExpression(
	node ast.Node,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	if node == nil {
		return nil
	}
	r.CurrentThisExpression = nil
	r.CurrentElementExpression = nil

	switch node.GetKind() {
	case ast.NodeKindIdentifier:
		return r.lookupIdentifierExpression(node.(*ast.IdentifierExpression), ctxFlow, reportMode)

	case ast.NodeKindThis:
		return r.lookupThisExpression(node.(*ast.IdentifierExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindSuper:
		return r.lookupSuperExpression(node.(*ast.IdentifierExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindTrue, ast.NodeKindFalse, ast.NodeKindNull, ast.NodeKindLiteral:
		return r.lookupLiteralExpression(node, ctxFlow, ctxType, reportMode)

	case ast.NodeKindPropertyAccess:
		return r.lookupPropertyAccessExpression(node.(*ast.PropertyAccessExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindElementAccess:
		return r.lookupElementAccessExpression(node.(*ast.ElementAccessExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindComma:
		return r.lookupCommaExpression(node.(*ast.CommaExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindInstanceOf:
		return r.lookupInstanceOfExpression(node.(*ast.InstanceOfExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindTernary:
		return r.lookupTernaryExpression(node.(*ast.TernaryExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindNew:
		return r.lookupNewExpression(node.(*ast.NewExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindParenthesized:
		return r.LookupExpression(node.(*ast.ParenthesizedExpression).Expression, ctxFlow, ctxType, reportMode)

	case ast.NodeKindCall:
		return r.lookupCallExpression(node.(*ast.CallExpression), ctxFlow, ctxType, reportMode)
	}

	if reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeExpressionDoesNotCompileToAValueAtRuntime,
			node.GetRange(),
			"", "", "",
		)
	}
	return nil
}

// lookupThisExpression looks up the program element a this expression refers to.
func (r *Resolver) lookupThisExpression(
	node *ast.IdentifierExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	if ctxFlow != nil && ctxFlow.IsInline() {
		if thisLocal := ctxFlow.LookupLocal(common.CommonNameThis); thisLocal != nil {
			r.CurrentThisExpression = nil
			r.CurrentElementExpression = nil
			if elem, ok := thisLocal.(Element); ok {
				return elem
			}
			return nil
		}
	}
	if ctxFlow != nil {
		if sourceFunction, ok := ctxFlow.SourceFunction().(Element); ok {
			if parent := sourceFunction.GetParent(); parent != nil {
				r.CurrentThisExpression = nil
				r.CurrentElementExpression = nil
				return parent
			}
		}
	}
	if reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeThisCannotBeReferencedInCurrentLocation,
			node.GetRange(),
		)
	}
	return nil
}

// resolveThisExpression resolves a this expression to its static type.
func (r *Resolver) resolveThisExpression(
	node *ast.IdentifierExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	element := r.lookupThisExpression(node, ctxFlow, ctxType, reportMode)
	if element == nil {
		return nil
	}
	typ := r.GetTypeOfElement(element)
	if typ == nil && reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeExpressionCannotBeRepresentedByAType,
			node.GetRange(),
		)
	}
	return typ
}

// lookupSuperExpression looks up the program element a super expression refers to.
func (r *Resolver) lookupSuperExpression(
	node *ast.IdentifierExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	if ctxFlow != nil && ctxFlow.IsInline() {
		if superLocal := ctxFlow.LookupLocal(common.CommonNameSuper); superLocal != nil {
			r.CurrentThisExpression = nil
			r.CurrentElementExpression = nil
			if elem, ok := superLocal.(Element); ok {
				return elem
			}
			return nil
		}
	}
	if ctxFlow != nil {
		if sourceFunction, ok := ctxFlow.SourceFunction().(Element); ok {
			parent := sourceFunction.GetParent()
			if parent != nil && parent.GetElementKind() == ElementKindClass {
				base := parent.(*Class).Base
				if base != nil {
					r.CurrentThisExpression = nil
					r.CurrentElementExpression = nil
					return base
				}
			}
		}
	}
	if reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeSuperCanOnlyBeReferencedInADerivedClass,
			node.GetRange(),
		)
	}
	return nil
}

// resolveSuperExpression resolves a super expression to its static type.
func (r *Resolver) resolveSuperExpression(
	node *ast.IdentifierExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	element := r.lookupSuperExpression(node, ctxFlow, ctxType, reportMode)
	if element == nil {
		return nil
	}
	typ := r.GetTypeOfElement(element)
	if typ == nil && reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeExpressionCannotBeRepresentedByAType,
			node.GetRange(),
		)
	}
	return typ
}

// lookupIdentifierExpression resolves an identifier to its program element.
func (r *Resolver) lookupIdentifierExpression(
	node *ast.IdentifierExpression,
	ctxFlow *flow.Flow,
	reportMode ReportMode,
) Element {
	name := node.Text

	// Check special identifiers
	switch node.GetKind() {
	case ast.NodeKindThis:
		// "this" refers to the class context
		if ctxFlow != nil {
			sourceFunc := ctxFlow.SourceFunction()
			parent := sourceFunc.(Element).GetParent()
			if parent != nil && parent.GetElementKind() == ElementKindClass {
				return parent
			}
		}
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCode0KeywordCannotBeUsedHere,
				node.GetRange(),
				"this", "", "",
			)
		}
		return nil
	case ast.NodeKindSuper:
		// "super" refers to the base class
		if ctxFlow != nil {
			sourceFunc := ctxFlow.SourceFunction()
			parent := sourceFunc.(Element).GetParent()
			if parent != nil && parent.GetElementKind() == ElementKindClass {
				classInstance := parent.(*Class)
				if classInstance.Base != nil {
					return classInstance.Base
				}
			}
		}
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeSuperCanOnlyBeReferencedInADerivedClass,
				node.GetRange(),
				"", "", "",
			)
		}
		return nil
	}

	// Normal identifier: look up through flow scope chain
	if ctxFlow != nil {
		// First check scoped locals — these don't return as Elements (they're FlowLocalRef).
		// The compiler handles locals directly; LookupExpression returns program Elements.
		// So skip locals and look up in program scope.
		elemRef := ctxFlow.TargetFunction.FlowLookup(name)
		if elemRef != nil {
			if elem, ok := elemRef.(Element); ok {
				return elem
			}
		}
	}

	// Fall back to global program lookup
	elem := r.program.Lookup(name)
	if elem != nil {
		return elem
	}

	if reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeCannotFindName0,
			node.GetRange(),
			name, "", "",
		)
	}
	return nil
}

// determineIntegerLiteralType determines the final type of an integer literal.
func (r *Resolver) determineIntegerLiteralType(
	expr *ast.IntegerLiteralExpression,
	negate bool,
	ctxType *types.Type,
) *types.Type {
	if ctxType == nil {
		ctxType = types.TypeAuto
	}

	intValue := expr.Value
	if negate {
		if intValue == 0 {
			if ctxType.IsFloatValue() {
				if ctxType == types.TypeF32 {
					return types.TypeF32
				}
				return types.TypeF64
			}
			if !ctxType.IsIntegerValue() {
				return types.TypeF64
			}
		}
		intValue = -intValue
	}

	if ctxType.IsValue() {
		switch ctxType.Kind {
		case types.TypeKindBool:
			if intValue == 0 || intValue == 1 {
				return types.TypeBool
			}
		case types.TypeKindI8:
			if intValue >= -128 && intValue <= 127 {
				return types.TypeI8
			}
		case types.TypeKindU8:
			if intValue >= 0 && intValue <= 255 {
				return types.TypeU8
			}
		case types.TypeKindI16:
			if intValue >= -32768 && intValue <= 32767 {
				return types.TypeI16
			}
		case types.TypeKindU16:
			if intValue >= 0 && intValue <= 65535 {
				return types.TypeU16
			}
		case types.TypeKindI32:
			if intValue >= -2147483648 && intValue <= 2147483647 {
				return types.TypeI32
			}
		case types.TypeKindU32:
			if intValue >= 0 && intValue <= 4294967295 {
				return types.TypeU32
			}
		case types.TypeKindIsize:
			if !r.program.Options.IsWasm64() {
				if intValue >= -2147483648 && intValue <= 2147483647 {
					return types.TypeIsize32
				}
				break
			}
			return types.TypeIsize64
		case types.TypeKindUsize:
			if !r.program.Options.IsWasm64() {
				if intValue >= 0 && intValue <= 4294967295 {
					return types.TypeUsize32
				}
				break
			}
			return types.TypeUsize64
		case types.TypeKindI64:
			return types.TypeI64
		case types.TypeKindU64:
			return types.TypeU64
		case types.TypeKindF32:
			return types.TypeF32
		case types.TypeKindF64:
			return types.TypeF64
		}
	}

	if intValue >= -2147483648 && intValue <= 2147483647 {
		return types.TypeI32
	}
	if intValue >= 0 && intValue <= 4294967295 {
		return types.TypeU32
	}
	return types.TypeI64
}

// lookupPropertyAccessExpression resolves a property access expression.
func (r *Resolver) lookupPropertyAccessExpression(
	node *ast.PropertyAccessExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	propertyName := node.Property.Text
	r.CurrentThisExpression = node.Expression

	// For local variable property access (e.g., x.foo where x is a local),
	// we need to resolve the TYPE of the expression, not just the element.
	// First, try resolving as an element.
	target := r.LookupExpression(node.Expression, ctxFlow, ctxType, ReportModeSwallow)

	// If target is nil (e.g., local variable), resolve the expression type
	// and look up the property on the type's class.
	if target == nil {
		exprType := r.ResolveExpression(node.Expression, ctxFlow, ctxType, reportMode)
		if exprType != nil {
			classRef := exprType.GetClassOrWrapper(r.program)
			if classRef != nil {
				if classInstance, ok := classRef.(*Class); ok {
					if classInstance.Prototype.InstanceMembers != nil {
						if member, mOk := classInstance.Prototype.InstanceMembers[propertyName]; mOk {
							return member
						}
					}
					// Also check static members
					member := classInstance.Prototype.GetMember(propertyName)
					if member != nil {
						return member
					}
				}
			}
		}
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeProperty0DoesNotExistOnType1,
				node.Property.GetRange(),
				propertyName, "", "",
			)
		}
		return nil
	}

	// Depending on the target kind, look up the property
	switch target.GetElementKind() {
	case ElementKindClass:
		classInstance := target.(*Class)
		// Look up in instance members first
		if classInstance.Prototype.InstanceMembers != nil {
			if member, ok := classInstance.Prototype.InstanceMembers[propertyName]; ok {
				return member
			}
		}
		// Look up in members (static members on the prototype)
		member := classInstance.Prototype.Lookup(propertyName, false)
		if member != nil {
			return member
		}

	case ElementKindClassPrototype:
		classPrototype := target.(*ClassPrototype)
		// Static member access — look in the prototype's own members
		member := classPrototype.GetMember(propertyName)
		if member != nil {
			return member
		}
		// Also check instance members
		if classPrototype.InstanceMembers != nil {
			if m, ok := classPrototype.InstanceMembers[propertyName]; ok {
				return m
			}
		}

	case ElementKindGlobal:
		// Global might resolve to a class type, check its type
		global := target.(*Global)
		globalType := global.GetResolvedType()
		if globalType != nil {
			classRef := globalType.GetClassOrWrapper(r.program)
			if classRef != nil {
				if classInstance, ok := classRef.(*Class); ok {
					if classInstance.Prototype.InstanceMembers != nil {
						if member, mOk := classInstance.Prototype.InstanceMembers[propertyName]; mOk {
							return member
						}
					}
				}
			}
		}

	case ElementKindNamespace:
		ns := target.(*Namespace)
		member := ns.Lookup(propertyName, false)
		if member != nil {
			return member
		}

	case ElementKindEnum:
		enum := target.(*Enum)
		member := enum.Lookup(propertyName, false)
		if member != nil {
			return member
		}

	case ElementKindPropertyPrototype:
		// Resolved to a property prototype, try resolving and looking up on its type
		pp := target.(*PropertyPrototype)
		prop := r.ResolveProperty(pp, reportMode)
		if prop != nil && prop.GetterInstance != nil {
			returnType := prop.GetterInstance.Signature.ReturnType
			if returnType != nil {
				classRef := returnType.GetClassOrWrapper(r.program)
				if classRef != nil {
					if classInstance, ok := classRef.(*Class); ok {
						if classInstance.Prototype.InstanceMembers != nil {
							if member, mOk := classInstance.Prototype.InstanceMembers[propertyName]; mOk {
								return member
							}
						}
					}
				}
			}
		}

	case ElementKindProperty:
		// Resolved property — look up on its return type
		prop := target.(*Property)
		if prop.GetterInstance != nil {
			returnType := prop.GetterInstance.Signature.ReturnType
			if returnType != nil {
				classRef := returnType.GetClassOrWrapper(r.program)
				if classRef != nil {
					if classInstance, ok := classRef.(*Class); ok {
						if classInstance.Prototype.InstanceMembers != nil {
							if member, mOk := classInstance.Prototype.InstanceMembers[propertyName]; mOk {
								return member
							}
						}
					}
				}
			}
		}
	}

	if reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeProperty0DoesNotExistOnType1,
			node.Property.GetRange(),
			propertyName, target.GetInternalName(), "",
		)
	}
	return nil
}

// lookupElementAccessExpression resolves an element access expression.
func (r *Resolver) lookupElementAccessExpression(
	node *ast.ElementAccessExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	r.CurrentThisExpression = node.Expression
	r.CurrentElementExpression = node.ElementExpression

	// Try resolving target as an element first
	target := r.LookupExpression(node.Expression, ctxFlow, ctxType, ReportModeSwallow)

	// If target is nil (e.g., local variable), resolve the type and find the class
	if target == nil {
		exprType := r.ResolveExpression(node.Expression, ctxFlow, ctxType, reportMode)
		if exprType != nil {
			classRef := exprType.GetClassOrWrapper(r.program)
			if classRef != nil {
				if classInstance, ok := classRef.(*Class); ok {
					return classInstance
				}
			}
		}
		return nil
	}

	// For element access, we need to resolve to the class that has [] operator
	switch target.GetElementKind() {
	case ElementKindClass:
		// The class itself — return it so the compiler can look up indexed operators
		return target

	case ElementKindGlobal:
		// Check if the global's type is a class with [] operator
		global := target.(*Global)
		globalType := global.GetResolvedType()
		if globalType != nil {
			classRef := globalType.GetClassOrWrapper(r.program)
			if classRef != nil {
				if classInstance, ok := classRef.(*Class); ok {
					return classInstance
				}
			}
		}
	}

	if reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeNotImplemented0,
			node.GetRange(),
			"Element access on type", "", "",
		)
	}
	return nil
}

// lookupLiteralExpression looks up the program element a literal expression refers to.
func (r *Resolver) lookupLiteralExpression(
	node ast.Node,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	r.CurrentThisExpression = node
	r.CurrentElementExpression = nil

	switch n := node.(type) {
	case *ast.IntegerLiteralExpression:
		return r.GetElementOfType(r.determineIntegerLiteralType(n, false, ctxType))

	case *ast.FloatLiteralExpression:
		if ctxType == types.TypeF32 {
			return r.GetElementOfType(types.TypeF32)
		}
		return r.GetElementOfType(types.TypeF64)

	case *ast.StringLiteralExpression, *ast.TemplateLiteralExpression:
		return r.program.StringInstance()

	case *ast.RegexpLiteralExpression:
		return r.program.RegexpInstance()

	case *ast.ArrayLiteralExpression:
		if ctxType != nil {
			if classRef := ctxType.GetClass(); classRef != nil {
				if classInstance, ok := classRef.(*Class); ok && classInstance.Prototype == r.program.ArrayPrototype() {
					return r.GetElementOfType(ctxType)
				}
			}
		}

		expressions := n.ElementExpressions
		elementType := types.TypeAuto
		numNullLiterals := 0
		for _, expression := range expressions {
			if expression == nil {
				continue
			}
			if expression.GetKind() == ast.NodeKindNull && len(expressions) > 1 {
				numNullLiterals++
				continue
			}
			currentType := r.ResolveExpression(expression, ctxFlow, elementType, reportMode)
			if currentType == nil {
				return nil
			}
			if elementType == types.TypeAuto {
				elementType = currentType
			} else if currentType != elementType {
				if commonType := types.CommonType(elementType, currentType, elementType, false); commonType != nil {
					elementType = commonType
				}
			}
		}
		if elementType == types.TypeAuto {
			if numNullLiterals == len(expressions) {
				elementType = r.program.Options.UsizeType()
			} else {
				if reportMode == ReportModeReport {
					r.program.Error(
						diagnostics.DiagnosticCodeTheTypeArgumentForTypeParameter0CannotBeInferredFromTheUsageConsiderSpecifyingTheTypeArgumentsExplicitly,
						node.GetRange(),
						"T",
					)
				}
				return nil
			}
		}
		if numNullLiterals > 0 && elementType.IsInternalReference() {
			elementType = elementType.AsNullable()
		}
		arrayPrototype := r.program.ArrayPrototype()
		if arrayPrototype == nil {
			return nil
		}
		return r.ResolveClass(arrayPrototype, []*types.Type{elementType}, make(map[string]*types.Type), reportMode)

	case *ast.ObjectLiteralExpression:
		if ctxType != nil && ctxType.IsClass() {
			return r.GetElementOfType(ctxType)
		}
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeExpressionCannotBeRepresentedByAType,
				node.GetRange(),
			)
		}
		return nil

	case *ast.IdentifierExpression:
		switch n.GetKind() {
		case ast.NodeKindTrue, ast.NodeKindFalse:
			return r.GetElementOfType(types.TypeBool)
		case ast.NodeKindNull:
			if ctxType == nil {
				return nil
			}
			return r.GetElementOfType(ctxType)
		}
	}

	return nil
}

// resolveLiteralExpression resolves a literal expression to its static type.
func (r *Resolver) resolveLiteralExpression(
	node ast.Node,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	switch n := node.(type) {
	case *ast.IntegerLiteralExpression:
		return r.determineIntegerLiteralType(n, false, ctxType)

	case *ast.FloatLiteralExpression:
		if ctxType == types.TypeF32 {
			return types.TypeF32
		}
		return types.TypeF64

	case *ast.StringLiteralExpression, *ast.TemplateLiteralExpression:
		stringInstance := r.program.StringInstance()
		if stringInstance != nil {
			return stringInstance.GetResolvedType()
		}
		return nil

	case *ast.RegexpLiteralExpression:
		regexpInstance := r.program.RegexpInstance()
		if regexpInstance != nil {
			return regexpInstance.GetResolvedType()
		}
		return nil

	case *ast.ArrayLiteralExpression:
		if ctxType != nil {
			if classRef := ctxType.GetClass(); classRef != nil {
				if classInstance, ok := classRef.(*Class); ok && classInstance.Prototype == r.program.ArrayPrototype() {
					return ctxType
				}
			}
		}

		element := r.lookupLiteralExpression(node, ctxFlow, ctxType, reportMode)
		if element == nil {
			return nil
		}
		typ := r.GetTypeOfElement(element)
		if typ == nil && reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeExpressionCannotBeRepresentedByAType,
				node.GetRange(),
			)
		}
		return typ

	case *ast.ObjectLiteralExpression:
		if ctxType != nil && ctxType.IsClass() {
			return ctxType
		}
		if reportMode == ReportModeReport {
			r.program.Error(
				diagnostics.DiagnosticCodeExpressionCannotBeRepresentedByAType,
				node.GetRange(),
			)
		}
		return nil

	case *ast.IdentifierExpression:
		switch n.GetKind() {
		case ast.NodeKindTrue, ast.NodeKindFalse:
			return types.TypeBool
		case ast.NodeKindNull:
			return ctxType
		}
	}

	element := r.lookupLiteralExpression(node, ctxFlow, ctxType, reportMode)
	if element == nil {
		return nil
	}
	typ := r.GetTypeOfElement(element)
	if typ == nil && reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeExpressionCannotBeRepresentedByAType,
			node.GetRange(),
		)
	}
	return typ
}

// lookupCallExpression looks up the program element a call expression refers to.
func (r *Resolver) lookupCallExpression(
	node *ast.CallExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	if ctxType == nil {
		ctxType = types.TypeVoid
	}
	typ := r.resolveCallExpression(node, ctxFlow, ctxType, reportMode)
	if typ == nil {
		return nil
	}
	element := r.GetElementOfType(typ)
	if element == nil && reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeType0IsIllegalInThisContext,
			node.GetRange(),
			typ.String(),
		)
	}
	return element
}

// resolveCallExpression resolves a call expression to its static type.
func (r *Resolver) resolveCallExpression(
	node *ast.CallExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	if ctxType == nil {
		ctxType = types.TypeVoid
	}

	targetExpression := node.Expression
	target := r.LookupExpression(targetExpression, ctxFlow, ctxType, reportMode)
	if target == nil {
		return nil
	}

	if target.GetElementKind() == ElementKindFunctionPrototype {
		functionPrototype := target.(*FunctionPrototype)
		if functionPrototype.GetInternalName() == "unchecked" && len(node.Args) > 0 {
			return r.ResolveExpression(node.Args[0], ctxFlow, ctxType, reportMode)
		}
		if ctxFlow == nil {
			return nil
		}
		functionInstance := r.MaybeInferCall(node, functionPrototype, ctxFlow, reportMode)
		if functionInstance == nil {
			return nil
		}
		target = functionInstance
	}

	if target.GetElementKind() == ElementKindFunction {
		return target.(*Function).Signature.ReturnType
	}

	if target.GetElementKind() == ElementKindPropertyPrototype {
		propertyInstance := r.ResolveProperty(target.(*PropertyPrototype), reportMode)
		if propertyInstance == nil {
			return nil
		}
		target = propertyInstance
	}

	if typedElement, ok := target.(TypedElement); ok {
		if targetElement := r.GetElementOfType(typedElement.GetResolvedType()); targetElement != nil {
			target = targetElement
		}
	}

	if target.GetElementKind() == ElementKindClass {
		typeArguments := target.(*Class).GetTypeArgumentsTo(r.program.FunctionPrototype())
		if len(typeArguments) > 0 {
			if signature := typeArguments[0].GetSignature(); signature != nil {
				return signature.ReturnType
			}
		}
	}

	if reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeCannotInvokeAnExpressionWhoseTypeLacksACallSignatureType0HasNoCompatibleCallSignatures,
			targetExpression.GetRange(),
			target.GetInternalName(),
		)
	}
	return nil
}

// lookupCommaExpression looks up the program element a comma expression refers to.
func (r *Resolver) lookupCommaExpression(
	node *ast.CommaExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	if len(node.Expressions) == 0 {
		return nil
	}
	return r.LookupExpression(node.Expressions[len(node.Expressions)-1], ctxFlow, ctxType, reportMode)
}

// resolveCommaExpression resolves a comma expression to its static type.
func (r *Resolver) resolveCommaExpression(
	node *ast.CommaExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	if len(node.Expressions) == 0 {
		return nil
	}
	return r.ResolveExpression(node.Expressions[len(node.Expressions)-1], ctxFlow, ctxType, reportMode)
}

// lookupInstanceOfExpression looks up the program element an instanceof expression refers to.
func (r *Resolver) lookupInstanceOfExpression(
	node *ast.InstanceOfExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	_ = node
	_ = ctxFlow
	_ = ctxType
	_ = reportMode
	return r.GetElementOfType(types.TypeBool)
}

// resolveInstanceOfExpression resolves an instanceof expression to its static type.
func (r *Resolver) resolveInstanceOfExpression(
	node *ast.InstanceOfExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	_ = node
	_ = ctxFlow
	_ = ctxType
	_ = reportMode
	return types.TypeBool
}

// lookupTernaryExpression looks up the program element a ternary expression refers to.
func (r *Resolver) lookupTernaryExpression(
	node *ast.TernaryExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	typ := r.resolveTernaryExpression(node, ctxFlow, ctxType, reportMode)
	if typ == nil {
		return nil
	}
	element := r.GetElementOfType(typ)
	if element == nil && reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeType0IsIllegalInThisContext,
			node.GetRange(),
			typ.String(),
		)
	}
	return element
}

// resolveTernaryExpression resolves a ternary expression to its static type.
func (r *Resolver) resolveTernaryExpression(
	node *ast.TernaryExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	thenType := r.ResolveExpression(node.IfThen, ctxFlow, ctxType, reportMode)
	if thenType == nil {
		return nil
	}
	elseType := r.ResolveExpression(node.IfElse, ctxFlow, thenType, reportMode)
	if elseType == nil {
		return nil
	}
	commonType := types.CommonType(thenType, elseType, ctxType, false)
	if commonType == nil && reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeOperator0CannotBeAppliedToTypes1And2,
			node.GetRange(),
			"?:",
			thenType.String(),
			elseType.String(),
		)
	}
	return commonType
}

// lookupNewExpression looks up the program element a new expression refers to.
func (r *Resolver) lookupNewExpression(
	node *ast.NewExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	_ = ctxType
	if ctxFlow == nil {
		return nil
	}
	sourceFunction, ok := ctxFlow.SourceFunction().(Element)
	if !ok {
		return nil
	}
	element := r.ResolveTypeName(node.TypeName, ctxFlow, sourceFunction, reportMode)
	if element == nil {
		return nil
	}
	if element.GetElementKind() == ElementKindClassPrototype {
		ctxTypes := util.CloneMap(ctxFlow.ContextualTypeArguments())
		if ctxTypes == nil {
			ctxTypes = make(map[string]*types.Type)
		}
		return r.ResolveClassInclTypeArguments(
			element.(*ClassPrototype),
			node.TypeArguments,
			ctxFlow,
			sourceFunction,
			ctxTypes,
			node,
			reportMode,
		)
	}
	if reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeThisExpressionIsNotConstructable,
			node.GetRange(),
		)
	}
	return nil
}

// resolveNewExpression resolves a new expression to its static type.
func (r *Resolver) resolveNewExpression(
	node *ast.NewExpression,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	element := r.lookupNewExpression(node, ctxFlow, ctxType, reportMode)
	if element == nil {
		return nil
	}
	typ := r.GetTypeOfElement(element)
	if typ == nil && reportMode == ReportModeReport {
		r.program.Error(
			diagnostics.DiagnosticCodeExpressionCannotBeRepresentedByAType,
			node.GetRange(),
		)
	}
	return typ
}

// ResolveExpression resolves the type of an expression.
// Ported from: assemblyscript/src/resolver.ts resolveExpression (lines 2270-2560).
func (r *Resolver) ResolveExpression(
	node ast.Node,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	if node == nil {
		return nil
	}
	switch node.GetKind() {
	case ast.NodeKindIdentifier:
		ident := node.(*ast.IdentifierExpression)
		if ctxFlow != nil {
			local := ctxFlow.LookupLocal(ident.Text)
			if local != nil {
				return local.GetType()
			}
		}
		element := r.lookupIdentifierExpression(ident, ctxFlow, reportMode)
		if element != nil {
			return r.GetTypeOfElement(element)
		}
		return nil

	case ast.NodeKindThis:
		return r.resolveThisExpression(node.(*ast.IdentifierExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindSuper:
		return r.resolveSuperExpression(node.(*ast.IdentifierExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindTrue, ast.NodeKindFalse, ast.NodeKindNull, ast.NodeKindLiteral:
		return r.resolveLiteralExpression(node, ctxFlow, ctxType, reportMode)

	case ast.NodeKindPropertyAccess:
		element := r.lookupPropertyAccessExpression(
			node.(*ast.PropertyAccessExpression), ctxFlow, ctxType, reportMode,
		)
		if element != nil {
			return r.GetTypeOfElement(element)
		}
		return nil

	case ast.NodeKindElementAccess:
		element := r.lookupElementAccessExpression(
			node.(*ast.ElementAccessExpression), ctxFlow, ctxType, reportMode,
		)
		if element != nil {
			return r.GetTypeOfElement(element)
		}
		return nil

	case ast.NodeKindCall:
		return r.resolveCallExpression(node.(*ast.CallExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindParenthesized:
		return r.ResolveExpression(
			node.(*ast.ParenthesizedExpression).Expression, ctxFlow, ctxType, reportMode,
		)

	case ast.NodeKindBinary:
		binExpr := node.(*ast.BinaryExpression)
		// Resolve based on left operand type (binary ops typically preserve left type)
		return r.ResolveExpression(binExpr.Left, ctxFlow, ctxType, reportMode)

	case ast.NodeKindUnaryPrefix:
		unary := node.(*ast.UnaryPrefixExpression)
		return r.ResolveExpression(unary.Operand, ctxFlow, ctxType, reportMode)

	case ast.NodeKindUnaryPostfix:
		return r.ResolveExpression(
			node.(*ast.UnaryPostfixExpression).Operand, ctxFlow, ctxType, reportMode,
		)

	case ast.NodeKindTernary:
		return r.resolveTernaryExpression(node.(*ast.TernaryExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindComma:
		return r.resolveCommaExpression(node.(*ast.CommaExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindInstanceOf:
		return r.resolveInstanceOfExpression(node.(*ast.InstanceOfExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindNew:
		return r.resolveNewExpression(node.(*ast.NewExpression), ctxFlow, ctxType, reportMode)

	case ast.NodeKindAssertion:
		assertion := node.(*ast.AssertionExpression)
		if assertion.ToType != nil {
			return r.ResolveType(
				assertion.ToType, ctxFlow, ctxFlow.TargetFunction.(Element), nil, reportMode,
			)
		}
		return r.ResolveExpression(assertion.Expression, ctxFlow, ctxType, reportMode)
	}

	// Default: try to resolve by context
	if ctxType != nil && ctxType != types.TypeVoid {
		return ctxType
	}
	return nil
}

// GetElementOfType gets the element corresponding to a type, if any.
func (r *Resolver) GetElementOfType(typ *types.Type) Element {
	if typ == nil {
		return nil
	}
	if classRef := typ.GetClass(); classRef != nil {
		if element, ok := classRef.(Element); ok {
			return element
		}
	}
	if signature := typ.GetSignature(); signature != nil {
		if functionPrototype := r.program.FunctionPrototype(); functionPrototype != nil {
			if functionClass := r.ResolveClass(functionPrototype, []*types.Type{signature.Type}, make(map[string]*types.Type), ReportModeSwallow); functionClass != nil {
				return functionClass
			}
		}
	}
	if wrapper, ok := r.program.WrapperClasses[typ]; ok {
		return wrapper
	}
	return nil
}

// GetTypeOfElement gets the concrete type of an element.
func (r *Resolver) GetTypeOfElement(element Element) *types.Type {
	kind := element.GetElementKind()
	if kind == ElementKindGlobal {
		g := element.(*Global)
		// TODO: check CommonFlagsLazy when that flag is ported
		if !r.ensureResolvedLazyGlobal(g, ReportModeSwallow) {
			return nil
		}
	}
	if IsTypedElement(kind) {
		te := element.(TypedElement)
		typ := te.GetResolvedType()
		if classRef := typ.GetClassOrWrapper(r.program); classRef != nil {
			if wrappedType := classRef.(*Class).WrappedType; wrappedType != nil {
				typ = wrappedType
			}
		}
		return typ
	}
	return nil
}

// ensureResolvedLazyGlobal resolves a lazy global if not yet resolved.
func (r *Resolver) ensureResolvedLazyGlobal(g *Global, reportMode ReportMode) bool {
	if g.Is(common.CommonFlagsResolved) {
		return true
	}
	typeNode := g.TypeNode()
	var resolved *types.Type
	if typeNode != nil {
		resolved = r.ResolveType(typeNode, nil, g.GetParent(), nil, reportMode)
	} else {
		initializer := g.InitializerNode()
		if initializer == nil {
			return false
		}
		resolved = r.ResolveExpression(initializer, g.File().StartFunction.Flow, types.TypeAuto, reportMode)
	}
	if resolved == nil {
		return false
	}
	g.SetType(resolved)
	return true
}

// EnsureOneTypeArgument verifies exactly one type argument is provided and resolves it.
func (r *Resolver) EnsureOneTypeArgument(
	typeArgumentNodes []ast.Node,
	f *flow.Flow,
	ctxElement Element,
	ctxTypes map[string]*types.Type,
	reportNode ast.Node,
	reportMode ReportMode,
) *types.Type {
	if len(typeArgumentNodes) != 1 {
		if reportMode == ReportModeReport {
			var rng *diagnostics.Range
			if len(typeArgumentNodes) > 0 {
				rng = typeArgumentNodes[0].GetRange()
			} else if reportNode != nil {
				rng = reportNode.GetRange()
			}
			r.program.Error(
				diagnostics.DiagnosticCodeExpected0TypeArgumentsButGot1,
				rng,
				"1",
				fmt.Sprintf("%d", len(typeArgumentNodes)),
			)
		}
		return nil
	}
	return r.ResolveType(typeArgumentNodes[0], f, ctxElement, ctxTypes, reportMode)
}

// ResolveOverrides resolves all override functions for a given function.
func (r *Resolver) ResolveOverrides(instance *Function) []*Function {
	prototype := instance.Prototype
	if prototype == nil {
		return nil
	}

	unboundOverrides := prototype.UnboundOverrides
	if unboundOverrides == nil || len(unboundOverrides) == 0 {
		return nil
	}

	var overrides []*Function
	for overridePrototype := range unboundOverrides {
		parent := overridePrototype.GetParent()
		if parent == nil {
			continue
		}
		if parent.GetElementKind() == ElementKindClass || parent.GetElementKind() == ElementKindInterface {
			cls := parent.(*Class)
			bound := overridePrototype.ToBound(cls)
			ctxTypes := make(map[string]*types.Type)
			for k, v := range cls.ContextualTypeArguments {
				ctxTypes[k] = v
			}
			overrideInstance := r.ResolveFunction(bound, instance.TypeArguments, ctxTypes, ReportModeSwallow)
			if overrideInstance != nil {
				overrides = append(overrides, overrideInstance)
			}
		}
	}

	return overrides
}

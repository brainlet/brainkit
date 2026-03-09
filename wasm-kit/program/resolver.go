package program

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/types"
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
	resolvedTypeArguments := r.ResolveTypeArguments(
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
	return r.ResolveFunction(prototype, resolvedTypeArguments, ctxTypes, reportMode)
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

	// Finish resolving if no pending bases
	if !anyPending {
		r.finishResolveClass(instance, reportMode)
		delete(r.resolveClassPending, instance)
	}

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
		if typeParameterNodes != nil {
			resolvedTypeArguments = r.ResolveTypeArguments(
				typeParameterNodes,
				typeArgumentNodes,
				f,
				ctxElement,
				ctxTypes,
				reportNode,
				reportMode,
			)
			if resolvedTypeArguments == nil {
				return nil
			}
		}
	}

	return r.ResolveClass(prototype, resolvedTypeArguments, ctxTypes, reportMode)
}

// finishResolveClass completes class resolution by resolving members and interface prototypes.
func (r *Resolver) finishResolveClass(instance *Class, reportMode ReportMode) {
	prototype := instance.Prototype

	// Resolve instance members
	members := prototype.GetMembers()
	if members != nil {
		for _, member := range members {
			r.resolveClassMember(instance, member, reportMode)
		}
	}

	// Resolve interface implementations if applicable
	interfacePrototypes := prototype.InterfacePrototypes
	if interfacePrototypes != nil {
		for _, ifaceProto := range interfacePrototypes {
			// Resolve each interface
			ctxTypes := make(map[string]*types.Type)
			for k, v := range instance.ContextualTypeArguments {
				ctxTypes[k] = v
			}
			resolved := r.ResolveClass(&ifaceProto.ClassPrototype, nil, ctxTypes, reportMode)
			if resolved != nil {
				iface := resolved.AsInterface()
				if iface != nil {
					instance.AddInterface(iface)
				}
			}
		}
	}

	// Now finish any derived classes that were pending on this base
	for pending := range r.resolveClassPending {
		if pending.Base == instance {
			r.finishResolveClass(pending, reportMode)
			delete(r.resolveClassPending, pending)
		}
	}
}

// resolveClassMember resolves a single class member within its class context.
func (r *Resolver) resolveClassMember(instance *Class, member DeclaredElement, reportMode ReportMode) {
	switch member.GetElementKind() {
	case ElementKindFunctionPrototype:
		fp := member.(*FunctionPrototype)
		// Bind to class instance
		bound := fp.ToBound(instance)
		// For non-generic instance methods, resolve immediately
		if !fp.Is(common.CommonFlagsGeneric) {
			ctxTypes := make(map[string]*types.Type)
			for k, v := range instance.ContextualTypeArguments {
				ctxTypes[k] = v
			}
			r.ResolveFunction(bound, nil, ctxTypes, reportMode)
		}

	case ElementKindPropertyPrototype:
		pp := member.(*PropertyPrototype)
		r.ResolveProperty(pp, reportMode)
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
		ctxTypes := make(map[string]*types.Type)
		parent := prototype.GetParent()
		if parent != nil && parent.GetElementKind() == ElementKindClass {
			cls := parent.(*Class)
			for k, v := range cls.ContextualTypeArguments {
				ctxTypes[k] = v
			}
		}
		getter := r.ResolveFunction(prototype.GetterPrototype, nil, ctxTypes, reportMode)
		if getter != nil {
			property.GetterInstance = getter
		}
	}

	// Resolve setter if present
	if prototype.SetterPrototype != nil {
		ctxTypes := make(map[string]*types.Type)
		parent := prototype.GetParent()
		if parent != nil && parent.GetElementKind() == ElementKindClass {
			cls := parent.(*Class)
			for k, v := range cls.ContextualTypeArguments {
				ctxTypes[k] = v
			}
		}
		setter := r.ResolveFunction(prototype.SetterPrototype, nil, ctxTypes, reportMode)
		if setter != nil {
			property.SetterInstance = setter
		}
	}

	return property
}

// =========================================================================
// Expression resolution (stubs for Phase 4)
// =========================================================================

// LookupExpression looks up the program element an expression refers to.
func (r *Resolver) LookupExpression(
	node ast.Node,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) Element {
	// Stub: will be implemented in Phase 4 (compiler)
	return nil
}

// ResolveExpression resolves the type of an expression.
func (r *Resolver) ResolveExpression(
	node ast.Node,
	ctxFlow *flow.Flow,
	ctxType *types.Type,
	reportMode ReportMode,
) *types.Type {
	// Stub: will be implemented in Phase 4 (compiler)
	return nil
}

// GetTypeOfElement gets the concrete type of an element.
func (r *Resolver) GetTypeOfElement(element Element) *types.Type {
	kind := element.GetElementKind()
	if kind == ElementKindGlobal {
		g := element.(*Global)
		// TODO: check CommonFlagsLazy when that flag is ported
		r.ensureResolvedLazyGlobal(g)
	}
	if IsTypedElement(kind) {
		te := element.(TypedElement)
		return te.GetResolvedType()
	}
	return nil
}

// ensureResolvedLazyGlobal resolves a lazy global if not yet resolved.
func (r *Resolver) ensureResolvedLazyGlobal(g *Global) bool {
	if g.Is(common.CommonFlagsResolved) {
		return true
	}
	// Stub: lazy global resolution requires expression resolution
	return false
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

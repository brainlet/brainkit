package program

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// PropertyPrototype represents an unresolved property (field or accessor pair).
type PropertyPrototype struct {
	DeclaredElementBase
	FieldDeclaration *ast.FieldDeclaration
	GetterPrototype  *FunctionPrototype
	SetterPrototype  *FunctionPrototype
	PropertyInstance *Property
	boundPrototypes  map[*Class]*PropertyPrototype
}

// NewPropertyPrototype creates a new property prototype.
func NewPropertyPrototype(name string, parent Element, firstDeclaration *ast.FunctionDeclaration) *PropertyPrototype {
	pp := &PropertyPrototype{}
	isInstance := (firstDeclaration.Flags & int32(common.CommonFlagsInstance)) != 0
	internalName := MangleInternalName(name, parent, isInstance, false)
	InitDeclaredElementBase(&pp.DeclaredElementBase, ElementKindPropertyPrototype, name, internalName, parent.GetProgram(), parent, firstDeclaration)
	// Clear Get/Set from flags since the prototype represents the property, not a specific accessor
	pp.flags &= ^(common.CommonFlagsGet | common.CommonFlagsSet)
	return pp
}

// PropertyPrototypeForField creates a property prototype representing a field.
// A field is a property with an attached memory offset. Implicit getter/setter
// prototypes are created so that fields and explicit accessors are interchangeable.
func PropertyPrototypeForField(name string, parent *ClassPrototype, fieldDeclaration *ast.FieldDeclaration, decoratorFlags DecoratorFlags) *PropertyPrototype {
	nativeRange := ast.NativeSource().GetRange()

	// Create getter declaration: get name(): type
	getterDeclaration := ast.NewFunctionDeclaration(
		fieldDeclaration.Name,
		fieldDeclaration.Decorators,
		fieldDeclaration.Flags|int32(common.CommonFlagsInstance|common.CommonFlagsGet),
		nil,
		ast.NewFunctionTypeNode(nil, ast.NewOmittedType(*nativeRange), nil, false, *nativeRange),
		nil,
		ast.ArrowKindNone,
		*nativeRange,
	)

	// Create setter declaration: set name(name: type)
	setterDeclaration := ast.NewFunctionDeclaration(
		fieldDeclaration.Name,
		fieldDeclaration.Decorators,
		fieldDeclaration.Flags|int32(common.CommonFlagsInstance|common.CommonFlagsSet),
		nil,
		ast.NewFunctionTypeNode(nil, ast.NewOmittedType(*nativeRange), nil, false, *nativeRange),
		nil,
		ast.ArrowKindNone,
		*nativeRange,
	)

	pp := NewPropertyPrototype(name, parent, getterDeclaration)
	pp.FieldDeclaration = fieldDeclaration
	pp.decoratorFlags = decoratorFlags
	pp.GetterPrototype = NewFunctionPrototype(common.GETTER_PREFIX+name, parent, getterDeclaration, decoratorFlags)
	pp.SetterPrototype = NewFunctionPrototype(common.SETTER_PREFIX+name, parent, setterDeclaration, decoratorFlags)
	return pp
}

// IsField tests if this property prototype represents a field.
func (pp *PropertyPrototype) IsField() bool {
	return pp.FieldDeclaration != nil
}

// PropertyTypeNode returns the associated type node.
func (pp *PropertyPrototype) PropertyTypeNode() ast.Node {
	if pp.FieldDeclaration != nil {
		return pp.FieldDeclaration.Type
	}
	if pp.GetterPrototype != nil {
		sig := pp.GetterPrototype.FunctionTypeNode()
		if sig != nil {
			return sig.ReturnType
		}
	}
	if pp.SetterPrototype != nil {
		sig := pp.SetterPrototype.FunctionTypeNode()
		if sig != nil && len(sig.Parameters) > 0 {
			return sig.Parameters[0].Type
		}
	}
	return nil
}

// PropertyInitializerNode returns the associated initializer expression.
func (pp *PropertyPrototype) PropertyInitializerNode() ast.Node {
	if pp.FieldDeclaration != nil {
		return pp.FieldDeclaration.Initializer
	}
	return nil
}

// PropertyParameterIndex returns the constructor parameter index, or -1.
func (pp *PropertyPrototype) PropertyParameterIndex() int32 {
	if pp.FieldDeclaration != nil {
		return pp.FieldDeclaration.ParameterIndex
	}
	return -1
}

// ThisType returns the resolved type of `this` for this property.
func (pp *PropertyPrototype) ThisType() *types.Type {
	parent := pp.GetParent()
	if c, ok := parent.(*Class); ok {
		return c.GetResolvedType()
	}
	return nil
}

// ToBound creates a clone bound to a concrete class instance.
func (pp *PropertyPrototype) ToBound(classInstance *Class) *PropertyPrototype {
	if pp.boundPrototypes == nil {
		pp.boundPrototypes = make(map[*Class]*PropertyPrototype)
	} else if bound, ok := pp.boundPrototypes[classInstance]; ok {
		return bound
	}
	firstDeclaration := pp.declaration.(*ast.FunctionDeclaration)
	bound := NewPropertyPrototype(pp.name, classInstance, firstDeclaration)
	bound.flags = pp.flags
	bound.FieldDeclaration = pp.FieldDeclaration
	if pp.GetterPrototype != nil {
		bound.GetterPrototype = pp.GetterPrototype.ToBound(classInstance)
	}
	if pp.SetterPrototype != nil {
		bound.SetterPrototype = pp.SetterPrototype.ToBound(classInstance)
	}
	pp.boundPrototypes[classInstance] = bound
	return bound
}

// --- FlowPropertyPrototypeRef implementation ---
//
// These adapter methods provide flow.FlowPropertyPrototypeRef compatibility.
// The flow package uses int32/uint32 and interface{} rather than named types
// to break circular dependencies between packages.
//
// Note: Full interface satisfaction requires return type compatibility between
// ElementKind/int32 and Element/FlowElementRef. These adapters follow the same
// pattern as Function's FlowFunctionRef adapters in function.go.

// Instance returns the resolved Property as a FlowPropertyRef.
func (pp *PropertyPrototype) Instance() flow.FlowPropertyRef {
	if pp.PropertyInstance == nil {
		return nil
	}
	return pp.PropertyInstance
}

// ParameterIndex returns the constructor parameter index, or -1.
func (pp *PropertyPrototype) ParameterIndex() int32 {
	return pp.PropertyParameterIndex()
}

// FlowGetParent returns the parent as a FlowElementRef for the flow package.
func (pp *PropertyPrototype) FlowGetParent() flow.FlowElementRef {
	parent := pp.GetParent()
	if parent == nil {
		return nil
	}
	if ref, ok := parent.(flow.FlowElementRef); ok {
		return ref
	}
	return nil
}

// Property represents a resolved property.
type Property struct {
	VariableLikeBase
	Prototype      *PropertyPrototype
	GetterInstance *Function
	SetterInstance *Function
	MemoryOffset   int32
}

// NewProperty creates a new resolved property.
func NewProperty(prototype *PropertyPrototype, parent Element) *Property {
	p := &Property{
		MemoryOffset: -1,
	}
	var declaration ast.Node
	if prototype.IsField() {
		declaration = prototype.FieldDeclaration
	} else {
		// Create a synthetic variable declaration for accessor-based properties
		ident := prototype.IdentifierNode()
		if ident != nil {
			declaration = parent.GetProgram().MakeNativeVariableDeclaration(ident.Text, 0)
		} else {
			declaration = parent.GetProgram().MakeNativeVariableDeclaration(prototype.GetName(), 0)
		}
	}
	InitVariableLikeBase(&p.VariableLikeBase, ElementKindProperty, prototype.GetName(), parent, declaration)
	p.Prototype = prototype
	p.flags = prototype.flags
	p.decoratorFlags = prototype.decoratorFlags
	if p.Is(common.CommonFlagsInstance) {
		RegisterConcreteElement(p.program, p)
	}
	return p
}

// IsField tests if this property represents a field.
func (p *Property) IsField() bool {
	return p.Prototype.IsField()
}

// TypeNode returns the associated type node. Overrides VariableLikeBase.TypeNode
// to handle FieldDeclaration in addition to VariableDeclaration.
func (p *Property) TypeNode() ast.Node {
	if fd, ok := p.declaration.(*ast.FieldDeclaration); ok {
		return fd.Type
	}
	return p.VariableLikeBase.TypeNode()
}

// InitializerNode returns the associated initializer node. Overrides VariableLikeBase.InitializerNode
// to handle FieldDeclaration in addition to VariableDeclaration.
func (p *Property) InitializerNode() ast.Node {
	if fd, ok := p.declaration.(*ast.FieldDeclaration); ok {
		return fd.Initializer
	}
	return p.VariableLikeBase.InitializerNode()
}

// CheckVisibility validates getter/setter visibility consistency.
func (p *Property) CheckVisibility(emitter *diagnostics.DiagnosticEmitter) {
	getter := p.GetterInstance
	setter := p.SetterInstance
	if getter != nil && setter != nil && !getter.VisibilityNoLessThan(setter) {
		getterIdent := getter.IdentifierNode()
		setterIdent := setter.IdentifierNode()
		if getterIdent != nil && setterIdent != nil {
			emitter.ErrorRelated(
				diagnostics.DiagnosticCodeGetAccessor0MustBeAtLeastAsAccessibleAsTheSetter,
				getterIdent.GetRange(),
				setterIdent.GetRange(),
				getterIdent.Text, "", "",
			)
		}
	}
}

// --- FlowPropertyRef implementation ---
//
// These adapter methods provide flow.FlowPropertyRef compatibility.
// The flow package uses interface{} for node references to avoid
// circular dependencies between packages.

// GetPrototype returns the prototype as a FlowPropertyPrototypeRef.
func (p *Property) GetPrototype() flow.FlowPropertyPrototypeRef {
	return p.Prototype
}

// FlowInitializerNode returns the initializer node as interface{} for the flow package.
// This adapts InitializerNode() ast.Node to the interface{} return type
// required by flow.FlowPropertyRef.
func (p *Property) FlowInitializerNode() interface{} {
	return p.InitializerNode()
}

// GetType returns the resolved type.
func (p *Property) GetType() *types.Type {
	return p.resolvedType
}

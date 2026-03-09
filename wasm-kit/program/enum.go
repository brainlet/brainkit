package program

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// Namespace represents a user-declared namespace.
type Namespace struct {
	DeclaredElementBase
}

// NewNamespace creates a new namespace element.
func NewNamespace(name string, parent Element, declaration *ast.NamespaceDeclaration, decoratorFlags DecoratorFlags) *Namespace {
	ns := &Namespace{}
	internalName := MangleInternalName(name, parent, false, false)
	InitDeclaredElementBase(&ns.DeclaredElementBase, ElementKindNamespace, name, internalName, parent.GetProgram(), parent, declaration)
	ns.decoratorFlags = decoratorFlags
	return ns
}

// Lookup looks up a member, then falls back to parent.
func (ns *Namespace) Lookup(name string, isType bool) Element {
	if member := ns.GetMember(name); member != nil {
		return member
	}
	return ns.ElementBase.Lookup(name, isType)
}

// Enum represents an enum type.
type Enum struct {
	TypedElementBase
	ToStringFunctionName string
}

// NewEnum creates a new enum element.
func NewEnum(name string, parent Element, declaration *ast.EnumDeclaration, decoratorFlags DecoratorFlags) *Enum {
	e := &Enum{}
	internalName := MangleInternalName(name, parent, false, false)
	InitTypedElementBase(&e.TypedElementBase, ElementKindEnum, name, internalName, parent.GetProgram(), parent, declaration)
	e.decoratorFlags = decoratorFlags
	e.SetType(types.TypeI32)
	return e
}

// Lookup looks up a member, then falls back to parent.
func (e *Enum) Lookup(name string, isType bool) Element {
	if member := e.GetMember(name); member != nil {
		return member
	}
	return e.ElementBase.Lookup(name, isType)
}

// EnumValue represents a value within an enum.
type EnumValue struct {
	VariableLikeBase
	IsImmutable bool
}

// NewEnumValue creates a new enum value.
func NewEnumValue(name string, parent *Enum, declaration *ast.EnumValueDeclaration, decoratorFlags DecoratorFlags) *EnumValue {
	ev := &EnumValue{}
	InitVariableLikeBase(&ev.VariableLikeBase, ElementKindEnumValue, name, parent, declaration)
	ev.decoratorFlags = decoratorFlags
	ev.SetType(types.TypeI32)
	return ev
}

// ValueNode returns the enum value's initializer expression.
func (ev *EnumValue) ValueNode() ast.Node {
	if decl, ok := ev.declaration.(*ast.EnumValueDeclaration); ok {
		return decl.Initializer
	}
	return nil
}

// Global represents a file-level global variable.
type Global struct {
	VariableLikeBase
}

// NewGlobal creates a new global variable.
func NewGlobal(name string, parent Element, decoratorFlags DecoratorFlags, declaration ast.Node) *Global {
	g := &Global{}
	if declaration == nil {
		declaration = parent.GetProgram().MakeNativeVariableDeclaration(name, 0)
	}
	InitVariableLikeBase(&g.VariableLikeBase, ElementKindGlobal, name, parent, declaration)
	g.decoratorFlags = decoratorFlags
	return g
}

// TypeDefinition represents a type alias (type X = Y).
type TypeDefinition struct {
	TypedElementBase
}

// NewTypeDefinition creates a new type definition.
func NewTypeDefinition(name string, parent Element, declaration *ast.TypeDeclaration, decoratorFlags DecoratorFlags) *TypeDefinition {
	td := &TypeDefinition{}
	internalName := MangleInternalName(name, parent, false, false)
	InitTypedElementBase(&td.TypedElementBase, ElementKindTypeDefinition, name, internalName, parent.GetProgram(), parent, declaration)
	td.decoratorFlags = decoratorFlags
	return td
}

// TypeParameterNodes returns the type parameter nodes.
func (td *TypeDefinition) TypeParameterNodes() []*ast.TypeParameterNode {
	if decl, ok := td.declaration.(*ast.TypeDeclaration); ok {
		return decl.TypeParameters
	}
	return nil
}

// TypeNode returns the aliased type node.
func (td *TypeDefinition) TypeNode() ast.Node {
	if decl, ok := td.declaration.(*ast.TypeDeclaration); ok {
		return decl.Type
	}
	return nil
}

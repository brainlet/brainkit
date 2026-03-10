package program

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// Element is the interface implemented by all program elements.
type Element interface {
	GetElementKind() ElementKind
	GetName() string
	GetInternalName() string
	SetInternalName(name string)
	GetProgram() *Program
	GetParent() Element
	SetParent(parent Element)
	GetFlags() common.CommonFlags
	SetFlags(flags common.CommonFlags)
	GetDecoratorFlags() DecoratorFlags
	SetDecoratorFlags(flags DecoratorFlags)
	GetMembers() map[string]DeclaredElement
	SetMembers(members map[string]DeclaredElement)
	GetShadowType() *TypeDefinition
	SetShadowType(td *TypeDefinition)

	// Flag operations
	Is(flag common.CommonFlags) bool
	IsAny(flags common.CommonFlags) bool
	Set(flag common.CommonFlags)
	Unset(flag common.CommonFlags)
	HasDecorator(flag DecoratorFlags) bool
	HasAnyDecorator(flags DecoratorFlags) bool

	// Member operations
	GetMember(name string) DeclaredElement
	Lookup(name string, isType bool) Element
	Add(name string, element DeclaredElement, localIdentifierIfImport *ast.IdentifierExpression) bool

	// Computed properties
	File() *File
	IsPublic() bool
	IsImplicitlyPublic() bool
	VisibilityEquals(other Element) bool
	VisibilityNoLessThan(other Element) bool
	IsBound() bool
	GetBoundClassOrInterface() *Class

	// String representation
	String() string
}

// DeclaredElement is an Element with an associated declaration.
type DeclaredElement interface {
	Element
	GetDeclaration() ast.Node
	SetDeclaration(decl ast.Node)
	IsDeclaredInLibrary() bool
	IdentifierNode() *ast.IdentifierExpression
	IdentifierAndSignatureRange() diagnostics.Range
	DecoratorNodes() []*ast.DecoratorNode
}

// TypedElement is an Element with a resolved type.
type TypedElement interface {
	DeclaredElement
	GetResolvedType() *types.Type
	SetResolvedType(t *types.Type)
}

// VariableLikeElement is a TypedElement with constant value support.
type VariableLikeElement interface {
	TypedElement
	GetConstantValueKind() ConstantValueKind
	GetConstantIntegerValue() int64
	GetConstantFloatValue() float64
	SetConstantIntegerValue(value int64, t *types.Type)
	SetConstantFloatValue(value float64, t *types.Type)
	TypeNode() ast.Node
	InitializerNode() ast.Node
}

// --- Base structs ---

// ElementBase provides the common fields and methods for all elements.
type ElementBase struct {
	kind           ElementKind
	name           string
	internalName   string
	program        *Program
	parent         Element
	flags          common.CommonFlags
	decoratorFlags DecoratorFlags
	members        map[string]DeclaredElement
	shadowType     *TypeDefinition
}

func (e *ElementBase) GetElementKind() ElementKind                   { return e.kind }
func (e *ElementBase) GetName() string                               { return e.name }
func (e *ElementBase) GetInternalName() string                       { return e.internalName }
func (e *ElementBase) SetInternalName(name string)                   { e.internalName = name }
func (e *ElementBase) GetProgram() *Program                          { return e.program }
func (e *ElementBase) GetParent() Element                            { return e.parent }
func (e *ElementBase) SetParent(parent Element)                      { e.parent = parent }
func (e *ElementBase) GetFlags() common.CommonFlags                  { return e.flags }
func (e *ElementBase) SetFlags(flags common.CommonFlags)             { e.flags = flags }
func (e *ElementBase) GetDecoratorFlags() DecoratorFlags             { return e.decoratorFlags }
func (e *ElementBase) SetDecoratorFlags(flags DecoratorFlags)        { e.decoratorFlags = flags }
func (e *ElementBase) GetMembers() map[string]DeclaredElement        { return e.members }
func (e *ElementBase) SetMembers(members map[string]DeclaredElement) { e.members = members }
func (e *ElementBase) GetShadowType() *TypeDefinition                { return e.shadowType }
func (e *ElementBase) SetShadowType(td *TypeDefinition)              { e.shadowType = td }

func (e *ElementBase) Is(flag common.CommonFlags) bool     { return (e.flags & flag) == flag }
func (e *ElementBase) IsAny(flags common.CommonFlags) bool { return (e.flags & flags) != 0 }
func (e *ElementBase) Set(flag common.CommonFlags)         { e.flags |= flag }
func (e *ElementBase) Unset(flag common.CommonFlags)       { e.flags &= ^flag }
func (e *ElementBase) HasDecorator(flag DecoratorFlags) bool {
	return (e.decoratorFlags & flag) == flag
}
func (e *ElementBase) HasAnyDecorator(flags DecoratorFlags) bool {
	return (e.decoratorFlags & flags) != 0
}

// GetMember returns the member with the given name, or nil.
func (e *ElementBase) GetMember(name string) DeclaredElement {
	if e.members != nil {
		if m, ok := e.members[name]; ok {
			return m
		}
	}
	return nil
}

// Lookup looks up an element by name relative to this element's parent.
func (e *ElementBase) Lookup(name string, isType bool) Element {
	return e.parent.Lookup(name, isType)
}

// Add adds an element as a member. Reports and returns false if duplicate.
func (e *ElementBase) Add(name string, element DeclaredElement, localIdentifierIfImport *ast.IdentifierExpression) bool {
	originalDeclaration := element.GetDeclaration()
	if e.members == nil {
		e.members = make(map[string]DeclaredElement)
	} else if existing, ok := e.members[name]; ok {
		if existing.GetParent() == e.parent {
			merged := TryMerge(existing, element)
			if merged != nil {
				element = merged
			} else {
				reportedIdentifier := localIdentifierIfImport
				if reportedIdentifier == nil {
					reportedIdentifier = element.IdentifierNode()
				}
				if IsDeclaredElement(existing.GetElementKind()) {
					e.program.ErrorRelated(
						diagnostics.DiagnosticCodeDuplicateIdentifier0,
						reportedIdentifier.GetRange(),
						existing.(DeclaredElement).IdentifierNode().GetRange(),
						reportedIdentifier.Text,
					)
				} else {
					e.program.Error(
						diagnostics.DiagnosticCodeDuplicateIdentifier0,
						reportedIdentifier.GetRange(),
						reportedIdentifier.Text,
					)
				}
				return false
			}
		}
		// else: override non-own element
	}
	e.members[name] = element
	prog := e.program
	if element.GetElementKind() != ElementKindFunctionPrototype || !element.IsBound() {
		prog.ElementsByNameMap[element.GetInternalName()] = element
		prog.ElementsByDeclaration[originalDeclaration] = element
	}
	return true
}

// File returns the enclosing file.
func (e *ElementBase) File() *File {
	current := e.parent
	for {
		if current.GetElementKind() == ElementKindFile {
			return current.(*File)
		}
		current = current.GetParent()
	}
}

// IsPublic checks if this element is public.
func (e *ElementBase) IsPublic() bool {
	return !e.IsAny(common.CommonFlagsPrivate | common.CommonFlagsProtected)
}

// IsImplicitlyPublic checks if this element is implicitly public.
func (e *ElementBase) IsImplicitlyPublic() bool {
	return e.IsPublic() && !e.Is(common.CommonFlagsPublic)
}

// VisibilityEquals checks if visibility matches another element.
func (e *ElementBase) VisibilityEquals(other Element) bool {
	if e.IsPublic() == other.IsPublic() {
		return true
	}
	const vis = common.CommonFlagsPrivate | common.CommonFlagsProtected
	return (e.flags & vis) == (other.GetFlags() & vis)
}

// VisibilityNoLessThan checks if visibility is no less than another element's.
func (e *ElementBase) VisibilityNoLessThan(other Element) bool {
	if e.IsPublic() {
		return true
	}
	if e.Is(common.CommonFlagsPrivate) {
		return other.Is(common.CommonFlagsPrivate)
	}
	if e.Is(common.CommonFlagsProtected) {
		return other.IsAny(common.CommonFlagsPrivate | common.CommonFlagsProtected)
	}
	return false
}

// IsBound tests if this element is bound to a class.
func (e *ElementBase) IsBound() bool {
	parent := e.parent
	if parent == nil {
		return false
	}
	switch parent.GetElementKind() {
	case ElementKindClass, ElementKindInterface:
		return true
	}
	return false
}

// GetBoundClassOrInterface gets the class/interface this is bound to.
func (e *ElementBase) GetBoundClassOrInterface() *Class {
	parent := e.parent
	if parent == nil {
		return nil
	}
	switch parent.GetElementKind() {
	case ElementKindClass, ElementKindInterface:
		return parent.(*Class)
	}
	return nil
}

// String returns a string representation.
func (e *ElementBase) String() string {
	return fmt.Sprintf("%s, kind=%d", e.internalName, e.kind)
}

// InitElementBase initializes the common fields of an ElementBase.
func InitElementBase(e *ElementBase, kind ElementKind, name, internalName string, prog *Program, parent Element) {
	e.kind = kind
	e.name = name
	e.internalName = internalName
	e.program = prog
	if parent != nil {
		e.parent = parent
	}
	// File has no parent (parent is self) — handled in File constructor
}

// --- DeclaredElementBase ---

// DeclaredElementBase provides common fields for declared elements.
type DeclaredElementBase struct {
	ElementBase
	declaration ast.Node // DeclarationStatement (any declaration node)
}

func (d *DeclaredElementBase) GetDeclaration() ast.Node     { return d.declaration }
func (d *DeclaredElementBase) SetDeclaration(decl ast.Node) { d.declaration = decl }

// IsDeclaredInLibrary tests if this element is declared in a library source.
func (d *DeclaredElementBase) IsDeclaredInLibrary() bool {
	r := d.declaration.GetRange()
	if r == nil {
		return false
	}
	if source, ok := r.Source.(interface{ IsLibrary() bool }); ok {
		return source.IsLibrary()
	}
	return false
}

// IdentifierNode returns the declaration's name identifier.
func (d *DeclaredElementBase) IdentifierNode() *ast.IdentifierExpression {
	switch decl := d.declaration.(type) {
	case *ast.ClassDeclaration:
		return decl.Name
	case *ast.FunctionDeclaration:
		return decl.Name
	case *ast.FieldDeclaration:
		return decl.Name
	case *ast.EnumDeclaration:
		return decl.Name
	case *ast.NamespaceDeclaration:
		return decl.Name
	case *ast.VariableDeclaration:
		return decl.Name
	case *ast.TypeDeclaration:
		return decl.Name
	case *ast.EnumValueDeclaration:
		return decl.Name
	}
	// Fallback for any declaration with a DeclarationBase
	return nil
}

// IdentifierAndSignatureRange returns the range covering both identifier and signature.
func (d *DeclaredElementBase) IdentifierAndSignatureRange() diagnostics.Range {
	ident := d.IdentifierNode()
	if ident == nil {
		return *d.declaration.GetRange()
	}
	if fn, ok := d.declaration.(*ast.FunctionDeclaration); ok && fn.Signature != nil {
		identRange := ident.GetRange()
		signatureRange := fn.Signature.GetRange()
		if identRange != nil && signatureRange != nil && identRange.Source == signatureRange.Source {
			if joined := diagnostics.JoinRanges(identRange, signatureRange); joined != nil {
				return *joined
			}
		}
	}
	return *ident.GetRange()
}

// DecoratorNodes returns the declaration's decorator nodes.
func (d *DeclaredElementBase) DecoratorNodes() []*ast.DecoratorNode {
	switch decl := d.declaration.(type) {
	case *ast.ClassDeclaration:
		return decl.Decorators
	case *ast.FunctionDeclaration:
		return decl.Decorators
	case *ast.FieldDeclaration:
		return decl.Decorators
	case *ast.EnumDeclaration:
		return decl.Decorators
	case *ast.NamespaceDeclaration:
		return decl.Decorators
	case *ast.VariableDeclaration:
		return decl.Decorators
	case *ast.TypeDeclaration:
		return decl.Decorators
	}
	return nil
}

// InitDeclaredElementBase initializes a DeclaredElementBase.
func InitDeclaredElementBase(d *DeclaredElementBase, kind ElementKind, name, internalName string, prog *Program, parent Element, declaration ast.Node) {
	InitElementBase(&d.ElementBase, kind, name, internalName, prog, parent)
	d.declaration = declaration
	RegisterDeclaredElementKind(kind)
	// Inherit flags from declaration
	if decl, ok := declaration.(*ast.ClassDeclaration); ok {
		d.flags = common.CommonFlags(decl.Flags)
	} else if decl, ok := declaration.(*ast.FunctionDeclaration); ok {
		d.flags = common.CommonFlags(decl.Flags)
	} else if decl, ok := declaration.(*ast.FieldDeclaration); ok {
		d.flags = common.CommonFlags(decl.Flags)
	} else if decl, ok := declaration.(*ast.EnumDeclaration); ok {
		d.flags = common.CommonFlags(decl.Flags)
	} else if decl, ok := declaration.(*ast.NamespaceDeclaration); ok {
		d.flags = common.CommonFlags(decl.Flags)
	} else if decl, ok := declaration.(*ast.VariableDeclaration); ok {
		d.flags = common.CommonFlags(decl.Flags)
	} else if decl, ok := declaration.(*ast.TypeDeclaration); ok {
		d.flags = common.CommonFlags(decl.Flags)
	} else if decl, ok := declaration.(*ast.EnumValueDeclaration); ok {
		d.flags = common.CommonFlags(decl.Flags)
	}
}

// --- TypedElementBase ---

// TypedElementBase provides common fields for typed elements.
type TypedElementBase struct {
	DeclaredElementBase
	resolvedType *types.Type
}

func (t *TypedElementBase) GetResolvedType() *types.Type    { return t.resolvedType }
func (t *TypedElementBase) SetResolvedType(typ *types.Type) { t.resolvedType = typ }

// SetType sets the resolved type. Panics if already resolved.
func (t *TypedElementBase) SetType(typ *types.Type) {
	if t.Is(common.CommonFlagsResolved) {
		panic("element already resolved")
	}
	t.resolvedType = typ
	t.Set(common.CommonFlagsResolved)
}

// InitTypedElementBase initializes a TypedElementBase.
func InitTypedElementBase(t *TypedElementBase, kind ElementKind, name, internalName string, prog *Program, parent Element, declaration ast.Node) {
	InitDeclaredElementBase(&t.DeclaredElementBase, kind, name, internalName, prog, parent, declaration)
	RegisterTypedElementKind(kind)
	t.resolvedType = types.TypeVoid
}

// --- VariableLikeBase ---

// VariableLikeBase provides common fields for variable-like elements.
type VariableLikeBase struct {
	TypedElementBase
	constantValueKind    ConstantValueKind
	constantIntegerValue int64
	constantFloatValue   float64
}

func (v *VariableLikeBase) GetConstantValueKind() ConstantValueKind { return v.constantValueKind }
func (v *VariableLikeBase) GetConstantIntegerValue() int64          { return v.constantIntegerValue }
func (v *VariableLikeBase) GetConstantFloatValue() float64          { return v.constantFloatValue }

// SetConstantIntegerValue applies a constant integer value.
func (v *VariableLikeBase) SetConstantIntegerValue(value int64, t *types.Type) {
	v.resolvedType = t
	v.constantValueKind = ConstantValueKindInteger
	v.constantIntegerValue = value
	v.Set(common.CommonFlagsConst | common.CommonFlagsInlined | common.CommonFlagsResolved)
}

// SetConstantFloatValue applies a constant float value.
func (v *VariableLikeBase) SetConstantFloatValue(value float64, t *types.Type) {
	v.resolvedType = t
	v.constantValueKind = ConstantValueKindFloat
	v.constantFloatValue = value
	v.Set(common.CommonFlagsConst | common.CommonFlagsInlined | common.CommonFlagsResolved)
}

// TypeNode returns the associated type node.
func (v *VariableLikeBase) TypeNode() ast.Node {
	switch decl := v.declaration.(type) {
	case *ast.VariableDeclaration:
		return decl.Type
	}
	return nil
}

// InitializerNode returns the associated initializer node.
func (v *VariableLikeBase) InitializerNode() ast.Node {
	switch decl := v.declaration.(type) {
	case *ast.VariableDeclaration:
		return decl.Initializer
	}
	return nil
}

// InitVariableLikeBase initializes a VariableLikeBase.
func InitVariableLikeBase(v *VariableLikeBase, kind ElementKind, name string, parent Element, declaration ast.Node) {
	isInstance := false
	if declaration != nil {
		if decl, ok := declaration.(*ast.VariableDeclaration); ok {
			isInstance = (decl.Flags & int32(common.CommonFlagsInstance)) != 0
		}
	}
	internalName := MangleInternalName(name, parent, isInstance, false)
	InitTypedElementBase(&v.TypedElementBase, kind, name, internalName, parent.GetProgram(), parent, declaration)
	if declaration != nil {
		if decl, ok := declaration.(*ast.VariableDeclaration); ok {
			v.flags = common.CommonFlags(decl.Flags)
		}
	}
}

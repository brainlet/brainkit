package program

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// ClassPrototype represents an unresolved/generic class.
type ClassPrototype struct {
	DeclaredElementBase
	InstanceMembers            map[string]DeclaredElement
	InstanceMemberOrder        []string
	BasePrototype              *ClassPrototype
	InterfacePrototypes        []*InterfacePrototype
	ConstructorPrototype       *FunctionPrototype
	OperatorOverloadPrototypes map[OperatorKind]*FunctionPrototype
	Instances                  map[string]*Class
	Extenders                  map[*ClassPrototype]struct{}
	ImplicitlyExtendsObject    bool
}

// NewClassPrototype creates a new class prototype.
func NewClassPrototype(name string, parent Element, declaration *ast.ClassDeclaration, decoratorFlags DecoratorFlags, isInterface bool) *ClassPrototype {
	cp := &ClassPrototype{
		OperatorOverloadPrototypes: make(map[OperatorKind]*FunctionPrototype),
		Extenders:                  make(map[*ClassPrototype]struct{}),
	}
	kind := ElementKindClassPrototype
	if isInterface {
		kind = ElementKindInterfacePrototype
	}
	isInstance := (declaration.Flags & int32(common.CommonFlagsInstance)) != 0
	internalName := MangleInternalName(name, parent, isInstance, false)
	InitDeclaredElementBase(&cp.DeclaredElementBase, kind, name, internalName, parent.GetProgram(), parent, declaration)
	cp.decoratorFlags = decoratorFlags
	return cp
}

// TypeParameterNodes returns the type parameter nodes.
func (cp *ClassPrototype) TypeParameterNodes() []*ast.TypeParameterNode {
	if decl, ok := cp.declaration.(*ast.ClassDeclaration); ok {
		return decl.TypeParameters
	}
	return nil
}

// ExtendsNode returns the extends type node.
func (cp *ClassPrototype) ExtendsNode() *ast.NamedTypeNode {
	if decl, ok := cp.declaration.(*ast.ClassDeclaration); ok {
		return decl.ExtendsType
	}
	return nil
}

// ImplementsNodes returns the implements type nodes.
func (cp *ClassPrototype) ImplementsNodes() []*ast.NamedTypeNode {
	if decl, ok := cp.declaration.(*ast.ClassDeclaration); ok {
		return decl.ImplementsTypes
	}
	return nil
}

// IsBuiltinArray tests if this prototype is of a builtin array type.
func (cp *ClassPrototype) IsBuiltinArray() bool {
	prog := cp.GetProgram()
	if prog == nil {
		return false
	}
	abvInstance := prog.ArrayBufferViewInstance()
	if abvInstance == nil {
		return false
	}
	return cp.Extends(abvInstance.Prototype)
}

// Extends tests if this prototype extends the specified base, with cycle detection.
func (cp *ClassPrototype) Extends(basePrototype *ClassPrototype) bool {
	current := cp
	seen := make(map[*ClassPrototype]struct{})
	for current != nil {
		if _, ok := seen[current]; ok {
			break
		}
		seen[current] = struct{}{}
		if current == basePrototype {
			return true
		}
		current = current.BasePrototype
	}
	return false
}

// AddInstance adds an element as an instance member.
func (cp *ClassPrototype) AddInstance(name string, element DeclaredElement) bool {
	originalDeclaration := element.GetDeclaration()
	hadExisting := false
	if cp.InstanceMembers == nil {
		cp.InstanceMembers = make(map[string]DeclaredElement)
	} else if existing, ok := cp.InstanceMembers[name]; ok {
		hadExisting = true
		merged := TryMerge(existing, element)
		if merged == nil {
			if IsDeclaredElement(existing.GetElementKind()) {
				cp.program.ErrorRelated(
					diagnostics.DiagnosticCodeDuplicateIdentifier0,
					element.IdentifierNode().GetRange(),
					existing.(DeclaredElement).IdentifierNode().GetRange(),
					element.IdentifierNode().Text,
				)
			} else {
				cp.program.Error(
					diagnostics.DiagnosticCodeDuplicateIdentifier0,
					element.IdentifierNode().GetRange(),
					element.IdentifierNode().Text,
				)
			}
			return false
		}
		element = merged
	}
	cp.InstanceMembers[name] = element
	if !hadExisting {
		cp.InstanceMemberOrder = append(cp.InstanceMemberOrder, name)
	}
	if element.Is(common.CommonFlagsExport) && cp.Is(common.CommonFlagsModuleExport) {
		element.Set(common.CommonFlagsModuleExport)
	}
	cp.program.ElementsByDeclaration[originalDeclaration] = element
	return true
}

// GetResolvedInstance gets the resolved class instance for the given key.
func (cp *ClassPrototype) GetResolvedInstance(instanceKey string) *Class {
	if cp.Instances != nil {
		if c, ok := cp.Instances[instanceKey]; ok {
			return c
		}
	}
	return nil
}

// SetResolvedInstance sets the resolved instance for the given key.
func (cp *ClassPrototype) SetResolvedInstance(instanceKey string, instance *Class) {
	if cp.Instances == nil {
		cp.Instances = make(map[string]*Class)
	}
	cp.Instances[instanceKey] = instance
}

// Class represents a resolved concrete class.
type Class struct {
	TypedElementBase
	Prototype                   *ClassPrototype
	TypeArguments               []*types.Type
	Base                        *Class
	Interfaces                  map[*Interface]struct{}
	ContextualTypeArguments     map[string]*types.Type
	NextMemoryOffset            uint32
	ConstructorInstance         *Function
	OperatorOverloads           map[OperatorKind]*Function
	IndexSignature_             *IndexSignature
	id                          uint32
	RttiFlags                   uint32
	WrappedType                 *types.Type
	Extenders                   map[*Class]struct{}
	Implementers                map[*Class]struct{}
	DidCheckFieldInitialization bool
	VisitRef                    FunctionRef
	interfaceRef                *Interface // set if this Class is embedded in an Interface
}

// NewClass creates a new resolved class.
func NewClass(nameInclTypeParameters string, prototype *ClassPrototype, typeArguments []*types.Type, isInterface bool) *Class {
	c := &Class{}
	kind := ElementKindClass
	if isInterface {
		kind = ElementKindInterface
	}
	isInstance := prototype.Is(common.CommonFlagsInstance)
	internalName := MangleInternalName(nameInclTypeParameters, prototype.GetParent(), isInstance, false)
	InitTypedElementBase(&c.TypedElementBase, kind, nameInclTypeParameters, internalName, prototype.GetProgram(), prototype.GetParent(), prototype.GetDeclaration())
	c.Prototype = prototype
	c.flags = prototype.flags
	c.decoratorFlags = prototype.decoratorFlags
	c.TypeArguments = typeArguments

	prog := c.program
	usizeType := prog.Options.UsizeType()
	typ := types.NewType(usizeType.Kind, usizeType.Flags&^types.TypeFlagValue|types.TypeFlagReference, usizeType.Size)
	typ.ClassRef = c
	c.SetType(typ)

	if !c.HasDecorator(DecoratorFlagsUnmanaged) {
		id := prog.NextClassId
		prog.NextClassId++
		c.id = id
		prog.ManagedClasses[int32(id)] = c
	}

	// Apply contextual type arguments
	typeParameters := prototype.TypeParameterNodes()
	if typeArguments != nil {
		numTypeArguments := len(typeArguments)
		if typeParameters == nil || numTypeArguments != len(typeParameters) {
			panic("type argument count mismatch")
		}
		if numTypeArguments > 0 {
			if c.ContextualTypeArguments == nil {
				c.ContextualTypeArguments = make(map[string]*types.Type)
			}
			for i := 0; i < numTypeArguments; i++ {
				c.ContextualTypeArguments[typeParameters[i].Name.Text] = typeArguments[i]
			}
		}
	} else if typeParameters != nil && len(typeParameters) > 0 {
		panic("type argument count mismatch")
	}
	RegisterConcreteElement(prog, c)
	return c
}

// Id returns the unique runtime id.
func (c *Class) Id() uint32 {
	return c.id
}

// IsInterface tests if this is an interface.
func (c *Class) IsInterface() bool {
	return c.kind == ElementKindInterface
}

// AsInterface returns the Interface wrapper, or nil if not an interface.
func (c *Class) AsInterface() *Interface {
	return c.interfaceRef
}

// SetInterfaceRef sets the interface reference for this class.
func (c *Class) SetInterfaceRef(iface *Interface) {
	c.interfaceRef = iface
}

// IsBuiltinArray tests if this is a builtin array type.
func (c *Class) IsBuiltinArray() bool {
	return c.Prototype.IsBuiltinArray()
}

// IsArrayLike tests if this class is array-like.
func (c *Class) IsArrayLike() bool {
	if c.IsBuiltinArray() {
		return true
	}
	lengthField := c.GetMember("length")
	if lengthField == nil {
		return false
	}
	hasLength := false
	if lengthField.GetElementKind() == ElementKindProperty {
		if p, ok := lengthField.(*Property); ok && p.GetterInstance != nil {
			hasLength = true
		}
	} else if lengthField.GetElementKind() == ElementKindPropertyPrototype {
		if pp, ok := lengthField.(*PropertyPrototype); ok && pp.GetterPrototype != nil {
			hasLength = true
		}
	}
	if !hasLength {
		return false
	}
	return c.FindOverload(OperatorKindIndexedGet, false) != nil ||
		c.FindOverload(OperatorKindUncheckedIndexedGet, false) != nil
}

// OrderedOwnMembers returns this class's own members in declaration order.
// This mirrors AssemblyScript's ordered Map traversal for concrete class members.
func (c *Class) OrderedOwnMembers() []DeclaredElement {
	members := c.GetMembers()
	if members == nil {
		return nil
	}

	order := c.Prototype.InstanceMemberOrder
	if len(order) == 0 {
		result := make([]DeclaredElement, 0, len(members))
		for _, member := range members {
			if member.GetParent() == c {
				result = append(result, member)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	}

	result := make([]DeclaredElement, 0, len(order))
	for _, name := range order {
		member, ok := members[name]
		if !ok || member.GetParent() != c {
			continue
		}
		result = append(result, member)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// LeastUpperBound computes the least upper bound of two class types.
func LeastUpperBound(a, b *Class) *Class {
	if a == b {
		return a
	}
	candidates := make(map[*Class]struct{})
	candidates[a] = struct{}{}
	candidates[b] = struct{}{}
	for {
		aBase := a.Base
		bBase := b.Base
		if aBase == nil && bBase == nil {
			return nil
		}
		if aBase != nil {
			if _, ok := candidates[aBase]; ok {
				return aBase
			}
			candidates[aBase] = struct{}{}
			a = aBase
		}
		if bBase != nil {
			if _, ok := candidates[bBase]; ok {
				return bBase
			}
			candidates[bBase] = struct{}{}
			b = bBase
		}
	}
}

// SetBase sets the base class and propagates relationships.
func (c *Class) SetBase(base *Class) {
	c.Base = base

	// Inherit contextual type arguments from base
	if base.ContextualTypeArguments != nil {
		if c.ContextualTypeArguments == nil {
			c.ContextualTypeArguments = make(map[string]*types.Type)
		}
		for baseName, baseType := range base.ContextualTypeArguments {
			if _, exists := c.ContextualTypeArguments[baseName]; !exists {
				c.ContextualTypeArguments[baseName] = baseType
			}
		}
	}

	// Propagate extenders up
	base.propagateExtenderUp(c)
	if c.Extenders != nil {
		for extender := range c.Extenders {
			base.propagateExtenderUp(extender)
		}
	}

	// Propagate interfaces down
	nextBase := base
	for nextBase != nil {
		if nextBase.Interfaces != nil {
			for iface := range nextBase.Interfaces {
				c.propagateInterfaceDown(iface)
			}
		}
		nextBase = nextBase.Base
	}
}

func (c *Class) propagateExtenderUp(extender *Class) {
	nextBase := c
	for nextBase != nil {
		if nextBase.Extenders == nil {
			nextBase.Extenders = make(map[*Class]struct{})
		}
		nextBase.Extenders[extender] = struct{}{}
		nextBase = nextBase.Base
	}
}

func (c *Class) propagateInterfaceDown(iface *Interface) {
	nextIface := iface
	for nextIface != nil {
		if nextIface.Implementers == nil {
			nextIface.Implementers = make(map[*Class]struct{})
		}
		nextIface.Implementers[c] = struct{}{}
		if c.Extenders != nil {
			for extender := range c.Extenders {
				nextIface.Implementers[extender] = struct{}{}
			}
		}
		if nextIface.Base != nil {
			if ifc := nextIface.Base.AsInterface(); ifc != nil {
				nextIface = ifc
			} else {
				break
			}
		} else {
			break
		}
	}
}

// AddInterface adds an implemented interface.
func (c *Class) AddInterface(iface *Interface) {
	if c.Interfaces == nil {
		c.Interfaces = make(map[*Interface]struct{})
	}
	c.Interfaces[iface] = struct{}{}
	c.propagateInterfaceDown(iface)
}

// InternalName satisfies types.ClassReference interface.
func (c *Class) InternalName() string {
	return c.GetInternalName()
}

// GetType satisfies types.ClassReference interface.
func (c *Class) GetType() *types.Type {
	return c.GetResolvedType()
}

// IsAssignableTo tests if this class is assignable to a target.
// Satisfies types.ClassReference interface.
func (c *Class) IsAssignableTo(target types.ClassReference) bool {
	t := target.(*Class)
	if t.IsInterface() {
		if c.IsInterface() {
			return c == t || c.ExtendsClass(t)
		}
		return c.ImplementsInterface(t.AsInterface())
	}
	if c.IsInterface() {
		return t == c.program.ObjectInstance()
	}
	return c == t || c.ExtendsClass(t)
}

// HasSubclassAssignableTo tests if any subclass is assignable to target.
// Satisfies types.ClassReference interface.
func (c *Class) HasSubclassAssignableTo(target types.ClassReference) bool {
	t := target.(*Class)
	if t.IsInterface() {
		if c.IsInterface() {
			return c.HasImplementerImplementing(t.AsInterface())
		}
		return c.HasExtenderImplementing(t.AsInterface())
	}
	if c.IsInterface() {
		return c.HasImplementer(t)
	}
	return c.HasExtender(t)
}

// FindOverload looks up an operator overload with unchecked fallback.
func (c *Class) FindOverload(kind OperatorKind, unchecked bool) *Function {
	if unchecked {
		switch kind {
		case OperatorKindIndexedGet:
			if overload := c.FindOverload(OperatorKindUncheckedIndexedGet, false); overload != nil {
				return overload
			}
		case OperatorKindIndexedSet:
			if overload := c.FindOverload(OperatorKindUncheckedIndexedSet, false); overload != nil {
				return overload
			}
		}
	}
	instance := c
	for instance != nil {
		if instance.OperatorOverloads != nil {
			if f, ok := instance.OperatorOverloads[kind]; ok {
				return f
			}
		}
		instance = instance.Base
	}
	return nil
}

// SetWrappedType satisfies types.ClassReference interface.
func (c *Class) SetWrappedType(t *types.Type) {
	c.WrappedType = t
}

// LookupOverload satisfies types.ClassReference interface.
func (c *Class) LookupOverload(kind int32) types.FunctionReference {
	result := c.FindOverload(kind, false)
	if result == nil {
		return nil
	}
	return result
}

// GetMethod resolves a method by name.
func (c *Class) GetMethod(name string, typeArguments []*types.Type) *Function {
	member := c.GetMember(name)
	if member != nil && member.GetElementKind() == ElementKindFunctionPrototype {
		if ResolveFunction != nil {
			return ResolveFunction(c.program.Resolver_, member.(*FunctionPrototype), typeArguments)
		}
	}
	return nil
}

// Offsetof returns the memory offset of a field.
func (c *Class) Offsetof(fieldName string) uint32 {
	member := c.GetMember(fieldName)
	if member == nil {
		return 0
	}
	if pp, ok := member.(*PropertyPrototype); ok {
		if pp.PropertyInstance != nil && pp.PropertyInstance.IsField() && pp.PropertyInstance.MemoryOffset >= 0 {
			return uint32(pp.PropertyInstance.MemoryOffset)
		}
	}
	return 0
}

// ExtendsPrototype tests if this class extends the specified prototype.
func (c *Class) ExtendsPrototype(prototype *ClassPrototype) bool {
	return c.Prototype.Extends(prototype)
}

// GetTypeArgumentsTo gets the type arguments to the specified extended prototype.
func (c *Class) GetTypeArgumentsTo(extendedPrototype *ClassPrototype) []*types.Type {
	current := c
	for current != nil {
		if current.Prototype == extendedPrototype {
			return current.TypeArguments
		}
		current = current.Base
	}
	return nil
}

// GetArrayValueType gets the value type of an array. Must be an array.
func (c *Class) GetArrayValueType() *types.Type {
	current := c
	prog := c.program

	arrayPrototype := prog.ArrayPrototype()
	if c.ExtendsPrototype(arrayPrototype) {
		typeArguments := c.GetTypeArgumentsTo(arrayPrototype)
		if len(typeArguments) > 0 {
			return typeArguments[0]
		}
	}

	staticArrayPrototype := prog.StaticArrayPrototype()
	if c.ExtendsPrototype(staticArrayPrototype) {
		typeArguments := c.GetTypeArgumentsTo(staticArrayPrototype)
		if len(typeArguments) > 0 {
			return typeArguments[0]
		}
	}

	abvInstance := prog.ArrayBufferViewInstance()
	for current.Base != abvInstance {
		if current.Base == nil {
			panic("array buffer view base not found")
		}
		current = current.Base
	}

	prototype := current.Prototype
	switch {
	case prototype == prog.Float32ArrayPrototype():
		return types.TypeF32
	case prototype == prog.Float64ArrayPrototype():
		return types.TypeF64
	case prototype == prog.Int8ArrayPrototype():
		return types.TypeI8
	case prototype == prog.Int16ArrayPrototype():
		return types.TypeI16
	case prototype == prog.Int32ArrayPrototype():
		return types.TypeI32
	case prototype == prog.Int64ArrayPrototype():
		return types.TypeI64
	case prototype == prog.Uint8ArrayPrototype():
		return types.TypeU8
	case prototype == prog.Uint8ClampedArrayPrototype():
		return types.TypeU8
	case prototype == prog.Uint16ArrayPrototype():
		return types.TypeU16
	case prototype == prog.Uint32ArrayPrototype():
		return types.TypeU32
	case prototype == prog.Uint64ArrayPrototype():
		return types.TypeU64
	default:
		panic("unknown array value type")
	}
}

// IsPointerfree tests if this class is pointerfree. Useful for the GC.
func (c *Class) IsPointerfree() bool {
	prog := c.program
	instanceMembers := c.GetMembers()
	if instanceMembers != nil {
		for _, member := range instanceMembers {
			if prototype, ok := member.(*PropertyPrototype); ok {
				property := prototype.PropertyInstance
				if property == nil {
					continue
				}
				if property.IsField() && property.GetResolvedType().IsManaged() {
					return false
				}
			}
		}

		if _, ok := instanceMembers[common.CommonNameVisit]; ok {
			prototype := c.Prototype
			if prototype == prog.ArrayPrototype() ||
				prototype == prog.StaticArrayPrototype() ||
				prototype == prog.SetPrototype() ||
				prototype == prog.MapPrototype() {
				typeArguments := c.GetTypeArgumentsTo(prototype)
				for _, typeArgument := range typeArguments {
					if typeArgument.IsManaged() {
						return false
					}
				}
				return true
			}
			return false
		}
	}
	return true
}

// ExtendsClass tests if this class extends another class.
func (c *Class) ExtendsClass(other *Class) bool {
	return other.HasExtender(c)
}

// HasExtender tests if this class has a direct or indirect extender.
func (c *Class) HasExtender(other *Class) bool {
	return c.Extenders != nil && hasKey(c.Extenders, other)
}

// HasExtenderImplementing tests if any extender implements the given interface.
func (c *Class) HasExtenderImplementing(other *Interface) bool {
	if c.Extenders != nil {
		for extender := range c.Extenders {
			if extender.ImplementsInterface(other) {
				return true
			}
		}
	}
	return false
}

// ImplementsInterface tests if this class implements an interface.
func (c *Class) ImplementsInterface(other *Interface) bool {
	return other.HasImplementer(c)
}

// HasImplementer tests if this interface has a direct or indirect implementer.
func (c *Class) HasImplementer(other *Class) bool {
	return c.Implementers != nil && hasKey(c.Implementers, other)
}

// HasImplementerImplementing tests if any implementer implements the given interface.
func (c *Class) HasImplementerImplementing(other *Interface) bool {
	if c.Implementers != nil {
		for implementer := range c.Implementers {
			if implementer.ImplementsInterface(other) {
				return true
			}
		}
	}
	return false
}

func hasKey(m map[*Class]struct{}, key *Class) bool {
	_, ok := m[key]
	return ok
}

// InterfacePrototype represents an unresolved interface.
type InterfacePrototype struct {
	ClassPrototype
}

// NewInterfacePrototype creates a new interface prototype.
func NewInterfacePrototype(name string, parent Element, declaration *ast.ClassDeclaration, decoratorFlags DecoratorFlags) *InterfacePrototype {
	ip := &InterfacePrototype{}
	ip.ClassPrototype = *NewClassPrototype(name, parent, declaration, decoratorFlags, true)
	return ip
}

// Interface represents a resolved interface.
type Interface struct {
	Class
}

// NewInterface creates a new resolved interface.
func NewInterface(nameInclTypeParameters string, prototype *InterfacePrototype, typeArguments []*types.Type) *Interface {
	iface := &Interface{}
	iface.Class = *NewClass(nameInclTypeParameters, &prototype.ClassPrototype, typeArguments, true)
	iface.Class.interfaceRef = iface
	return iface
}

// IndexSignature represents a class index signature.
type IndexSignature struct {
	TypedElementBase
}

// NewIndexSignature creates a new index signature.
func NewIndexSignature(parent *Class) *IndexSignature {
	is := &IndexSignature{}
	declaration := parent.GetProgram().MakeNativeVariableDeclaration("[]", 0)
	InitTypedElementBase(&is.TypedElementBase, ElementKindIndexSignature, "[]", parent.GetInternalName()+"[]", parent.GetProgram(), parent, declaration)
	return is
}

// GetGetterInstance returns the getter overload.
func (is *IndexSignature) GetGetterInstance(isUnchecked bool) *Function {
	return is.GetParent().(*Class).FindOverload(OperatorKindIndexedGet, isUnchecked)
}

// GetSetterInstance returns the setter overload.
func (is *IndexSignature) GetSetterInstance(isUnchecked bool) *Function {
	return is.GetParent().(*Class).FindOverload(OperatorKindIndexedSet, isUnchecked)
}

// --- FlowClassRef implementation ---

// FlowMembers returns members as a map of FlowElementRef for the flow package.
func (c *Class) FlowMembers() map[string]flow.FlowElementRef {
	members := c.GetMembers()
	if members == nil {
		return nil
	}
	result := make(map[string]flow.FlowElementRef, len(members))
	for name, member := range members {
		result[name] = member
	}
	return result
}

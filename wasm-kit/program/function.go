package program

import (
	"fmt"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// Parameter represents a function parameter descriptor.
type Parameter struct {
	Name        string
	Type        *types.Type
	Initializer ast.Node // Expression, may be nil
}

// Local represents a function-scoped local variable.
type Local struct {
	VariableLikeBase
	Index        int32
	OriginalName string
}

// NewLocal creates a new local variable.
func NewLocal(name string, index int32, typ *types.Type, parent *Function, declaration ast.Node) *Local {
	l := &Local{}
	if declaration == nil {
		declaration = parent.GetProgram().MakeNativeVariableDeclaration(name, 0)
	}
	InitVariableLikeBase(&l.VariableLikeBase, ElementKindLocal, name, parent, declaration)
	l.OriginalName = name
	l.Index = index
	if typ == types.TypeVoid {
		panic("local type cannot be void")
	}
	l.SetType(typ)
	return l
}

// DeclaredByFlow tests if this local was declared by the given flow's target function.
func (l *Local) DeclaredByFlow(f *flow.Flow) bool {
	return l.GetParent() == f.TargetFunction.(Element)
}

// FunctionPrototype represents an unresolved/generic function.
type FunctionPrototype struct {
	DeclaredElementBase
	OperatorKind     OperatorKind
	Instances        map[string]*Function
	UnboundOverrides map[*FunctionPrototype]struct{}
	boundPrototypes  map[*Class]*FunctionPrototype
}

// NewFunctionPrototype creates a new function prototype.
func NewFunctionPrototype(name string, parent Element, declaration *ast.FunctionDeclaration, decoratorFlags DecoratorFlags) *FunctionPrototype {
	fp := &FunctionPrototype{}
	isInstance := (declaration.Flags & int32(common.CommonFlagsInstance)) != 0
	internalName := MangleInternalName(name, parent, isInstance, false)
	InitDeclaredElementBase(&fp.DeclaredElementBase, ElementKindFunctionPrototype, name, internalName, parent.GetProgram(), parent, declaration)
	fp.decoratorFlags = decoratorFlags
	return fp
}

// TypeParameterNodes returns the type parameter nodes.
func (fp *FunctionPrototype) TypeParameterNodes() []*ast.TypeParameterNode {
	if decl, ok := fp.declaration.(*ast.FunctionDeclaration); ok {
		return decl.TypeParameters
	}
	return nil
}

// FunctionTypeNode returns the function's type (signature) node.
func (fp *FunctionPrototype) FunctionTypeNode() *ast.FunctionTypeNode {
	if decl, ok := fp.declaration.(*ast.FunctionDeclaration); ok {
		return decl.Signature
	}
	return nil
}

// BodyNode returns the function body.
func (fp *FunctionPrototype) BodyNode() ast.Node {
	if decl, ok := fp.declaration.(*ast.FunctionDeclaration); ok {
		return decl.Body
	}
	return nil
}

// ArrowKind returns the arrow function kind.
func (fp *FunctionPrototype) ArrowKind() ast.ArrowKind {
	if decl, ok := fp.declaration.(*ast.FunctionDeclaration); ok {
		return decl.ArrowKind
	}
	return ast.ArrowKindNone
}

// ToBound creates a clone bound to a concrete class instance.
func (fp *FunctionPrototype) ToBound(classInstance *Class) *FunctionPrototype {
	if fp.boundPrototypes == nil {
		fp.boundPrototypes = make(map[*Class]*FunctionPrototype)
	} else if bound, ok := fp.boundPrototypes[classInstance]; ok {
		return bound
	}
	declaration := fp.declaration.(*ast.FunctionDeclaration)
	bound := NewFunctionPrototype(fp.name, classInstance, declaration, fp.decoratorFlags)
	bound.flags = fp.flags
	bound.OperatorKind = fp.OperatorKind
	bound.UnboundOverrides = fp.UnboundOverrides
	fp.boundPrototypes[classInstance] = bound
	return bound
}

// GetResolvedInstance gets the resolved instance for the given key.
func (fp *FunctionPrototype) GetResolvedInstance(instanceKey string) *Function {
	if fp.Instances != nil {
		if f, ok := fp.Instances[instanceKey]; ok {
			return f
		}
	}
	return nil
}

// SetResolvedInstance sets the resolved instance for the given key.
func (fp *FunctionPrototype) SetResolvedInstance(instanceKey string, instance *Function) {
	if fp.Instances == nil {
		fp.Instances = make(map[string]*Function)
	}
	fp.Instances[instanceKey] = instance
}

// Function represents a resolved concrete function.
type Function struct {
	TypedElementBase
	Prototype               *FunctionPrototype
	Signature               *types.Signature
	LocalsByIndex           []*Local
	TypeArguments           []*types.Type
	ContextualTypeArguments map[string]*types.Type
	Flow                    *flow.Flow
	DebugLocations          map[ExpressionRef]*diagnostics.Range
	Ref                     FunctionRef
	VarargsStub             *Function
	OverrideStub            *Function
	MemorySegment           *MemorySegment
	Original                *Function
	NextInlineId            int32
	NextAnonymousId         int32
	NextBreakId             int32
	BreakStack              []int32
}

// NewFunction creates a new resolved function.
func NewFunction(
	nameInclTypeParameters string,
	prototype *FunctionPrototype,
	typeArguments []*types.Type,
	signature *types.Signature,
	contextualTypeArguments map[string]*types.Type,
) *Function {
	f := &Function{
		DebugLocations: make(map[ExpressionRef]*diagnostics.Range),
	}
	isInstance := prototype.Is(common.CommonFlagsInstance)
	internalName := MangleInternalName(nameInclTypeParameters, prototype.GetParent(), isInstance, false)
	InitTypedElementBase(&f.TypedElementBase, ElementKindFunction, nameInclTypeParameters, internalName, prototype.GetProgram(), prototype.GetParent(), prototype.GetDeclaration())
	f.Prototype = prototype
	f.TypeArguments = typeArguments
	f.Signature = signature
	f.flags = prototype.flags | common.CommonFlagsResolved
	f.decoratorFlags = prototype.decoratorFlags
	f.ContextualTypeArguments = contextualTypeArguments
	f.Original = f
	f.resolvedType = signature.Type

	// Create default flow
	f.Flow = flow.CreateDefault(f)

	// Add this and parameter locals if not ambient
	if !prototype.Is(common.CommonFlagsAmbient) {
		localIndex := int32(0)
		thisType := signature.ThisType
		if thisType != nil {
			local := NewLocal(common.CommonNameThis, localIndex, thisType, f, nil)
			localIndex++
			scopedLocals := f.Flow.ScopedLocals
			if scopedLocals == nil {
				scopedLocals = make(map[string]flow.FlowLocalRef)
				f.Flow.ScopedLocals = scopedLocals
			}
			scopedLocals[common.CommonNameThis] = local
			if int(local.Index) >= len(f.LocalsByIndex) {
				newLocals := make([]*Local, local.Index+1)
				copy(newLocals, f.LocalsByIndex)
				f.LocalsByIndex = newLocals
			}
			f.LocalsByIndex[local.Index] = local
			f.Flow.SetLocalFlag(local.Index, flow.LocalFlagInitialized)
		}
		parameterTypes := signature.ParameterTypes
		for i, parameterType := range parameterTypes {
			parameterName := f.GetParameterName(int32(i))
			local := NewLocal(parameterName, localIndex, parameterType, f, nil)
			localIndex++
			scopedLocals := f.Flow.ScopedLocals
			if scopedLocals == nil {
				scopedLocals = make(map[string]flow.FlowLocalRef)
				f.Flow.ScopedLocals = scopedLocals
			}
			scopedLocals[parameterName] = local
			if int(local.Index) >= len(f.LocalsByIndex) {
				newLocals := make([]*Local, local.Index+1)
				copy(newLocals, f.LocalsByIndex)
				f.LocalsByIndex = newLocals
			}
			f.LocalsByIndex[local.Index] = local
			f.Flow.SetLocalFlag(local.Index, flow.LocalFlagInitialized)
		}
	}
	RegisterConcreteElement(prototype.GetProgram(), f)
	return f
}

// GetNonParameterLocalTypes returns the types of additional locals that are not parameters.
func (f *Function) GetNonParameterLocalTypes() []*types.Type {
	numTotal := len(f.LocalsByIndex)
	numFixed := len(f.Signature.ParameterTypes)
	if f.Signature.ThisType != nil {
		numFixed++
	}
	numAdditional := numTotal - numFixed
	result := make([]*types.Type, numAdditional)
	for i := 0; i < numAdditional; i++ {
		result[i] = f.LocalsByIndex[numFixed+i].resolvedType
	}
	return result
}

// GetParameterName returns the name of the parameter at the given index.
func (f *Function) GetParameterName(index int32) string {
	if decl, ok := f.declaration.(*ast.FunctionDeclaration); ok {
		if decl.Signature != nil && int(index) < len(decl.Signature.Parameters) {
			return decl.Signature.Parameters[index].Name.Text
		}
	}
	return GetDefaultParameterName(index)
}

// NewStub creates a stub function for varargs or override calls.
func (f *Function) NewStub(postfix string, requiredParameters int32) *Function {
	if requiredParameters < 0 {
		requiredParameters = f.Signature.RequiredParameters
	}
	stub := NewFunction(
		f.Original.name+common.STUB_DELIMITER+postfix,
		f.Prototype,
		f.TypeArguments,
		f.Signature.Clone(requiredParameters, f.Signature.HasRest),
		f.ContextualTypeArguments,
	)
	stub.Original = f.Original
	stub.Set(f.flags & ^common.CommonFlagsCompiled | common.CommonFlagsStub)
	return stub
}

// AddLocal adds a local variable of the specified type.
func (f *Function) AddLocal(typ *types.Type, name string, declaration ast.Node) *Local {
	localIndex := int32(len(f.LocalsByIndex))
	localName := name
	if localName == "" {
		localName = fmt.Sprintf("%d", localIndex)
	}
	if declaration == nil {
		declaration = f.program.MakeNativeVariableDeclaration(localName, 0)
	}
	local := NewLocal(localName, localIndex, typ, f, declaration)
	if name != "" {
		defaultFlow := f.Flow
		scopedLocals := defaultFlow.ScopedLocals
		if scopedLocals == nil {
			scopedLocals = make(map[string]flow.FlowLocalRef)
			defaultFlow.ScopedLocals = scopedLocals
		}
		if _, exists := scopedLocals[name]; exists {
			panic("duplicate local name")
		}
		scopedLocals[name] = local
	}
	f.LocalsByIndex = append(f.LocalsByIndex, local)
	return local
}

// Lookup looks up an element by name.
func (f *Function) Lookup(name string, isType bool) Element {
	if !isType {
		scopedLocals := f.Flow.ScopedLocals
		if scopedLocals != nil {
			if local, ok := scopedLocals[name]; ok {
				return local.(Element)
			}
		}
	}
	return f.ElementBase.Lookup(name, isType)
}

// Finalize finalizes the function after compilation.
func (f *Function) Finalize(module *Module, ref FunctionRef) {
	f.Ref = ref
	f.BreakStack = nil
	f.AddDebugInfo(module, ref)
}

// AddDebugInfo adds debug info to the compiled function.
// Sets debug locations from DebugLocations map and local variable names
// from LocalsByIndex when the respective compiler options are enabled.
func (f *Function) AddDebugInfo(module *Module, ref FunctionRef) {
	opts := f.program.Options
	if opts.SourceMap {
		for exprRef, r := range f.DebugLocations {
			source, ok := r.Source.(*ast.Source)
			if !ok {
				continue
			}
			line := source.LineAt(r.Start)
			col := source.ColumnAt() - 1 // source maps are 0-based
			ModuleSetDebugLocation(module, ref, exprRef, source.DebugInfoIndex, line, col)
		}
	}
	if opts.DebugInfo {
		localNameSet := make(map[string]struct{})
		for i, local := range f.LocalsByIndex {
			name := local.GetName()
			if _, exists := localNameSet[name]; exists {
				name = fmt.Sprintf("%s|%d", name, i)
			}
			localNameSet[name] = struct{}{}
			ModuleSetLocalName(module, ref, int32(i), name)
		}
	}
}

// --- FlowFunctionRef implementation for flow package ---
//
// ElementBase.Is/Set/GetElementKind satisfy FlowFunctionRef/FlowElementRef
// via type aliases (CommonFlags = uint32, ElementKind = int32).

func (f *Function) FlowAddLocal(typ *types.Type) flow.FlowLocalRef {
	return f.AddLocal(typ, "", nil)
}
func (f *Function) FlowLocalsByIndex() []flow.FlowLocalRef {
	result := make([]flow.FlowLocalRef, len(f.LocalsByIndex))
	for i, l := range f.LocalsByIndex {
		result[i] = l
	}
	return result
}
func (f *Function) FlowTruncateLocals(n int) {
	if n < len(f.LocalsByIndex) {
		f.LocalsByIndex = f.LocalsByIndex[:n]
	}
}
func (f *Function) FlowInternalName() string { return f.internalName }
func (f *Function) AllocBreakId() int32 {
	id := f.NextBreakId
	f.NextBreakId++
	return id
}
func (f *Function) PushBreakStack(id int32) {
	f.BreakStack = append(f.BreakStack, id)
}
func (f *Function) PopBreakStack() int32 {
	n := len(f.BreakStack)
	id := f.BreakStack[n-1]
	f.BreakStack = f.BreakStack[:n-1]
	return id
}
func (f *Function) AllocInlineId() int32 {
	id := f.NextInlineId
	f.NextInlineId++
	return id
}
func (f *Function) FlowParent() flow.FlowElementRef {
	parent := f.GetParent()
	if parent == nil {
		return nil
	}
	if ref, ok := parent.(flow.FlowElementRef); ok {
		return ref
	}
	return nil
}
func (f *Function) FlowProgram() flow.FlowProgramRef {
	return f.program.FlowProgramRef()
}
func (f *Function) FlowSignature() *types.Signature {
	return f.Signature
}
func (f *Function) FlowContextualTypeArguments() map[string]*types.Type {
	return f.ContextualTypeArguments
}
func (f *Function) FlowLookup(name string) flow.FlowElementRef {
	elem := f.Lookup(name, false)
	if elem == nil {
		return nil
	}
	if ref, ok := elem.(flow.FlowElementRef); ok {
		return ref
	}
	return nil
}
func (f *Function) GetFlow() *flow.Flow {
	return f.Flow
}

// --- FlowLocalRef implementation for Local ---
//
// ElementBase provides: GetElementKind() int32, GetName() string,
// SetInternalName(string), Set(uint32) — all satisfied via type aliases.

func (l *Local) FlowIndex() int32     { return l.Index }
func (l *Local) GetType() *types.Type { return l.resolvedType }
func (l *Local) SetName(name string)  { l.name = name }
func (l *Local) FlowGetParent() flow.FlowElementRef {
	parent := l.GetParent()
	if parent == nil {
		return nil
	}
	if ref, ok := parent.(flow.FlowElementRef); ok {
		return ref
	}
	return nil
}
func (l *Local) DeclarationRange() interface{} {
	return l.declaration.GetRange()
}
func (l *Local) DeclarationNameRange() interface{} {
	ident := l.IdentifierNode()
	if ident != nil {
		return ident.GetRange()
	}
	return l.declaration.GetRange()
}
func (l *Local) DeclarationIsNative() bool {
	if l.declaration == nil {
		return false
	}
	rng := l.declaration.GetRange()
	if rng == nil || rng.Source == nil {
		return false
	}
	if src, ok := rng.Source.(*ast.Source); ok {
		return src.IsNative()
	}
	return false
}

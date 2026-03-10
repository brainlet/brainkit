package flow

import (
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/types"
)

// --- Interface stubs for circular dependency breaking ---

// ExpressionRef is an opaque reference to a Binaryen expression.
type ExpressionRef = uintptr

// ElementKind constants. Set by program package at init.
var (
	ElementKindFunction          int32
	ElementKindClass             int32
	ElementKindPropertyPrototype int32
	ElementKindGlobal            int32
	ElementKindEnumValue         int32
)

// FlowFunctionRef represents a Function from the program package.
// Methods prefixed with "Flow" avoid name conflicts with program.Element methods.
type FlowFunctionRef interface {
	FlowElementRef
	Is(flags uint32) bool
	Set(flags uint32)
	FlowAddLocal(typ *types.Type) FlowLocalRef
	FlowLocalsByIndex() []FlowLocalRef
	FlowTruncateLocals(n int)
	FlowInternalName() string
	AllocBreakId() int32
	PushBreakStack(id int32)
	PopBreakStack() int32
	AllocInlineId() int32
	FlowParent() FlowElementRef
	FlowProgram() FlowProgramRef
	FlowSignature() *types.Signature
	FlowContextualTypeArguments() map[string]*types.Type
	FlowLookup(name string) FlowElementRef
	GetFlow() *Flow
}

// FlowLocalRef represents a Local from the program package.
// FlowIndex/FlowGetParent prefixed to avoid conflicts with program.Element fields/methods.
type FlowLocalRef interface {
	FlowElementRef
	FlowIndex() int32
	GetType() *types.Type
	GetName() string
	SetName(name string)
	SetInternalName(name string)
	FlowGetParent() FlowElementRef
	Set(flags uint32)
	DeclarationRange() interface{} // *diagnostics.Range
	DeclarationNameRange() interface{}
	DeclarationIsNative() bool
}

// FlowElementRef represents an Element from the program package.
type FlowElementRef interface {
	GetElementKind() int32
}

// FlowTypedElementRef represents a TypedElement from the program package.
type FlowTypedElementRef interface {
	FlowElementRef
	GetType() *types.Type
}

// FlowPropertyRef represents a Property from the program package.
type FlowPropertyRef interface {
	IsField() bool
	GetPrototype() FlowPropertyPrototypeRef
	FlowInitializerNode() interface{}
	GetType() *types.Type
}

// FlowPropertyPrototypeRef represents a PropertyPrototype from the program package.
type FlowPropertyPrototypeRef interface {
	FlowElementRef
	Instance() FlowPropertyRef
	FlowGetParent() FlowElementRef
	ParameterIndex() int32
}

// FlowClassRef represents a Class from the program package (used in initThisFieldFlags).
type FlowClassRef interface {
	FlowElementRef
	FlowMembers() map[string]FlowElementRef
}

// FlowTypeDefinitionRef represents a TypeDefinition from the program package.
type FlowTypeDefinitionRef interface{}

// FlowProgramRef represents the Program from the program package.
type FlowProgramRef interface {
	UncheckedBehaviorAlways() bool
	Error(code int32, rng interface{}, args ...string)
	ErrorRelated(code int32, rng1 interface{}, rng2 interface{}, args ...string)
	ElementsByName() map[string]FlowElementRef
	InstancesByName() map[string]FlowElementRef
}

// --- Function variables set by other packages at init ---

// NewLocalFunc creates a new Local. Set by program package at init.
var NewLocalFunc func(name string, index int32, typ *types.Type, parent FlowFunctionRef) FlowLocalRef

// MangleInternalNameFunc mangles a name. Set by program package at init.
var MangleInternalNameFunc func(name string, parent FlowElementRef, asGlobal bool) string

// CommonFlagsConstructor is the Constructor flag from common. Set by common package or program at init.
var CommonFlagsConstructor uint32

// CommonFlagsScoped is the Scoped flag from common. Set by common package or program at init.
var CommonFlagsScoped uint32

// DiagnosticCodeCannotRedeclare is the code for block-scoped redeclaration. Set at init.
var DiagnosticCodeCannotRedeclare int32

// DiagnosticCodeDuplicateIdentifier is the code for duplicate identifiers. Set at init.
var DiagnosticCodeDuplicateIdentifier int32

// --- Binaryen IR inspection function variables (set by module package at init) ---
// These are used by canOverflow, inheritNonnullIfTrue/False, isNonnull.
// Until module package is ported, these are nil and the methods use safe defaults.

var (
	GetExpressionId    func(expr ExpressionRef) int32
	GetLocalGetIndex   func(expr ExpressionRef) int32
	IsLocalTee         func(expr ExpressionRef) bool
	GetLocalSetValue   func(expr ExpressionRef) ExpressionRef
	GetLocalSetIndex   func(expr ExpressionRef) int32
	GetGlobalGetName   func(expr ExpressionRef) string
	GetBinaryOp        func(expr ExpressionRef) int32
	GetBinaryLeft      func(expr ExpressionRef) ExpressionRef
	GetBinaryRight     func(expr ExpressionRef) ExpressionRef
	GetUnaryOp         func(expr ExpressionRef) int32
	GetUnaryValue      func(expr ExpressionRef) ExpressionRef
	GetExpressionType  func(expr ExpressionRef) int32
	GetConstValueI32   func(expr ExpressionRef) int32
	GetConstValueI64Low func(expr ExpressionRef) int32
	GetConstValueF32   func(expr ExpressionRef) float32
	GetConstValueF64   func(expr ExpressionRef) float64
	GetLoadBytes       func(expr ExpressionRef) int32
	IsLoadSigned       func(expr ExpressionRef) bool
	GetBlockName       func(expr ExpressionRef) string
	GetBlockChildCount func(expr ExpressionRef) int32
	GetBlockChildAt    func(expr ExpressionRef, index int32) ExpressionRef
	GetIfCondition     func(expr ExpressionRef) ExpressionRef
	GetIfTrue          func(expr ExpressionRef) ExpressionRef
	GetIfFalse         func(expr ExpressionRef) ExpressionRef
	GetSelectThen      func(expr ExpressionRef) ExpressionRef
	GetSelectElse      func(expr ExpressionRef) ExpressionRef
	GetCallTarget      func(expr ExpressionRef) string
	GetCallOperandAt   func(expr ExpressionRef, index int32) ExpressionRef
	GetCallOperandCount func(expr ExpressionRef) int32
	IsConstZero        func(expr ExpressionRef) bool
	IsConstNonZero     func(expr ExpressionRef) bool
)

// Binaryen ExpressionId constants. Set by module package at init.
var (
	ExpressionIdLocalGet    int32
	ExpressionIdLocalSet    int32
	ExpressionIdGlobalGet   int32
	ExpressionIdBinary      int32
	ExpressionIdUnary       int32
	ExpressionIdConst       int32
	ExpressionIdLoad        int32
	ExpressionIdBlock       int32
	ExpressionIdIf          int32
	ExpressionIdSelect      int32
	ExpressionIdCall        int32
	ExpressionIdUnreachable int32
)

// Binaryen BinaryOp constants. Set by module package at init.
var (
	BinaryOpEqI32  int32
	BinaryOpEqI64  int32
	BinaryOpEqF32  int32
	BinaryOpEqF64  int32
	BinaryOpNeI32  int32
	BinaryOpNeI64  int32
	BinaryOpNeF32  int32
	BinaryOpNeF64  int32
	BinaryOpLtI32  int32
	BinaryOpLtU32  int32
	BinaryOpLtI64  int32
	BinaryOpLtU64  int32
	BinaryOpLtF32  int32
	BinaryOpLtF64  int32
	BinaryOpLeI32  int32
	BinaryOpLeU32  int32
	BinaryOpLeI64  int32
	BinaryOpLeU64  int32
	BinaryOpLeF32  int32
	BinaryOpLeF64  int32
	BinaryOpGtI32  int32
	BinaryOpGtU32  int32
	BinaryOpGtI64  int32
	BinaryOpGtU64  int32
	BinaryOpGtF32  int32
	BinaryOpGtF64  int32
	BinaryOpGeI32  int32
	BinaryOpGeU32  int32
	BinaryOpGeI64  int32
	BinaryOpGeU64  int32
	BinaryOpGeF32  int32
	BinaryOpGeF64  int32
	BinaryOpMulI32 int32
	BinaryOpAndI32 int32
	BinaryOpShlI32 int32
	BinaryOpShrI32 int32
	BinaryOpShrU32 int32
	BinaryOpDivU32 int32
	BinaryOpRemI32 int32
	BinaryOpRemU32 int32
)

// Binaryen UnaryOp constants. Set by module package at init.
var (
	UnaryOpEqzI32      int32
	UnaryOpEqzI64      int32
	UnaryOpClzI32      int32
	UnaryOpCtzI32      int32
	UnaryOpPopcntI32   int32
	UnaryOpExtend8I32  int32
	UnaryOpExtend8I64  int32
	UnaryOpExtend16I32 int32
	UnaryOpExtend16I64 int32
	UnaryOpExtend32I64 int32
)

// Binaryen TypeRef constants for getExpressionType. Set by module package at init.
var (
	TypeRefI32  int32
	TypeRefI64  int32
	TypeRefF32  int32
	TypeRefF64  int32
	TypeRefV128 int32
)

// BuiltinNames string constants. Set by builtins package at init.
var (
	BuiltinNameStringEq  string
	BuiltinNameStringNe  string
	BuiltinNameStringNot string
	BuiltinNameTostack   string
)

// --- Flow struct ---

// Flow is a concurrent code flow analyzer.
type Flow struct {
	// TargetFunction is the function this flow generates code into.
	TargetFunction FlowFunctionRef
	// InlineFunction is the function being inlined, if any.
	InlineFunction FlowFunctionRef
	// Parent is the parent flow.
	Parent *Flow
	// Outer flow. Only relevant for first-class functions.
	Outer *Flow
	// Flags indicates specific control flow conditions.
	Flags FlowFlags
	// ContinueLabel is the break target for continue statements.
	ContinueLabel string
	// BreakLabel is the break target for break statements.
	BreakLabel string
	// ScopedLocals are scoped local variables.
	ScopedLocals map[string]FlowLocalRef
	// ScopedTypeAlias are scoped type aliases.
	ScopedTypeAlias map[string]FlowTypeDefinitionRef
	// LocalFlags are per-local-index flags.
	LocalFlags []LocalFlags
	// ThisFieldFlags are per-field flags on `this` (constructors only).
	ThisFieldFlags map[FlowPropertyRef]FieldFlags
	// InlineReturnLabel is the break target for return when inlining.
	InlineReturnLabel string
	// TrueFlows are alternative flows if a compound expression is true-ish.
	TrueFlows map[ExpressionRef]*Flow
	// FalseFlows are alternative flows if a compound expression is false-ish.
	FalseFlows map[ExpressionRef]*Flow
}

// newFlow creates a new flow (private constructor).
func newFlow(targetFunction, inlineFunction FlowFunctionRef) *Flow {
	return &Flow{
		TargetFunction: targetFunction,
		InlineFunction: inlineFunction,
	}
}

// CreateDefault creates the default top-level flow of the specified function.
func CreateDefault(targetFunction FlowFunctionRef) *Flow {
	f := newFlow(targetFunction, nil)
	if targetFunction.Is(CommonFlagsConstructor) {
		f.InitThisFieldFlags()
	}
	if targetFunction.FlowProgram().UncheckedBehaviorAlways() {
		f.SetFlag(FlowFlagUncheckedContext)
	}
	return f
}

// CreateInline creates an inline flow, compiling inlineFunction into targetFunction.
func CreateInline(targetFunction, inlineFunction FlowFunctionRef) *Flow {
	f := newFlow(targetFunction, inlineFunction)
	inlineId := inlineFunction.AllocInlineId()
	f.InlineReturnLabel = fmt.Sprintf("%s|inlined.%d", inlineFunction.FlowInternalName(), inlineId)
	if inlineFunction.Is(CommonFlagsConstructor) {
		f.InitThisFieldFlags()
	}
	if targetFunction.FlowProgram().UncheckedBehaviorAlways() {
		f.SetFlag(FlowFlagUncheckedContext)
	}
	return f
}

// IsInline tests if this is an inline flow.
func (f *Flow) IsInline() bool {
	return f.InlineFunction != nil
}

// SourceFunction gets the source function being compiled.
func (f *Flow) SourceFunction() FlowFunctionRef {
	if f.InlineFunction != nil {
		return f.InlineFunction
	}
	return f.TargetFunction
}

// Program gets the program this flow belongs to.
func (f *Flow) Program() FlowProgramRef {
	return f.TargetFunction.FlowProgram()
}

// ReturnType gets the current return type.
func (f *Flow) ReturnType() *types.Type {
	return f.SourceFunction().FlowSignature().ReturnType
}

// ContextualTypeArguments gets the current contextual type arguments.
func (f *Flow) ContextualTypeArguments() map[string]*types.Type {
	return f.SourceFunction().FlowContextualTypeArguments()
}

// --- Flag operations ---

// Is tests if this flow has ALL of the specified flags.
func (f *Flow) Is(flag FlowFlags) bool {
	return f.Flags&flag == flag
}

// IsAny tests if this flow has ANY of the specified flags.
func (f *Flow) IsAny(flag FlowFlags) bool {
	return f.Flags&flag != 0
}

// SetFlag sets the specified flag or flags.
func (f *Flow) SetFlag(flag FlowFlags) {
	f.Flags |= flag
}

// UnsetFlag unsets the specified flag or flags.
func (f *Flow) UnsetFlag(flag FlowFlags) {
	f.Flags &^= flag
}

// DeriveConditionalFlags converts categorical flags to conditional counterparts.
func (f *Flow) DeriveConditionalFlags() FlowFlags {
	condiFlags := f.Flags & FlowFlagAnyConditional
	if f.Is(FlowFlagReturns) {
		condiFlags |= FlowFlagConditionallyReturns
	}
	if f.Is(FlowFlagThrows) {
		condiFlags |= FlowFlagConditionallyThrows
	}
	if f.Is(FlowFlagBreaks) {
		condiFlags |= FlowFlagConditionallyBreaks
	}
	if f.Is(FlowFlagContinues) {
		condiFlags |= FlowFlagConditionallyContinues
	}
	if f.Is(FlowFlagAccessesThis) {
		condiFlags |= FlowFlagConditionallyAccessesThis
	}
	return condiFlags
}

// --- Fork ---

// Fork forks this flow to a child flow.
func (f *Flow) Fork(newBreakContext, newContinueContext bool) *Flow {
	branch := newFlow(f.TargetFunction, f.InlineFunction)
	branch.Parent = f
	branch.Flags = f.Flags
	branch.Outer = f.Outer
	if newBreakContext {
		branch.Flags &^= FlowFlagBreaks | FlowFlagConditionallyBreaks
	} else {
		branch.BreakLabel = f.BreakLabel
	}
	if newContinueContext {
		branch.Flags &^= FlowFlagContinues | FlowFlagConditionallyContinues
	} else {
		branch.ContinueLabel = f.ContinueLabel
	}
	branch.LocalFlags = make([]LocalFlags, len(f.LocalFlags))
	copy(branch.LocalFlags, f.LocalFlags)
	if f.SourceFunction().Is(CommonFlagsConstructor) {
		branch.ThisFieldFlags = cloneFieldFlags(f.ThisFieldFlags)
	}
	branch.InlineReturnLabel = f.InlineReturnLabel
	return branch
}

// ForkThen forks this flow to a child flow where condExpr is true-ish.
func (f *Flow) ForkThen(condExpr ExpressionRef, newBreakContext, newContinueContext bool) *Flow {
	child := f.Fork(newBreakContext, newContinueContext)
	if f.TrueFlows != nil {
		if trueFlow, ok := f.TrueFlows[condExpr]; ok {
			child.Inherit(trueFlow)
		}
	}
	child.InheritNonnullIfTrue(condExpr, nil)
	return child
}

// NoteThen remembers the alternative flow if condExpr turns out true.
func (f *Flow) NoteThen(condExpr ExpressionRef, trueFlow *Flow) {
	if f.TrueFlows == nil {
		f.TrueFlows = make(map[ExpressionRef]*Flow)
	}
	f.TrueFlows[condExpr] = trueFlow
}

// ForkElse forks this flow to a child flow where condExpr is false-ish.
func (f *Flow) ForkElse(condExpr ExpressionRef) *Flow {
	child := f.Fork(false, false)
	if f.FalseFlows != nil {
		if falseFlow, ok := f.FalseFlows[condExpr]; ok {
			child.Inherit(falseFlow)
		}
	}
	child.InheritNonnullIfFalse(condExpr, nil)
	return child
}

// NoteElse remembers the alternative flow if condExpr turns out false.
func (f *Flow) NoteElse(condExpr ExpressionRef, falseFlow *Flow) {
	if f.FalseFlows == nil {
		f.FalseFlows = make(map[ExpressionRef]*Flow)
	}
	f.FalseFlows[condExpr] = falseFlow
}

// --- Scoped type aliases ---

// AddScopedTypeAlias adds a scoped type alias to this flow.
func (f *Flow) AddScopedTypeAlias(name string, definition FlowTypeDefinitionRef) {
	if f.ScopedTypeAlias == nil {
		f.ScopedTypeAlias = make(map[string]FlowTypeDefinitionRef)
	}
	f.ScopedTypeAlias[name] = definition
}

// LookupScopedTypeAlias walks the parent chain to find a scoped type alias.
func (f *Flow) LookupScopedTypeAlias(name string) FlowTypeDefinitionRef {
	current := f
	for current != nil {
		if current.ScopedTypeAlias != nil {
			if def, ok := current.ScopedTypeAlias[name]; ok {
				return def
			}
		}
		current = current.Parent
	}
	return nil
}

// LookupTypeAlias looks up a type alias in scope, then parent function scope.
func (f *Flow) LookupTypeAlias(name string) FlowTypeDefinitionRef {
	if def := f.LookupScopedTypeAlias(name); def != nil {
		return def
	}
	sourceParent := f.SourceFunction().FlowParent()
	if sourceParent.GetElementKind() == ElementKindFunction {
		parentFn := sourceParent.(FlowFunctionRef)
		return parentFn.GetFlow().LookupTypeAlias(name)
	}
	return nil
}

// --- Scoped locals ---

// GetTempLocal gets a free temporary local of the specified type.
func (f *Flow) GetTempLocal(typ *types.Type) FlowLocalRef {
	local := f.TargetFunction.FlowAddLocal(typ)
	f.UnsetLocalFlag(local.FlowIndex(), ^LocalFlags(0))
	return local
}

// GetScopedLocal gets the scoped local of the specified name.
func (f *Flow) GetScopedLocal(name string) FlowLocalRef {
	if f.ScopedLocals != nil {
		if local, ok := f.ScopedLocals[name]; ok {
			return local
		}
	}
	return nil
}

// AddScopedLocal adds a new scoped local of the specified name.
func (f *Flow) AddScopedLocal(name string, typ *types.Type) FlowLocalRef {
	scopedLocal := f.GetTempLocal(typ)
	scopedLocal.SetName(name)
	if MangleInternalNameFunc != nil {
		scopedLocal.SetInternalName(MangleInternalNameFunc(name, scopedLocal.FlowGetParent(), false))
	}
	if f.ScopedLocals == nil {
		f.ScopedLocals = make(map[string]FlowLocalRef)
	} else if _, exists := f.ScopedLocals[name]; exists {
		panic(fmt.Sprintf("flow: AddScopedLocal: scoped local '%s' already exists", name))
	}
	scopedLocal.Set(CommonFlagsScoped)
	f.ScopedLocals[name] = scopedLocal
	return scopedLocal
}

// AddScopedDummyLocal adds a new scoped dummy local with index -1.
func (f *Flow) AddScopedDummyLocal(name string, typ *types.Type, declarationNode interface{}) FlowLocalRef {
	if NewLocalFunc == nil {
		panic("flow: NewLocalFunc not set")
	}
	scopedDummy := NewLocalFunc(name, -1, typ, f.TargetFunction)
	if f.ScopedLocals == nil {
		f.ScopedLocals = make(map[string]FlowLocalRef)
	} else if _, exists := f.ScopedLocals[name]; exists {
		f.Program().Error(
			DiagnosticCodeCannotRedeclare,
			declarationNode, name,
		)
	}
	scopedDummy.Set(CommonFlagsScoped)
	f.ScopedLocals[name] = scopedDummy
	return scopedDummy
}

// AddScopedAlias adds a scoped alias for an existing local index.
func (f *Flow) AddScopedAlias(name string, typ *types.Type, index int32, reportNode interface{}) FlowLocalRef {
	if f.ScopedLocals == nil {
		f.ScopedLocals = make(map[string]FlowLocalRef)
	} else if existing, ok := f.ScopedLocals[name]; ok {
		if reportNode != nil {
			if !existing.DeclarationIsNative() {
				f.Program().ErrorRelated(
					DiagnosticCodeDuplicateIdentifier,
					reportNode,
					existing.DeclarationNameRange(),
					name,
				)
			} else {
				f.Program().Error(
					DiagnosticCodeDuplicateIdentifier,
					reportNode, name,
				)
			}
		}
		return existing
	}
	localsByIndex := f.TargetFunction.FlowLocalsByIndex()
	if int(index) >= len(localsByIndex) {
		panic(fmt.Sprintf("flow: AddScopedAlias: index %d >= localsByIndex length %d", index, len(localsByIndex)))
	}
	if NewLocalFunc == nil {
		panic("flow: NewLocalFunc not set")
	}
	scopedAlias := NewLocalFunc(name, index, typ, f.TargetFunction)
	scopedAlias.Set(CommonFlagsScoped)
	f.ScopedLocals[name] = scopedAlias
	return scopedAlias
}

// FreeScopedDummyLocal frees a single scoped local by its name.
func (f *Flow) FreeScopedDummyLocal(name string) {
	if f.ScopedLocals == nil {
		panic(fmt.Sprintf("flow: FreeScopedDummyLocal: scopedLocals is nil (name '%s')", name))
	}
	local, ok := f.ScopedLocals[name]
	if !ok {
		panic(fmt.Sprintf("flow: FreeScopedDummyLocal: scoped local '%s' not found", name))
	}
	if local.FlowIndex() != -1 {
		panic("flow: FreeScopedDummyLocal called on non-dummy")
	}
	delete(f.ScopedLocals, name)
}

// LookupLocal walks the parent chain to find a scoped local by name.
func (f *Flow) LookupLocal(name string) FlowLocalRef {
	current := f
	for current != nil {
		if current.ScopedLocals != nil {
			if local, ok := current.ScopedLocals[name]; ok {
				return local
			}
		}
		current = current.Parent
	}
	return nil
}

// Lookup looks up the element with the specified name relative to this flow's scope.
func (f *Flow) Lookup(name string) FlowElementRef {
	local := f.LookupLocal(name)
	if local != nil {
		return local
	}
	return f.SourceFunction().FlowLookup(name)
}

// --- Local flag operations ---

// IsLocalFlag tests if the local at the specified index has ALL of the specified flags.
func (f *Flow) IsLocalFlag(index int32, flag LocalFlags) bool {
	return f.isLocalFlagDefault(index, flag, true)
}

// IsLocalFlagDefault tests if the local at the specified index has ALL of the specified flags,
// with a configurable default for inlined flows (when index < 0).
func (f *Flow) IsLocalFlagDefault(index int32, flag LocalFlags, defaultIfInlined bool) bool {
	return f.isLocalFlagDefault(index, flag, defaultIfInlined)
}

func (f *Flow) isLocalFlagDefault(index int32, flag LocalFlags, defaultIfInlined bool) bool {
	if index < 0 {
		return defaultIfInlined
	}
	localFlags := f.LocalFlags
	return int(index) < len(localFlags) && (localFlags[index]&flag) == flag
}

// IsAnyLocalFlag tests if the local at the specified index has ANY of the specified flags.
func (f *Flow) IsAnyLocalFlag(index int32, flag LocalFlags) bool {
	return f.isAnyLocalFlagDefault(index, flag, true)
}

func (f *Flow) isAnyLocalFlagDefault(index int32, flag LocalFlags, defaultIfInlined bool) bool {
	if index < 0 {
		return defaultIfInlined
	}
	localFlags := f.LocalFlags
	return int(index) < len(localFlags) && (localFlags[index]&flag) != 0
}

// SetLocalFlag sets flag(s) on a local by index.
func (f *Flow) SetLocalFlag(index int32, flag LocalFlags) {
	if index < 0 {
		return
	}
	for int(index) >= len(f.LocalFlags) {
		f.LocalFlags = append(f.LocalFlags, LocalFlagNone)
	}
	f.LocalFlags[index] |= flag
}

// UnsetLocalFlag unsets flag(s) on a local by index.
func (f *Flow) UnsetLocalFlag(index int32, flag LocalFlags) {
	if index < 0 {
		return
	}
	for int(index) >= len(f.LocalFlags) {
		f.LocalFlags = append(f.LocalFlags, LocalFlagNone)
	}
	f.LocalFlags[index] &^= flag
}

// --- Field flag operations ---

// InitThisFieldFlags initializes field flags for constructor flows.
func (f *Flow) InitThisFieldFlags() {
	f.ThisFieldFlags = make(map[FlowPropertyRef]FieldFlags)
	sourceFunction := f.SourceFunction()
	parent := sourceFunction.FlowParent()
	if parent.GetElementKind() != ElementKindClass {
		return
	}
	classRef, ok := parent.(FlowClassRef)
	if !ok {
		return
	}
	members := classRef.FlowMembers()
	if members == nil {
		return
	}
	for _, member := range members {
		if member.GetElementKind() != ElementKindPropertyPrototype {
			continue
		}
		propProto, ok := member.(FlowPropertyPrototypeRef)
		if !ok {
			continue
		}
		property := propProto.Instance()
		if property == nil || !property.IsField() {
			continue
		}
		if propProto.FlowGetParent() != parent.(FlowElementRef) ||
			property.FlowInitializerNode() != nil ||
			propProto.ParameterIndex() != -1 ||
			property.GetType().IsAny(types.TypeFlagValue|types.TypeFlagNullable) {
			f.SetThisFieldFlag(property, FieldFlagInitialized)
		}
	}
}

// IsThisFieldFlag tests if the specified this field has the specified flag(s).
func (f *Flow) IsThisFieldFlag(field FlowPropertyRef, flag FieldFlags) bool {
	if f.ThisFieldFlags != nil {
		if flags, ok := f.ThisFieldFlags[field]; ok {
			return (flags & flag) == flag
		}
	}
	return false
}

// SetThisFieldFlag sets flag(s) on the given this field.
func (f *Flow) SetThisFieldFlag(field FlowPropertyRef, flag FieldFlags) {
	if f.ThisFieldFlags != nil {
		if !f.SourceFunction().Is(CommonFlagsConstructor) {
			panic("flow: SetThisFieldFlag: sourceFunction is not a constructor but fieldFlags is set")
		}
		if flags, ok := f.ThisFieldFlags[field]; ok {
			f.ThisFieldFlags[field] = flags | flag
		} else {
			f.ThisFieldFlags[field] = flag
		}
	} else {
		if f.SourceFunction().Is(CommonFlagsConstructor) {
			panic("flow: SetThisFieldFlag: sourceFunction is a constructor but fieldFlags is nil")
		}
	}
}

// --- Control flow labels ---

// PushControlFlowLabel pushes a new break label. Returns the label ID.
func (f *Flow) PushControlFlowLabel() int32 {
	id := f.TargetFunction.AllocBreakId()
	f.TargetFunction.PushBreakStack(id)
	return id
}

// PopControlFlowLabel pops the most recent break label.
func (f *Flow) PopControlFlowLabel(expectedLabel int32) {
	popped := f.TargetFunction.PopBreakStack()
	if popped != expectedLabel {
		panic("flow: mismatched control flow label")
	}
}

// --- Inherit / merge ---

// Inherit inherits flags of another flow into this one (finished inner block).
func (f *Flow) Inherit(other *Flow) {
	if other.TargetFunction != f.TargetFunction {
		panic("flow: Inherit: targetFunction mismatch")
	}
	otherFlags := other.Flags

	if f.BreakLabel != other.BreakLabel {
		if otherFlags&(FlowFlagBreaks|FlowFlagConditionallyBreaks) != 0 {
			otherFlags &^= FlowFlagTerminates
		}
		otherFlags &^= FlowFlagBreaks | FlowFlagConditionallyBreaks
	}
	if f.ContinueLabel != other.ContinueLabel {
		otherFlags &^= FlowFlagContinues | FlowFlagConditionallyContinues
	}

	f.Flags = f.Flags | otherFlags
	f.LocalFlags = other.LocalFlags
	f.ThisFieldFlags = other.ThisFieldFlags
}

// MergeSideEffects merges only the side effects of a branch (not taken path).
func (f *Flow) MergeSideEffects(other *Flow) {
	if other.TargetFunction != f.TargetFunction {
		panic("flow: MergeSideEffects: targetFunction mismatch")
	}
	thisFlags := f.Flags
	otherFlags := other.Flags
	newFlags := FlowFlagNone

	if thisFlags&FlowFlagReturns != 0 {
		newFlags |= FlowFlagReturns
	} else if otherFlags&FlowFlagReturns != 0 {
		newFlags |= FlowFlagConditionallyReturns
	} else {
		newFlags |= (thisFlags | otherFlags) & FlowFlagConditionallyReturns
	}

	newFlags |= thisFlags & otherFlags & FlowFlagReturnsWrapped
	newFlags |= thisFlags & otherFlags & FlowFlagReturnsNonNull

	if thisFlags&FlowFlagThrows != 0 {
		newFlags |= FlowFlagThrows
	} else if otherFlags&FlowFlagThrows != 0 {
		newFlags |= FlowFlagConditionallyThrows
	} else {
		newFlags |= (thisFlags | otherFlags) & FlowFlagConditionallyThrows
	}

	if thisFlags&FlowFlagBreaks != 0 {
		newFlags |= FlowFlagBreaks
	} else if other.BreakLabel == f.BreakLabel {
		if otherFlags&FlowFlagBreaks != 0 {
			newFlags |= FlowFlagConditionallyBreaks
		} else {
			newFlags |= (thisFlags | otherFlags) & FlowFlagConditionallyBreaks
		}
	} else {
		newFlags |= thisFlags & FlowFlagConditionallyBreaks
	}

	if thisFlags&FlowFlagContinues != 0 {
		newFlags |= FlowFlagContinues
	} else if other.ContinueLabel == f.ContinueLabel {
		if otherFlags&FlowFlagContinues != 0 {
			newFlags |= FlowFlagConditionallyContinues
		} else {
			newFlags |= (thisFlags | otherFlags) & FlowFlagConditionallyContinues
		}
	} else {
		newFlags |= thisFlags & FlowFlagConditionallyContinues
	}

	if thisFlags&FlowFlagAccessesThis != 0 {
		if otherFlags&FlowFlagAccessesThis != 0 {
			newFlags |= FlowFlagAccessesThis
		} else {
			newFlags |= FlowFlagConditionallyAccessesThis
		}
	} else if otherFlags&FlowFlagAccessesThis != 0 {
		newFlags |= FlowFlagConditionallyAccessesThis
	}

	newFlags |= (thisFlags | otherFlags) & FlowFlagMayReturnNonThis
	newFlags |= thisFlags & otherFlags & FlowFlagCallsSuper

	if thisFlags&FlowFlagTerminates != 0 {
		newFlags |= FlowFlagTerminates
	}

	f.Flags = newFlags | (thisFlags & (FlowFlagUncheckedContext | FlowFlagCtorParamContext))
}

// MergeBranch merges a branch joining again with this flow (if without else).
func (f *Flow) MergeBranch(other *Flow) {
	f.MergeSideEffects(other)

	thisLocalFlags := f.LocalFlags
	numThisFlags := len(thisLocalFlags)
	otherLocalFlags := other.LocalFlags
	numOtherFlags := len(otherLocalFlags)
	maxFlags := numThisFlags
	if numOtherFlags > maxFlags {
		maxFlags = numOtherFlags
	}
	for int(maxFlags) > len(thisLocalFlags) {
		thisLocalFlags = append(thisLocalFlags, LocalFlagNone)
	}
	for i := 0; i < maxFlags; i++ {
		var thisF, otherF LocalFlags
		if i < numThisFlags {
			thisF = f.LocalFlags[i]
		}
		if i < numOtherFlags {
			otherF = otherLocalFlags[i]
		}
		thisLocalFlags[i] = thisF & otherF & AllLocalFlags
	}
	f.LocalFlags = thisLocalFlags
}

// InheritAlternatives inherits two alternate branches (if/else).
func (f *Flow) InheritAlternatives(left, right *Flow) {
	if left.TargetFunction != right.TargetFunction {
		panic("flow: InheritAlternatives: left and right targetFunction mismatch")
	}
	if left.TargetFunction != f.TargetFunction {
		panic("flow: InheritAlternatives: targetFunction mismatch with this flow")
	}
	leftFlags := left.Flags
	rightFlags := right.Flags
	newFlags := FlowFlagNone

	if leftFlags&FlowFlagReturns != 0 {
		if rightFlags&FlowFlagReturns != 0 {
			newFlags |= FlowFlagReturns
		} else {
			newFlags |= FlowFlagConditionallyReturns
		}
	} else if rightFlags&FlowFlagReturns != 0 {
		newFlags |= FlowFlagConditionallyReturns
	} else {
		newFlags |= (leftFlags | rightFlags) & FlowFlagConditionallyReturns
	}

	if leftFlags&FlowFlagReturnsWrapped != 0 && rightFlags&FlowFlagReturnsWrapped != 0 {
		newFlags |= FlowFlagReturnsWrapped
	}
	if leftFlags&FlowFlagReturnsNonNull != 0 && rightFlags&FlowFlagReturnsNonNull != 0 {
		newFlags |= FlowFlagReturnsNonNull
	}

	if leftFlags&FlowFlagThrows != 0 {
		if rightFlags&FlowFlagThrows != 0 {
			newFlags |= FlowFlagThrows
		} else {
			newFlags |= FlowFlagConditionallyThrows
		}
	} else if rightFlags&FlowFlagThrows != 0 {
		newFlags |= FlowFlagConditionallyThrows
	} else {
		newFlags |= (leftFlags | rightFlags) & FlowFlagConditionallyThrows
	}

	if leftFlags&FlowFlagBreaks != 0 {
		if rightFlags&FlowFlagBreaks != 0 {
			newFlags |= FlowFlagBreaks
		} else {
			newFlags |= FlowFlagConditionallyBreaks
		}
	} else if rightFlags&FlowFlagBreaks != 0 {
		newFlags |= FlowFlagConditionallyBreaks
	} else {
		newFlags |= (leftFlags | rightFlags) & FlowFlagConditionallyBreaks
	}

	if leftFlags&FlowFlagContinues != 0 {
		if rightFlags&FlowFlagContinues != 0 {
			newFlags |= FlowFlagContinues
		} else {
			newFlags |= FlowFlagConditionallyContinues
		}
	} else if rightFlags&FlowFlagContinues != 0 {
		newFlags |= FlowFlagConditionallyContinues
	} else {
		newFlags |= (leftFlags | rightFlags) & FlowFlagConditionallyContinues
	}

	if leftFlags&FlowFlagAccessesThis != 0 {
		if rightFlags&FlowFlagAccessesThis != 0 {
			newFlags |= FlowFlagAccessesThis
		} else {
			newFlags |= FlowFlagConditionallyAccessesThis
		}
	} else if rightFlags&FlowFlagAccessesThis != 0 {
		newFlags |= FlowFlagConditionallyAccessesThis
	} else {
		newFlags |= (leftFlags | rightFlags) & FlowFlagConditionallyAccessesThis
	}

	newFlags |= (leftFlags | rightFlags) & FlowFlagMayReturnNonThis

	if leftFlags&FlowFlagCallsSuper != 0 && rightFlags&FlowFlagCallsSuper != 0 {
		newFlags |= FlowFlagCallsSuper
	}
	if leftFlags&FlowFlagTerminates != 0 && rightFlags&FlowFlagTerminates != 0 {
		newFlags |= FlowFlagTerminates
	}

	f.Flags = newFlags | (f.Flags & (FlowFlagUncheckedContext | FlowFlagCtorParamContext))

	// local flags
	thisLocalFlags := f.LocalFlags
	if leftFlags&FlowFlagTerminates != 0 {
		if rightFlags&FlowFlagTerminates == 0 {
			rightLocalFlags := right.LocalFlags
			for i := 0; i < len(rightLocalFlags); i++ {
				for int(i) >= len(thisLocalFlags) {
					thisLocalFlags = append(thisLocalFlags, LocalFlagNone)
				}
				thisLocalFlags[i] = rightLocalFlags[i]
			}
		}
	} else if rightFlags&FlowFlagTerminates != 0 {
		leftLocalFlags := left.LocalFlags
		for i := 0; i < len(leftLocalFlags); i++ {
			for int(i) >= len(thisLocalFlags) {
				thisLocalFlags = append(thisLocalFlags, LocalFlagNone)
			}
			thisLocalFlags[i] = leftLocalFlags[i]
		}
	} else {
		leftLocalFlags := left.LocalFlags
		numLeftFlags := len(leftLocalFlags)
		rightLocalFlags := right.LocalFlags
		numRightFlags := len(rightLocalFlags)
		maxFlags := numLeftFlags
		if numRightFlags > maxFlags {
			maxFlags = numRightFlags
		}
		for int(maxFlags) > len(thisLocalFlags) {
			thisLocalFlags = append(thisLocalFlags, LocalFlagNone)
		}
		for i := 0; i < maxFlags; i++ {
			var lf, rf LocalFlags
			if i < numLeftFlags {
				lf = leftLocalFlags[i]
			}
			if i < numRightFlags {
				rf = rightLocalFlags[i]
			}
			thisLocalFlags[i] = lf & rf & AllLocalFlags
		}
	}
	f.LocalFlags = thisLocalFlags

	// field flags
	leftFieldFlags := left.ThisFieldFlags
	if leftFieldFlags != nil {
		newFieldFlags := make(map[FlowPropertyRef]FieldFlags)
		rightFieldFlags := right.ThisFieldFlags
		for key, lf := range leftFieldFlags {
			if lf&FieldFlagInitialized != 0 {
				if rf, ok := rightFieldFlags[key]; ok && rf&FieldFlagInitialized != 0 {
					newFieldFlags[key] = FieldFlagInitialized
				}
			}
		}
		f.ThisFieldFlags = newFieldFlags
	}
}

// ResetIfNeedsRecompile tests if loop recompilation is needed and resets if so.
func (f *Flow) ResetIfNeedsRecompile(other *Flow, numLocalsBefore int) bool {
	if f.TargetFunction != other.TargetFunction {
		panic("flow: ResetIfNeedsRecompile: targetFunction mismatch")
	}
	numThisFlags := len(f.LocalFlags)
	numOtherFlags := len(other.LocalFlags)
	targetFunction := f.TargetFunction
	localsByIndex := targetFunction.FlowLocalsByIndex()
	// TS: assert(localsByIndex == other.targetFunction.localsByIndex)
	// Redundant given targetFunction equality, but kept for faithfulness
	needsRecompile := false
	minFlags := numThisFlags
	if numOtherFlags < minFlags {
		minFlags = numOtherFlags
	}
	for i := 0; i < minFlags; i++ {
		local := localsByIndex[i]
		typ := local.GetType()
		if typ.IsShortIntegerValue() {
			if f.IsLocalFlag(int32(i), LocalFlagWrapped) && !other.IsLocalFlag(int32(i), LocalFlagWrapped) {
				f.UnsetLocalFlag(int32(i), LocalFlagWrapped)
				needsRecompile = true
			}
		}
		if typ.IsNullableReference() {
			if f.IsLocalFlag(int32(i), LocalFlagNonNull) && !other.IsLocalFlag(int32(i), LocalFlagNonNull) {
				f.UnsetLocalFlag(int32(i), LocalFlagNonNull)
				needsRecompile = true
			}
		}
	}
	if needsRecompile {
		if len(localsByIndex) < numLocalsBefore {
			panic(fmt.Sprintf("flow: ResetIfNeedsRecompile: localsByIndex length %d < numLocalsBefore %d", len(localsByIndex), numLocalsBefore))
		}
		targetFunction.FlowTruncateLocals(numLocalsBefore)
		if len(f.LocalFlags) > numLocalsBefore {
			f.LocalFlags = f.LocalFlags[:numLocalsBefore]
		}
	}
	return needsRecompile
}

// --- Binaryen IR inspection methods ---
// These methods inspect Binaryen IR using accessor functions from the module package.
// They use safe defaults when the module package hasn't been initialized.

// IsNonnull checks if an expression is known to be non-null.
func (f *Flow) IsNonnull(expr ExpressionRef, typ *types.Type) bool {
	if !typ.IsNullableReference() {
		return true
	}
	if GetExpressionId == nil {
		return false
	}
	exprId := GetExpressionId(expr)
	switch {
	case exprId == ExpressionIdLocalSet:
		if !IsLocalTee(expr) {
			break
		}
		local := f.TargetFunction.FlowLocalsByIndex()[GetLocalSetIndex(expr)]
		return !local.GetType().IsNullableReference() || f.isLocalFlagDefault(local.FlowIndex(), LocalFlagNonNull, false)
	case exprId == ExpressionIdLocalGet:
		local := f.TargetFunction.FlowLocalsByIndex()[GetLocalGetIndex(expr)]
		return !local.GetType().IsNullableReference() || f.isLocalFlagDefault(local.FlowIndex(), LocalFlagNonNull, false)
	}
	return false
}

// InheritNonnullIfTrue updates local states to reflect that this branch is taken when expr is true-ish.
func (f *Flow) InheritNonnullIfTrue(expr ExpressionRef, iff *Flow) {
	if GetExpressionId == nil {
		return
	}
	exprId := GetExpressionId(expr)
	switch {
	case exprId == ExpressionIdLocalSet:
		if !IsLocalTee(expr) {
			return
		}
		local := f.TargetFunction.FlowLocalsByIndex()[GetLocalSetIndex(expr)]
		if iff == nil || iff.IsLocalFlag(local.FlowIndex(), LocalFlagNonNull) {
			f.SetLocalFlag(local.FlowIndex(), LocalFlagNonNull)
		}
		f.InheritNonnullIfTrue(GetLocalSetValue(expr), iff)
	case exprId == ExpressionIdLocalGet:
		local := f.TargetFunction.FlowLocalsByIndex()[GetLocalGetIndex(expr)]
		if iff == nil || iff.IsLocalFlag(local.FlowIndex(), LocalFlagNonNull) {
			f.SetLocalFlag(local.FlowIndex(), LocalFlagNonNull)
		}
	case exprId == ExpressionIdIf:
		ifFalse := GetIfFalse(expr)
		if ifFalse != 0 && IsConstZero(ifFalse) {
			f.InheritNonnullIfTrue(GetIfCondition(expr), iff)
			f.InheritNonnullIfTrue(GetIfTrue(expr), iff)
		}
	case exprId == ExpressionIdUnary:
		op := GetUnaryOp(expr)
		if op == UnaryOpEqzI32 || op == UnaryOpEqzI64 {
			f.InheritNonnullIfFalse(GetUnaryValue(expr), iff)
		}
	case exprId == ExpressionIdBinary:
		op := GetBinaryOp(expr)
		left := GetBinaryLeft(expr)
		right := GetBinaryRight(expr)
		if op == BinaryOpEqI32 || op == BinaryOpEqI64 {
			if IsConstNonZero(left) {
				f.InheritNonnullIfTrue(right, iff)
			} else if IsConstNonZero(right) {
				f.InheritNonnullIfTrue(left, iff)
			}
		} else if op == BinaryOpNeI32 || op == BinaryOpNeI64 {
			if IsConstZero(left) {
				f.InheritNonnullIfTrue(right, iff)
			} else if IsConstZero(right) {
				f.InheritNonnullIfTrue(left, iff)
			}
		}
	case exprId == ExpressionIdCall:
		name := GetCallTarget(expr)
		if name == BuiltinNameStringEq {
			left := GetCallOperandAt(expr, 0)
			right := GetCallOperandAt(expr, 1)
			if IsConstNonZero(left) {
				f.InheritNonnullIfTrue(right, iff)
			} else if IsConstNonZero(right) {
				f.InheritNonnullIfTrue(left, iff)
			}
		} else if name == BuiltinNameStringNe {
			left := GetCallOperandAt(expr, 0)
			right := GetCallOperandAt(expr, 1)
			if IsConstZero(left) {
				f.InheritNonnullIfTrue(right, iff)
			} else if IsConstZero(right) {
				f.InheritNonnullIfTrue(left, iff)
			}
		} else if name == BuiltinNameStringNot {
			f.InheritNonnullIfFalse(GetCallOperandAt(expr, 0), iff)
		} else if name == BuiltinNameTostack {
			f.InheritNonnullIfTrue(GetCallOperandAt(expr, 0), iff)
		}
	}
}

// InheritNonnullIfFalse updates local states to reflect that this branch is taken when expr is false-ish.
func (f *Flow) InheritNonnullIfFalse(expr ExpressionRef, iff *Flow) {
	if GetExpressionId == nil {
		return
	}
	exprId := GetExpressionId(expr)
	switch {
	case exprId == ExpressionIdUnary:
		op := GetUnaryOp(expr)
		if op == UnaryOpEqzI32 || op == UnaryOpEqzI64 {
			f.InheritNonnullIfTrue(GetUnaryValue(expr), iff)
		}
	case exprId == ExpressionIdIf:
		ifTrue := GetIfTrue(expr)
		ifFalse := GetIfFalse(expr)
		if ifFalse != 0 && IsConstNonZero(ifTrue) {
			f.InheritNonnullIfFalse(GetIfCondition(expr), iff)
			f.InheritNonnullIfFalse(ifFalse, iff)
		}
	case exprId == ExpressionIdBinary:
		op := GetBinaryOp(expr)
		left := GetBinaryLeft(expr)
		right := GetBinaryRight(expr)
		if op == BinaryOpEqI32 || op == BinaryOpEqI64 {
			if IsConstZero(left) {
				f.InheritNonnullIfTrue(right, iff)
			} else if IsConstZero(right) {
				f.InheritNonnullIfTrue(left, iff)
			}
		} else if op == BinaryOpNeI32 || op == BinaryOpNeI64 {
			if IsConstNonZero(left) {
				f.InheritNonnullIfTrue(right, iff)
			} else if IsConstNonZero(right) {
				f.InheritNonnullIfTrue(left, iff)
			}
		}
	case exprId == ExpressionIdCall:
		name := GetCallTarget(expr)
		if name == BuiltinNameStringEq {
			left := GetCallOperandAt(expr, 0)
			right := GetCallOperandAt(expr, 1)
			if IsConstZero(left) {
				f.InheritNonnullIfTrue(right, iff)
			} else if IsConstZero(right) {
				f.InheritNonnullIfTrue(left, iff)
			}
		} else if name == BuiltinNameStringNe {
			left := GetCallOperandAt(expr, 0)
			right := GetCallOperandAt(expr, 1)
			if IsConstNonZero(left) {
				f.InheritNonnullIfTrue(right, iff)
			} else if IsConstNonZero(right) {
				f.InheritNonnullIfTrue(left, iff)
			}
		} else if name == BuiltinNameStringNot {
			f.InheritNonnullIfTrue(GetCallOperandAt(expr, 0), iff)
		} else if name == BuiltinNameTostack {
			f.InheritNonnullIfFalse(GetCallOperandAt(expr, 0), iff)
		}
	}
}

// CanOverflow tests if an expression can possibly overflow for small integer types.
// Inspects Binaryen IR to determine if wrapping is needed.
// Returns true (conservative) when module accessors are not available.
func (f *Flow) CanOverflow(expr ExpressionRef, typ *types.Type) bool {
	// types other than i8, u8, i16, u16 and bool do not overflow
	if !typ.IsShortIntegerValue() {
		return false
	}
	if GetExpressionId == nil {
		return true // conservative default
	}

	var operand ExpressionRef
	exprId := GetExpressionId(expr)

	switch {
	// overflows if the local isn't wrapped or the conversion does
	case exprId == ExpressionIdLocalGet:
		local := f.TargetFunction.FlowLocalsByIndex()[GetLocalGetIndex(expr)]
		return !f.IsLocalFlag(local.FlowIndex(), LocalFlagWrapped) ||
			CanConversionOverflow(local.GetType(), typ)

	// overflows if the value does (local.tee)
	case exprId == ExpressionIdLocalSet:
		return f.CanOverflow(GetLocalSetValue(expr), typ)

	// overflows if the conversion does (globals are wrapped on set)
	case exprId == ExpressionIdGlobalGet:
		name := GetGlobalGetName(expr)
		elemsByName := f.Program().ElementsByName()
		global, ok := elemsByName[name]
		if !ok {
			return true
		}
		typedElem, ok := global.(FlowTypedElementRef)
		if !ok {
			return true
		}
		return CanConversionOverflow(typedElem.GetType(), typ)

	case exprId == ExpressionIdBinary:
		op := GetBinaryOp(expr)

		// comparisons do not overflow (result is 0 or 1)
		if op == BinaryOpEqI32 || op == BinaryOpEqI64 ||
			op == BinaryOpEqF32 || op == BinaryOpEqF64 ||
			op == BinaryOpNeI32 || op == BinaryOpNeI64 ||
			op == BinaryOpNeF32 || op == BinaryOpNeF64 ||
			op == BinaryOpLtI32 || op == BinaryOpLtU32 ||
			op == BinaryOpLtI64 || op == BinaryOpLtU64 ||
			op == BinaryOpLtF32 || op == BinaryOpLtF64 ||
			op == BinaryOpLeI32 || op == BinaryOpLeU32 ||
			op == BinaryOpLeI64 || op == BinaryOpLeU64 ||
			op == BinaryOpLeF32 || op == BinaryOpLeF64 ||
			op == BinaryOpGtI32 || op == BinaryOpGtU32 ||
			op == BinaryOpGtI64 || op == BinaryOpGtU64 ||
			op == BinaryOpGtF32 || op == BinaryOpGtF64 ||
			op == BinaryOpGeI32 || op == BinaryOpGeU32 ||
			op == BinaryOpGeI64 || op == BinaryOpGeU64 ||
			op == BinaryOpGeF32 || op == BinaryOpGeF64 {
			return false
		}

		// result won't overflow if one side is 0 or if one side is 1 and the other wrapped
		if op == BinaryOpMulI32 {
			operand = GetBinaryLeft(expr)
			if GetExpressionId(operand) == ExpressionIdConst &&
				(GetConstValueI32(operand) == 0 ||
					(GetConstValueI32(operand) == 1 &&
						!f.CanOverflow(GetBinaryRight(expr), typ))) {
				return false
			}
			operand = GetBinaryRight(expr)
			if GetExpressionId(operand) == ExpressionIdConst &&
				(GetConstValueI32(operand) == 0 ||
					(GetConstValueI32(operand) == 1 &&
						!f.CanOverflow(GetBinaryLeft(expr), typ))) {
				return false
			}
			return true
		}

		// result won't overflow if one side is a constant less than this type's mask
		// or one side is wrapped
		if op == BinaryOpAndI32 {
			mask := typ.ComputeSmallIntegerMask(types.TypeI32)
			operand = GetBinaryLeft(expr)
			leftOk := (GetExpressionId(operand) == ExpressionIdConst &&
				GetConstValueI32(operand) <= mask) ||
				!f.CanOverflow(operand, typ)
			if leftOk {
				return false
			}
			operand = GetBinaryRight(expr)
			rightOk := (GetExpressionId(operand) == ExpressionIdConst &&
				GetConstValueI32(operand) <= mask) ||
				!f.CanOverflow(operand, typ)
			if rightOk {
				return false
			}
			return true
		}

		// overflows if the shift doesn't clear potential garbage bits
		if op == BinaryOpShlI32 {
			shift := 32 - typ.Size
			operand = GetBinaryRight(expr)
			return GetExpressionId(operand) != ExpressionIdConst ||
				GetConstValueI32(operand) < shift
		}

		// overflows if the value does and the shift doesn't clear potential garbage bits
		if op == BinaryOpShrI32 {
			shift := 32 - typ.Size
			operand = GetBinaryRight(expr)
			return f.CanOverflow(GetBinaryLeft(expr), typ) &&
				(GetExpressionId(operand) != ExpressionIdConst ||
					GetConstValueI32(operand) < shift)
		}

		// overflows if the shift does not clear potential garbage bits.
		// if an unsigned value is wrapped, it can't overflow.
		if op == BinaryOpShrU32 {
			shift := 32 - typ.Size
			if typ.IsSignedIntegerValue() {
				operand = GetBinaryRight(expr)
				return !(GetExpressionId(operand) == ExpressionIdConst &&
					GetConstValueI32(operand) > shift) // must clear MSB
			}
			if !f.CanOverflow(GetBinaryLeft(expr), typ) {
				return false
			}
			operand = GetBinaryRight(expr)
			return !(GetExpressionId(operand) == ExpressionIdConst &&
				GetConstValueI32(operand) >= shift) // can leave MSB
		}

		// overflows if any side does
		if op == BinaryOpDivU32 || op == BinaryOpRemI32 || op == BinaryOpRemU32 {
			return f.CanOverflow(GetBinaryLeft(expr), typ) ||
				f.CanOverflow(GetBinaryRight(expr), typ)
		}

	case exprId == ExpressionIdUnary:
		op := GetUnaryOp(expr)

		// comparisons do not overflow (result is 0 or 1)
		if op == UnaryOpEqzI32 || op == UnaryOpEqzI64 {
			return false
		}

		// overflow if the maximum result (32) cannot be represented in the target type
		if op == UnaryOpClzI32 || op == UnaryOpCtzI32 || op == UnaryOpPopcntI32 {
			return typ.Size < 7
		}

		// sign extensions overflow if result can have high garbage bits in the target type
		if op == UnaryOpExtend8I32 {
			if typ.IsUnsignedIntegerValue() {
				return typ.Size < 32
			}
			return typ.Size < 8
		}
		if op == UnaryOpExtend8I64 {
			if typ.IsUnsignedIntegerValue() {
				return typ.Size < 64
			}
			return typ.Size < 8
		}
		if op == UnaryOpExtend16I32 {
			if typ.IsUnsignedIntegerValue() {
				return typ.Size < 32
			}
			return typ.Size < 16
		}
		if op == UnaryOpExtend16I64 {
			if typ.IsUnsignedIntegerValue() {
				return typ.Size < 64
			}
			return typ.Size < 16
		}
		if op == UnaryOpExtend32I64 {
			if typ.IsUnsignedIntegerValue() {
				return typ.Size < 64
			}
			return typ.Size < 32
		}

	// overflows if the value cannot be represented in the target type
	case exprId == ExpressionIdConst:
		var value int32
		exprType := GetExpressionType(expr)
		switch {
		case exprType == TypeRefI32:
			value = GetConstValueI32(expr)
		case exprType == TypeRefI64:
			value = GetConstValueI64Low(expr) // discards upper bits
		case exprType == TypeRefF32:
			value = int32(GetConstValueF32(expr))
		case exprType == TypeRefF64:
			value = int32(GetConstValueF64(expr))
		case exprType == TypeRefV128:
			return false
		default:
			panic("flow: unexpected expression type in canOverflow/Const")
		}
		switch typ.Kind {
		case types.TypeKindBool:
			return (value & ^int32(1)) != 0
		case types.TypeKindI8:
			return value < -128 || value > 127
		case types.TypeKindI16:
			return value < -32768 || value > 32767
		case types.TypeKindU8:
			return value < 0 || value > 255
		case types.TypeKindU16:
			return value < 0 || value > 65535
		}

	// overflows if the conversion does
	case exprId == ExpressionIdLoad:
		var fromType *types.Type
		signed := IsLoadSigned(expr)
		switch GetLoadBytes(expr) {
		case 1:
			if signed {
				fromType = types.TypeI8
			} else {
				fromType = types.TypeU8
			}
		case 2:
			if signed {
				fromType = types.TypeI16
			} else {
				fromType = types.TypeU16
			}
		default:
			if signed {
				fromType = types.TypeI32
			} else {
				fromType = types.TypeU32
			}
		}
		return CanConversionOverflow(fromType, typ)

	// overflows if the result does (last expression of unnamed block)
	case exprId == ExpressionIdBlock:
		name := GetBlockName(expr)
		if name == "" {
			size := GetBlockChildCount(expr)
			if size == 0 {
				return true
			}
			last := GetBlockChildAt(expr, size-1)
			return f.CanOverflow(last, typ)
		}

	// overflows if either side does
	case exprId == ExpressionIdIf:
		return f.CanOverflow(GetIfTrue(expr), typ) ||
			f.CanOverflow(GetIfFalse(expr), typ)

	// overflows if either side does
	case exprId == ExpressionIdSelect:
		return f.CanOverflow(GetSelectThen(expr), typ) ||
			f.CanOverflow(GetSelectElse(expr), typ)

	// overflows if the call does not return a wrapped value or the conversion does
	case exprId == ExpressionIdCall:
		instancesByName := f.Program().InstancesByName()
		instanceName := GetCallTarget(expr)
		if instance, ok := instancesByName[instanceName]; ok {
			funcRef, ok := instance.(FlowFunctionRef)
			if !ok {
				return true
			}
			returnType := funcRef.FlowSignature().ReturnType
			funcFlow := funcRef.GetFlow()
			return !funcFlow.Is(FlowFlagReturnsWrapped) ||
				CanConversionOverflow(returnType, typ)
		}
		return false // assume no overflow for builtins

	// doesn't technically overflow
	case exprId == ExpressionIdUnreachable:
		return false
	}

	return true
}

// --- Utility ---

// CanConversionOverflow tests if a conversion from one type to another can overflow.
func CanConversionOverflow(fromType, toType *types.Type) bool {
	return toType.IsShortIntegerValue() && (
		!fromType.IsIntegerValue() ||
		fromType.Size > toType.Size ||
		fromType.IsSignedIntegerValue() != toType.IsSignedIntegerValue())
}

// String returns a debug string representation of this flow.
func (f *Flow) String() string {
	levels := 0
	parent := f.Parent
	for parent != nil {
		parent = parent.Parent
		levels++
	}
	var sb strings.Builder
	if f.Is(FlowFlagReturns) {
		sb.WriteString("RETURNS ")
	}
	if f.Is(FlowFlagReturnsWrapped) {
		sb.WriteString("RETURNS_WRAPPED ")
	}
	if f.Is(FlowFlagReturnsNonNull) {
		sb.WriteString("RETURNS_NONNULL ")
	}
	if f.Is(FlowFlagThrows) {
		sb.WriteString("THROWS ")
	}
	if f.Is(FlowFlagBreaks) {
		sb.WriteString("BREAKS ")
	}
	if f.Is(FlowFlagContinues) {
		sb.WriteString("CONTINUES ")
	}
	if f.Is(FlowFlagAccessesThis) {
		sb.WriteString("ACCESSES_THIS ")
	}
	if f.Is(FlowFlagCallsSuper) {
		sb.WriteString("CALLS_SUPER ")
	}
	if f.Is(FlowFlagTerminates) {
		sb.WriteString("TERMINATES ")
	}
	if f.Is(FlowFlagConditionallyReturns) {
		sb.WriteString("CONDITIONALLY_RETURNS ")
	}
	if f.Is(FlowFlagConditionallyThrows) {
		sb.WriteString("CONDITIONALLY_THROWS ")
	}
	if f.Is(FlowFlagConditionallyBreaks) {
		sb.WriteString("CONDITIONALLY_BREAKS ")
	}
	if f.Is(FlowFlagConditionallyContinues) {
		sb.WriteString("CONDITIONALLY_CONTINUES ")
	}
	if f.Is(FlowFlagConditionallyAccessesThis) {
		sb.WriteString("CONDITIONALLY_ACCESSES_THIS ")
	}
	if f.Is(FlowFlagMayReturnNonThis) {
		sb.WriteString("MAY_RETURN_NONTHIS ")
	}
	return fmt.Sprintf("Flow[%d] %s", levels, strings.TrimSpace(sb.String()))
}

// --- Helpers ---

func cloneFieldFlags(m map[FlowPropertyRef]FieldFlags) map[FlowPropertyRef]FieldFlags {
	if m == nil {
		return nil
	}
	clone := make(map[FlowPropertyRef]FieldFlags, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

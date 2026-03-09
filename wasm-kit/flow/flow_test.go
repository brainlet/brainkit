package flow

import (
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/types"
)

// --- FlowFlags tests ---

func TestFlowFlagValues(t *testing.T) {
	if FlowFlagNone != 0 {
		t.Errorf("FlowFlagNone = %d, want 0", FlowFlagNone)
	}
	if FlowFlagReturns != 1 {
		t.Errorf("FlowFlagReturns = %d, want 1", FlowFlagReturns)
	}
	if FlowFlagTerminates != 256 {
		t.Errorf("FlowFlagTerminates = %d, want 256", FlowFlagTerminates)
	}
	if FlowFlagUncheckedContext != 1<<15 {
		t.Errorf("FlowFlagUncheckedContext = %d, want %d", FlowFlagUncheckedContext, 1<<15)
	}
	if FlowFlagInlineContext != 1<<17 {
		t.Errorf("FlowFlagInlineContext = %d, want %d", FlowFlagInlineContext, 1<<17)
	}
}

func TestFlowFlagAnyCategorical(t *testing.T) {
	expected := FlowFlagReturns | FlowFlagReturnsWrapped | FlowFlagReturnsNonNull |
		FlowFlagThrows | FlowFlagBreaks | FlowFlagContinues |
		FlowFlagAccessesThis | FlowFlagCallsSuper | FlowFlagTerminates
	if FlowFlagAnyCategorical != expected {
		t.Errorf("FlowFlagAnyCategorical = %d, want %d", FlowFlagAnyCategorical, expected)
	}
}

func TestFlowFlagAnyConditional(t *testing.T) {
	expected := FlowFlagConditionallyReturns | FlowFlagConditionallyThrows |
		FlowFlagConditionallyBreaks | FlowFlagConditionallyContinues |
		FlowFlagConditionallyAccessesThis
	if FlowFlagAnyConditional != expected {
		t.Errorf("FlowFlagAnyConditional = %d, want %d", FlowFlagAnyConditional, expected)
	}
}

// --- LocalFlags tests ---

func TestLocalFlagValues(t *testing.T) {
	if LocalFlagNone != 0 {
		t.Error("LocalFlagNone should be 0")
	}
	if LocalFlagConstant != 1 {
		t.Error("LocalFlagConstant should be 1")
	}
	if LocalFlagWrapped != 2 {
		t.Error("LocalFlagWrapped should be 2")
	}
	if LocalFlagNonNull != 4 {
		t.Error("LocalFlagNonNull should be 4")
	}
	if LocalFlagInitialized != 8 {
		t.Error("LocalFlagInitialized should be 8")
	}
}

// --- ConditionKind tests ---

func TestConditionKindValues(t *testing.T) {
	if ConditionKindUnknown != 0 {
		t.Error("ConditionKindUnknown should be 0")
	}
	if ConditionKindTrue != 1 {
		t.Error("ConditionKindTrue should be 1")
	}
	if ConditionKindFalse != 2 {
		t.Error("ConditionKindFalse should be 2")
	}
}

// --- Mock types for testing ---

type mockElement struct {
	kind int32
}

func (m *mockElement) GetElementKind() int32 { return m.kind }

type mockProgram struct {
	uncheckedAlways bool
}

func (m *mockProgram) UncheckedBehaviorAlways() bool                    { return m.uncheckedAlways }
func (m *mockProgram) Error(code int32, rng interface{}, args ...string) {}
func (m *mockProgram) ErrorRelated(code int32, rng1, rng2 interface{}, args ...string) {}
func (m *mockProgram) ElementsByName() map[string]FlowElementRef        { return nil }
func (m *mockProgram) InstancesByName() map[string]FlowElementRef       { return nil }

type mockLocal struct {
	index int32
	typ   *types.Type
	name  string
}

func (m *mockLocal) GetElementKind() int32    { return 0 }
func (m *mockLocal) FlowIndex() int32          { return m.index }
func (m *mockLocal) GetType() *types.Type     { return m.typ }
func (m *mockLocal) GetName() string          { return m.name }
func (m *mockLocal) SetName(n string)         { m.name = n }
func (m *mockLocal) SetInternalName(n string) {}
func (m *mockLocal) FlowGetParent() FlowElementRef { return nil }
func (m *mockLocal) Set(flags uint32)         {}
func (m *mockLocal) DeclarationRange() interface{}     { return nil }
func (m *mockLocal) DeclarationNameRange() interface{} { return nil }
func (m *mockLocal) DeclarationIsNative() bool         { return false }

type mockFunction struct {
	program        *mockProgram
	locals         []FlowLocalRef
	nextBreakId    int32
	breakStack     []int32
	nextInlineId   int32
	internalName   string
	isConstructor  bool
	parent         FlowElementRef
	sig            *types.Signature
	flow           *Flow
}

func newMockFunction(prog *mockProgram) *mockFunction {
	return &mockFunction{
		program:      prog,
		internalName: "test",
		parent:       &mockElement{kind: 0},
	}
}

func (m *mockFunction) GetElementKind() int32 { return ElementKindFunction }
func (m *mockFunction) Is(flags uint32) bool  {
	if flags == CommonFlagsConstructor {
		return m.isConstructor
	}
	return false
}
func (m *mockFunction) Set(flags uint32) {}
func (m *mockFunction) FlowAddLocal(typ *types.Type) FlowLocalRef {
	local := &mockLocal{index: int32(len(m.locals)), typ: typ}
	m.locals = append(m.locals, local)
	return local
}
func (m *mockFunction) FlowLocalsByIndex() []FlowLocalRef { return m.locals }
func (m *mockFunction) FlowTruncateLocals(n int)          { m.locals = m.locals[:n] }
func (m *mockFunction) FlowInternalName() string          { return m.internalName }
func (m *mockFunction) AllocBreakId() int32           { id := m.nextBreakId; m.nextBreakId++; return id }
func (m *mockFunction) PushBreakStack(id int32)       { m.breakStack = append(m.breakStack, id) }
func (m *mockFunction) PopBreakStack() int32 {
	n := len(m.breakStack)
	id := m.breakStack[n-1]
	m.breakStack = m.breakStack[:n-1]
	return id
}
func (m *mockFunction) AllocInlineId() int32                        { id := m.nextInlineId; m.nextInlineId++; return id }
func (m *mockFunction) FlowParent() FlowElementRef                 { return m.parent }
func (m *mockFunction) FlowProgram() FlowProgramRef                { return m.program }
func (m *mockFunction) FlowSignature() *types.Signature             { return m.sig }
func (m *mockFunction) FlowContextualTypeArguments() map[string]*types.Type { return nil }
func (m *mockFunction) FlowLookup(name string) FlowElementRef             { return nil }
func (m *mockFunction) GetFlow() *Flow                             { return m.flow }

// --- Flow basic tests ---

func TestFlowFlagOperations(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	// Initially no flags
	if f.Is(FlowFlagReturns) {
		t.Error("should not have Returns initially")
	}

	// Set a flag
	f.SetFlag(FlowFlagReturns)
	if !f.Is(FlowFlagReturns) {
		t.Error("should have Returns after Set")
	}

	// Set another flag
	f.SetFlag(FlowFlagThrows)
	if !f.Is(FlowFlagReturns) {
		t.Error("should still have Returns")
	}
	if !f.Is(FlowFlagThrows) {
		t.Error("should have Throws")
	}

	// IsAny
	if !f.IsAny(FlowFlagReturns | FlowFlagBreaks) {
		t.Error("IsAny should match when at least one flag is set")
	}
	if f.IsAny(FlowFlagBreaks | FlowFlagContinues) {
		t.Error("IsAny should not match when no flag is set")
	}

	// Unset
	f.UnsetFlag(FlowFlagReturns)
	if f.Is(FlowFlagReturns) {
		t.Error("should not have Returns after Unset")
	}
	if !f.Is(FlowFlagThrows) {
		t.Error("Throws should be unaffected")
	}
}

func TestDeriveConditionalFlags(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	f.SetFlag(FlowFlagReturns)
	f.SetFlag(FlowFlagBreaks)

	condi := f.DeriveConditionalFlags()
	if condi&FlowFlagConditionallyReturns == 0 {
		t.Error("should derive ConditionallyReturns from Returns")
	}
	if condi&FlowFlagConditionallyBreaks == 0 {
		t.Error("should derive ConditionallyBreaks from Breaks")
	}
	if condi&FlowFlagConditionallyThrows != 0 {
		t.Error("should not derive ConditionallyThrows without Throws")
	}
}

func TestFlowIsInline(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)

	f1 := newFlow(fn, nil)
	if f1.IsInline() {
		t.Error("non-inline flow should not be inline")
	}

	fn2 := newMockFunction(prog)
	f2 := newFlow(fn, fn2)
	if !f2.IsInline() {
		t.Error("inline flow should be inline")
	}
}

func TestFlowSourceFunction(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)

	f1 := newFlow(fn, nil)
	if f1.SourceFunction() != fn {
		t.Error("source function should be target when not inlining")
	}

	fn2 := newMockFunction(prog)
	f2 := newFlow(fn, fn2)
	if f2.SourceFunction() != fn2 {
		t.Error("source function should be inline function when inlining")
	}
}

// --- Fork tests ---

func TestFork(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	f.SetFlag(FlowFlagReturns)
	f.BreakLabel = "outer"
	f.ContinueLabel = "loop"
	f.SetLocalFlag(0, LocalFlagWrapped)
	f.SetLocalFlag(1, LocalFlagNonNull)

	child := f.Fork(false, false)

	// Child should inherit flags
	if !child.Is(FlowFlagReturns) {
		t.Error("child should inherit Returns flag")
	}
	// Child should have parent set
	if child.Parent != f {
		t.Error("child.Parent should be parent flow")
	}
	// Labels inherited
	if child.BreakLabel != "outer" {
		t.Error("child should inherit break label")
	}
	if child.ContinueLabel != "loop" {
		t.Error("child should inherit continue label")
	}
	// Local flags copied (not shared)
	if !child.IsLocalFlag(0, LocalFlagWrapped) {
		t.Error("child should have copied local flag")
	}
	// Modify child's local flags - should not affect parent
	child.UnsetLocalFlag(0, LocalFlagWrapped)
	if !f.IsLocalFlag(0, LocalFlagWrapped) {
		t.Error("parent local flags should be unaffected by child modification")
	}
}

func TestForkNewBreakContext(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	f.SetFlag(FlowFlagBreaks)
	f.SetFlag(FlowFlagConditionallyBreaks)
	f.BreakLabel = "outer"

	child := f.Fork(true, false)

	if child.Is(FlowFlagBreaks) {
		t.Error("child with new break context should not have Breaks")
	}
	if child.Is(FlowFlagConditionallyBreaks) {
		t.Error("child with new break context should not have ConditionallyBreaks")
	}
	if child.BreakLabel != "" {
		t.Error("child with new break context should have empty break label")
	}
}

func TestForkNewContinueContext(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	f.SetFlag(FlowFlagContinues)
	f.SetFlag(FlowFlagConditionallyContinues)
	f.ContinueLabel = "loop"

	child := f.Fork(false, true)

	if child.Is(FlowFlagContinues) {
		t.Error("child with new continue context should not have Continues")
	}
	if child.ContinueLabel != "" {
		t.Error("child with new continue context should have empty continue label")
	}
}

// --- Local flag tests ---

func TestLocalFlags(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	// Set flags
	f.SetLocalFlag(0, LocalFlagWrapped)
	if !f.IsLocalFlag(0, LocalFlagWrapped) {
		t.Error("should have Wrapped flag after set")
	}

	f.SetLocalFlag(0, LocalFlagNonNull)
	if !f.IsLocalFlag(0, LocalFlagWrapped|LocalFlagNonNull) {
		t.Error("should have both Wrapped and NonNull flags")
	}

	// Unset
	f.UnsetLocalFlag(0, LocalFlagWrapped)
	if f.IsLocalFlag(0, LocalFlagWrapped) {
		t.Error("should not have Wrapped after unset")
	}
	if !f.IsLocalFlag(0, LocalFlagNonNull) {
		t.Error("NonNull should be unaffected")
	}
}

func TestLocalFlagNegativeIndex(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	// Negative index should return defaultIfInlined (true by default)
	if !f.IsLocalFlag(-1, LocalFlagWrapped) {
		t.Error("negative index should return true (default for inlined)")
	}

	// SetLocalFlag with negative index should be a no-op
	f.SetLocalFlag(-1, LocalFlagWrapped)
	// No panic = success
}

func TestLocalFlagAutoGrow(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	// Setting a flag at a high index should auto-grow the slice
	f.SetLocalFlag(10, LocalFlagConstant)
	if !f.IsLocalFlag(10, LocalFlagConstant) {
		t.Error("should have Constant at index 10 after auto-grow")
	}
	if len(f.LocalFlags) < 11 {
		t.Error("LocalFlags slice should have grown")
	}
}

// --- Inherit tests ---

func TestInherit(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	parent := newFlow(fn, nil)
	parent.BreakLabel = "test"
	child := parent.Fork(false, false)

	child.SetFlag(FlowFlagReturns)
	child.SetFlag(FlowFlagTerminates)
	child.SetLocalFlag(0, LocalFlagWrapped)

	parent.Inherit(child)

	if !parent.Is(FlowFlagReturns) {
		t.Error("parent should inherit Returns")
	}
	if !parent.Is(FlowFlagTerminates) {
		t.Error("parent should inherit Terminates")
	}
	if !parent.IsLocalFlag(0, LocalFlagWrapped) {
		t.Error("parent should take child's local flags")
	}
}

func TestInheritMasksBreaksOnDifferentContext(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	parent := newFlow(fn, nil)
	parent.BreakLabel = "outer"
	child := parent.Fork(true, false) // new break context
	child.BreakLabel = "inner"

	child.SetFlag(FlowFlagBreaks)
	child.SetFlag(FlowFlagTerminates)

	parent.Inherit(child)

	// Breaks should be masked out since labels differ
	if parent.Is(FlowFlagBreaks) {
		t.Error("parent should not inherit Breaks when break labels differ")
	}
	// Terminates should also be cleared when breaks are present
	if parent.Is(FlowFlagTerminates) {
		t.Error("parent should not inherit Terminates when it was due to breaks in different context")
	}
}

// --- MergeBranch tests ---

func TestMergeBranch(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	f.SetLocalFlag(0, LocalFlagWrapped|LocalFlagNonNull)
	f.SetLocalFlag(1, LocalFlagWrapped)

	branch := f.Fork(false, false)
	branch.UnsetLocalFlag(0, LocalFlagNonNull)

	f.MergeBranch(branch)

	// Wrapped preserved (both have it)
	if !f.IsLocalFlag(0, LocalFlagWrapped) {
		t.Error("Wrapped should be preserved (both branches have it)")
	}
	// NonNull NOT preserved (branch doesn't have it)
	if f.IsLocalFlag(0, LocalFlagNonNull) {
		t.Error("NonNull should not be preserved (branch doesn't have it)")
	}
	// Index 1 wrapped preserved
	if !f.IsLocalFlag(1, LocalFlagWrapped) {
		t.Error("local 1 Wrapped should be preserved")
	}
}

// --- InheritAlternatives tests ---

func TestInheritAlternatives(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	left := f.Fork(false, false)
	left.SetFlag(FlowFlagReturns)

	right := f.Fork(false, false)
	right.SetFlag(FlowFlagReturns)

	f.InheritAlternatives(left, right)

	// Both return -> categorical Returns
	if !f.Is(FlowFlagReturns) {
		t.Error("should be Returns when both branches return")
	}
}

func TestInheritAlternativesConditional(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	left := f.Fork(false, false)
	left.SetFlag(FlowFlagReturns)

	right := f.Fork(false, false)
	// right does NOT return

	f.InheritAlternatives(left, right)

	// Only one returns -> conditional
	if f.Is(FlowFlagReturns) {
		t.Error("should not be categorically Returns when only one branch returns")
	}
	if !f.Is(FlowFlagConditionallyReturns) {
		t.Error("should be ConditionallyReturns when one branch returns")
	}
}

func TestInheritAlternativesTerminating(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	f.SetLocalFlag(0, LocalFlagWrapped)

	left := f.Fork(false, false)
	left.SetFlag(FlowFlagTerminates)

	right := f.Fork(false, false)
	right.SetLocalFlag(0, LocalFlagWrapped|LocalFlagNonNull)

	f.InheritAlternatives(left, right)

	// Left terminates, so take right's local flags
	if !f.IsLocalFlag(0, LocalFlagNonNull) {
		t.Error("should take right's NonNull since left terminates")
	}
}

func TestInheritAlternativesBothTerminate(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	left := f.Fork(false, false)
	left.SetFlag(FlowFlagTerminates)

	right := f.Fork(false, false)
	right.SetFlag(FlowFlagTerminates)

	f.InheritAlternatives(left, right)

	if !f.Is(FlowFlagTerminates) {
		t.Error("should Terminate when both branches terminate")
	}
}

// --- Control flow label tests ---

func TestControlFlowLabels(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	label1 := f.PushControlFlowLabel()
	if label1 != 0 {
		t.Errorf("first label should be 0, got %d", label1)
	}

	label2 := f.PushControlFlowLabel()
	if label2 != 1 {
		t.Errorf("second label should be 1, got %d", label2)
	}

	f.PopControlFlowLabel(label2)
	f.PopControlFlowLabel(label1)
	// No panic = success
}

func TestControlFlowLabelMismatch(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	label := f.PushControlFlowLabel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic on mismatched label")
		}
	}()
	f.PopControlFlowLabel(label + 1) // wrong label
}

// --- ResetIfNeedsRecompile tests ---

func TestResetIfNeedsRecompile(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	// Add some locals to the function
	fn.locals = []FlowLocalRef{
		&mockLocal{index: 0, typ: types.TypeI8},
		&mockLocal{index: 1, typ: types.TypeI32},
	}

	f := newFlow(fn, nil)
	f.SetLocalFlag(0, LocalFlagWrapped)
	f.SetLocalFlag(1, LocalFlagWrapped)

	other := f.Fork(false, false)
	other.UnsetLocalFlag(0, LocalFlagWrapped) // i8 not wrapped anymore

	result := f.ResetIfNeedsRecompile(other, 2)
	if !result {
		t.Error("should need recompile when short int goes from wrapped to unwrapped")
	}
	if f.IsLocalFlag(0, LocalFlagWrapped) {
		t.Error("should have unset Wrapped on the recompile")
	}
}

func TestResetIfNeedsRecompileNoChange(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	fn.locals = []FlowLocalRef{
		&mockLocal{index: 0, typ: types.TypeI32},
	}

	f := newFlow(fn, nil)
	f.SetLocalFlag(0, LocalFlagWrapped)

	other := f.Fork(false, false)
	// i32 is not a short integer, so Wrapped change doesn't trigger recompile

	result := f.ResetIfNeedsRecompile(other, 1)
	if result {
		t.Error("should not need recompile for non-short integer types")
	}
}

// --- MergeSideEffects tests ---

func TestMergeSideEffects(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	f.BreakLabel = "test"

	other := f.Fork(false, false)
	other.SetFlag(FlowFlagReturns)

	f.MergeSideEffects(other)

	// other Returns but f doesn't -> ConditionallyReturns
	if !f.Is(FlowFlagConditionallyReturns) {
		t.Error("should have ConditionallyReturns")
	}
	if f.Is(FlowFlagReturns) {
		t.Error("should not have categorical Returns")
	}
}

func TestMergeSideEffectsPreservesUnchecked(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	f.SetFlag(FlowFlagUncheckedContext)

	other := f.Fork(false, false)

	f.MergeSideEffects(other)

	if !f.Is(FlowFlagUncheckedContext) {
		t.Error("should preserve UncheckedContext across merge")
	}
}

// --- CanConversionOverflow tests ---

func TestCanConversionOverflow(t *testing.T) {
	// float to short int -> can overflow
	if !CanConversionOverflow(types.TypeF32, types.TypeI8) {
		t.Error("f32 to i8 should overflow")
	}
	// i32 to i8 -> can overflow (size 32 > 8)
	if !CanConversionOverflow(types.TypeI32, types.TypeI8) {
		t.Error("i32 to i8 should overflow")
	}
	// i8 to i8 same signedness -> no overflow
	if CanConversionOverflow(types.TypeI8, types.TypeI8) {
		t.Error("i8 to i8 should not overflow")
	}
	// i8 to u8 -> overflow (signedness mismatch)
	if !CanConversionOverflow(types.TypeI8, types.TypeU8) {
		t.Error("i8 to u8 should overflow (signedness mismatch)")
	}
	// i32 to i32 -> no overflow (not short integer target)
	if CanConversionOverflow(types.TypeI32, types.TypeI32) {
		t.Error("i32 to i32 should not overflow")
	}
}

// --- String tests ---

func TestFlowString(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	f.SetFlag(FlowFlagReturns)
	f.SetFlag(FlowFlagTerminates)

	s := f.String()
	if s != "Flow[0] RETURNS TERMINATES" {
		t.Errorf("String() = %q", s)
	}
}

func TestFlowStringWithDepth(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)
	child := f.Fork(false, false)
	grandchild := child.Fork(false, false)

	s := grandchild.String()
	if s != "Flow[2] " {
		t.Errorf("String() = %q, want %q", s, "Flow[2] ")
	}
}

// --- NoteThen / NoteElse tests ---

func TestNoteThenNoteElse(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	thenFlow := newFlow(fn, nil)
	thenFlow.SetFlag(FlowFlagReturns)

	var expr ExpressionRef = 42
	f.NoteThen(expr, thenFlow)

	if f.TrueFlows == nil {
		t.Fatal("TrueFlows should be initialized")
	}
	if f.TrueFlows[expr] != thenFlow {
		t.Error("should store the then flow")
	}

	elseFlow := newFlow(fn, nil)
	f.NoteElse(expr, elseFlow)

	if f.FalseFlows == nil {
		t.Fatal("FalseFlows should be initialized")
	}
	if f.FalseFlows[expr] != elseFlow {
		t.Error("should store the else flow")
	}
}

// --- Scoped type alias tests ---

func TestScopedTypeAlias(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	type mockTypeDef struct{}
	def := &mockTypeDef{}

	f.AddScopedTypeAlias("MyType", def)

	result := f.LookupScopedTypeAlias("MyType")
	if result != def {
		t.Error("should find scoped type alias")
	}

	if f.LookupScopedTypeAlias("Unknown") != nil {
		t.Error("should return nil for unknown alias")
	}
}

func TestScopedTypeAliasParentWalk(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	parent := newFlow(fn, nil)

	type mockTypeDef struct{}
	def := &mockTypeDef{}
	parent.AddScopedTypeAlias("ParentType", def)

	child := parent.Fork(false, false)

	result := child.LookupScopedTypeAlias("ParentType")
	if result != def {
		t.Error("should find type alias in parent flow")
	}
}

// --- CanOverflow with no module ---

func TestCanOverflowWithoutModule(t *testing.T) {
	prog := &mockProgram{}
	fn := newMockFunction(prog)
	f := newFlow(fn, nil)

	// Without module accessors, should return true (conservative) for short ints
	if !f.CanOverflow(0, types.TypeI8) {
		t.Error("CanOverflow should return true (conservative) for i8 without module")
	}

	// Non-short integers never overflow
	if f.CanOverflow(0, types.TypeI32) {
		t.Error("CanOverflow should return false for i32")
	}
}

package binaryen

import (
	"strings"
	"testing"
)

func TestModuleLifecycle(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()

	if mod.ref == nil {
		t.Fatal("module ref is nil after creation")
	}
}

func TestTypeConstants(t *testing.T) {
	// Type constants should be non-zero (except None)
	if TypeNone() != 0 {
		t.Errorf("TypeNone should be 0, got %d", TypeNone())
	}
	if TypeInt32() == 0 {
		t.Error("TypeInt32 should not be 0")
	}
	if TypeInt64() == 0 {
		t.Error("TypeInt64 should not be 0")
	}
	if TypeFloat32() == 0 {
		t.Error("TypeFloat32 should not be 0")
	}
	if TypeFloat64() == 0 {
		t.Error("TypeFloat64 should not be 0")
	}
}

func TestHeapTypeConstants(t *testing.T) {
	// Heap types should be distinct
	seen := make(map[HeapType]string)
	types := map[string]HeapType{
		"Ext":    HeapTypeExt(),
		"Func":   HeapTypeFunc(),
		"Any":    HeapTypeAny(),
		"Eq":     HeapTypeEq(),
		"I31":    HeapTypeI31(),
		"Struct": HeapTypeStruct(),
		"Array":  HeapTypeArray(),
		"String": HeapTypeString(),
		"None":   HeapTypeNone(),
		"Noext":  HeapTypeNoext(),
		"Nofunc": HeapTypeNofunc(),
	}
	for name, ht := range types {
		if prev, ok := seen[ht]; ok {
			t.Errorf("HeapType %s has same value as %s (%d)", name, prev, ht)
		}
		seen[ht] = name
	}
}

func TestTypeCreate(t *testing.T) {
	// Create a tuple type
	tuple := TypeCreate([]Type{TypeInt32(), TypeFloat64()})
	if TypeArity(tuple) != 2 {
		t.Errorf("tuple arity should be 2, got %d", TypeArity(tuple))
	}
	expanded := TypeExpand(tuple)
	if len(expanded) != 2 {
		t.Fatalf("expanded should have 2 elements, got %d", len(expanded))
	}
	if expanded[0] != TypeInt32() {
		t.Errorf("first element should be i32")
	}
	if expanded[1] != TypeFloat64() {
		t.Errorf("second element should be f64")
	}
}

func TestFeatureFlags(t *testing.T) {
	if FeatureMVP() != 0 {
		t.Errorf("FeatureMVP should be 0, got %d", FeatureMVP())
	}
	// All should be non-zero and combine with bitwise OR
	combined := FeatureAtomics() | FeatureSIMD128() | FeatureGC()
	if combined == 0 {
		t.Error("combined features should not be 0")
	}
}

func TestModuleValidateEmpty(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()
	if !mod.Validate() {
		t.Error("empty module should be valid")
	}
}

func TestModuleEmitText(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()
	text := mod.EmitText()
	if !strings.Contains(text, "(module") {
		t.Errorf("emitted text should contain '(module', got: %s", text)
	}
}

func TestModuleEmitBinary(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()
	binary := mod.EmitBinary()
	// Wasm binary starts with magic bytes: \0asm
	if len(binary) < 8 {
		t.Fatalf("binary too short: %d bytes", len(binary))
	}
	if binary[0] != 0x00 || binary[1] != 0x61 || binary[2] != 0x73 || binary[3] != 0x6d {
		t.Errorf("binary should start with wasm magic, got: %x", binary[:4])
	}
}

func TestModuleFeatures(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()

	features := FeatureAtomics() | FeatureSIMD128()
	mod.SetFeatures(features)
	got := mod.GetFeatures()
	if got != features {
		t.Errorf("features mismatch: want %d, got %d", features, got)
	}
}

func TestStringPool(t *testing.T) {
	pool := newStringPool()
	defer pool.free()

	cs1 := pool.CStr("hello")
	cs2 := pool.CStr("hello")
	if cs1 != cs2 {
		t.Error("same string should return same C pointer")
	}

	cs3 := pool.CStr("world")
	if cs3 == cs1 {
		t.Error("different strings should return different C pointers")
	}

	// Empty string returns nil
	if pool.CStr("") != nil {
		t.Error("empty string should return nil")
	}
}

func TestExpressionIDs(t *testing.T) {
	// Expression IDs should be distinct and non-zero (except Invalid)
	ids := []struct {
		name string
		id   ExpressionID
	}{
		{"Block", BlockId()},
		{"If", IfId()},
		{"Loop", LoopId()},
		{"Break", BreakId()},
		{"Call", CallId()},
		{"LocalGet", LocalGetId()},
		{"LocalSet", LocalSetId()},
		{"GlobalGet", GlobalGetId()},
		{"GlobalSet", GlobalSetId()},
		{"Load", LoadId()},
		{"Store", StoreId()},
		{"Const", ConstId()},
		{"Unary", UnaryId()},
		{"Binary", BinaryId()},
		{"Select", SelectId()},
		{"Drop", DropId()},
		{"Return", ReturnId()},
		{"Nop", NopId()},
		{"Unreachable", UnreachableId()},
	}

	seen := make(map[ExpressionID]string)
	for _, tc := range ids {
		if tc.id == InvalidId() {
			t.Errorf("%s ID should not equal InvalidId", tc.name)
		}
		if prev, ok := seen[tc.id]; ok {
			t.Errorf("%s ID (%d) collides with %s", tc.name, tc.id, prev)
		}
		seen[tc.id] = tc.name
	}
}

func TestTypeBuilder(t *testing.T) {
	// Build a simple recursive struct type: struct { i32, ref(self) }
	tb := NewTypeBuilder(1)
	tempHT := tb.GetTempHeapType(0)

	// Set as struct with two fields: i32 and nullable ref to self
	selfRef := tb.GetTempRefType(tempHT, true)
	tb.SetStructType(0,
		[]Type{TypeInt32(), selfRef},
		[]PackedType{PackedTypeNotPacked(), PackedTypeNotPacked()},
		[]bool{false, true},
	)
	tb.SetOpen(0)

	heapTypes, err := tb.BuildAndDispose()
	if err != nil {
		t.Fatalf("TypeBuilder failed: %v", err)
	}
	if len(heapTypes) != 1 {
		t.Fatalf("expected 1 heap type, got %d", len(heapTypes))
	}
	if heapTypes[0] == 0 {
		t.Error("heap type should not be zero")
	}
}

func TestTypeBuilderSignature(t *testing.T) {
	tb := NewTypeBuilder(1)
	tb.SetSignatureType(0, TypeInt32(), TypeInt64())
	heapTypes, err := tb.BuildAndDispose()
	if err != nil {
		t.Fatalf("TypeBuilder failed: %v", err)
	}
	if len(heapTypes) != 1 {
		t.Fatalf("expected 1 heap type, got %d", len(heapTypes))
	}
}

func TestTypeBuilderArray(t *testing.T) {
	tb := NewTypeBuilder(1)
	tb.SetArrayType(0, TypeFloat64(), PackedTypeNotPacked(), true)
	heapTypes, err := tb.BuildAndDispose()
	if err != nil {
		t.Fatalf("TypeBuilder failed: %v", err)
	}
	if len(heapTypes) != 1 {
		t.Fatalf("expected 1 heap type, got %d", len(heapTypes))
	}
}

func TestOpConstants(t *testing.T) {
	// Ops are only unique within their category (unary vs binary vs SIMD etc.)
	// Verify binary ops are distinct from each other
	binaryOps := map[string]Op{
		"AddInt32": AddInt32(),
		"SubInt32": SubInt32(),
		"MulInt32": MulInt32(),
		"EqInt32":  EqInt32(),
	}
	seen := make(map[Op]string)
	for name, op := range binaryOps {
		if prev, ok := seen[op]; ok {
			t.Errorf("BinaryOp %s has same value as %s (%d)", name, prev, op)
		}
		seen[op] = name
	}

	// Verify unary ops are distinct from each other
	unaryOps := map[string]Op{
		"ClzInt32":    ClzInt32(),
		"CtzInt32":    CtzInt32(),
		"PopcntInt32": PopcntInt32(),
		"NegFloat64":  NegFloat64(),
	}
	seen2 := make(map[Op]string)
	for name, op := range unaryOps {
		if prev, ok := seen2[op]; ok {
			t.Errorf("UnaryOp %s has same value as %s (%d)", name, prev, op)
		}
		seen2[op] = name
	}
}

func TestExpressionBuilders(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()

	// Test basic expression builders
	c42 := mod.ConstInt32(42)
	if c42 == 0 {
		t.Fatal("ConstInt32(42) returned 0")
	}

	c64 := mod.ConstInt64(100)
	if c64 == 0 {
		t.Fatal("ConstInt64(100) returned 0")
	}

	cf32 := mod.ConstFloat32(3.14)
	if cf32 == 0 {
		t.Fatal("ConstFloat32(3.14) returned 0")
	}

	cf64 := mod.ConstFloat64(2.718)
	if cf64 == 0 {
		t.Fatal("ConstFloat64(2.718) returned 0")
	}

	// Nop
	nop := mod.Nop()
	if nop == 0 {
		t.Fatal("Nop returned 0")
	}

	// Unreachable
	unr := mod.Unreachable()
	if unr == 0 {
		t.Fatal("Unreachable returned 0")
	}

	// Drop
	drop := mod.Drop(c42)
	if drop == 0 {
		t.Fatal("Drop returned 0")
	}

	// Return
	ret := mod.Return(c42)
	if ret == 0 {
		t.Fatal("Return returned 0")
	}

	// LocalGet / LocalSet
	lg := mod.LocalGet(0, TypeInt32())
	if lg == 0 {
		t.Fatal("LocalGet returned 0")
	}
	ls := mod.LocalSet(0, c42)
	if ls == 0 {
		t.Fatal("LocalSet returned 0")
	}

	// Binary expression
	add := mod.Binary(AddInt32(), c42, c42)
	if add == 0 {
		t.Fatal("Binary(AddInt32) returned 0")
	}

	// Unary expression
	clz := mod.Unary(ClzInt32(), c42)
	if clz == 0 {
		t.Fatal("Unary(ClzInt32) returned 0")
	}

	// Block
	block := mod.Block("myblock", []ExpressionRef{nop, c42}, TypeInt32())
	if block == 0 {
		t.Fatal("Block returned 0")
	}

	// If
	cond := mod.ConstInt32(1)
	ifExpr := mod.If(cond, c42, c64)
	if ifExpr == 0 {
		t.Fatal("If returned 0")
	}

	// Select
	sel := mod.Select(cond, c42, c42)
	if sel == 0 {
		t.Fatal("Select returned 0")
	}
}

func TestAddFunctionAndExport(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()

	// Create a function that returns i32.const(42)
	body := mod.ConstInt32(42)
	fn := mod.AddFunction("answer", TypeNone(), TypeInt32(), nil, body)
	if fn == 0 {
		t.Fatal("AddFunction returned 0")
	}

	// Export the function
	exp := mod.AddFunctionExport("answer", "answer")
	if exp == 0 {
		t.Fatal("AddFunctionExport returned 0")
	}

	// Validate
	if !mod.Validate() {
		t.Error("module with function should be valid")
	}

	// Check WAT output
	text := mod.EmitText()
	if !strings.Contains(text, "answer") {
		t.Errorf("WAT should contain 'answer', got: %s", text)
	}
	if !strings.Contains(text, "i32.const 42") {
		t.Errorf("WAT should contain 'i32.const 42', got: %s", text)
	}
}

func TestAddGlobal(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()

	init := mod.ConstInt32(0)
	g := mod.AddGlobal("counter", TypeInt32(), true, init)
	if g == 0 {
		t.Fatal("AddGlobal returned 0")
	}

	if !mod.Validate() {
		t.Error("module with global should be valid")
	}
}

func TestRelooper(t *testing.T) {
	mod := NewModule()
	defer mod.Dispose()

	// Build a simple CFG: entry -> exit
	r := mod.NewRelooper()
	entry := r.AddBlock(mod.Nop())
	exit := r.AddBlock(mod.Return(mod.ConstInt32(42)))
	RelooperAddBranch(entry, exit, 0, 0)

	// Add a local for label helper
	body := r.RenderAndDispose(entry, 0)
	if body == 0 {
		t.Fatal("RenderAndDispose returned 0")
	}

	fn := mod.AddFunction("relooper_test", TypeNone(), TypeInt32(), []Type{TypeInt32()}, body)
	if fn == 0 {
		t.Fatal("AddFunction with relooper body returned 0")
	}

	if !mod.Validate() {
		t.Error("module with relooper function should be valid")
	}
}

func TestFullModule(t *testing.T) {
	// Build a module that computes factorial-like: return n * 2
	mod := NewModule()
	defer mod.Dispose()

	// Enable features
	mod.SetFeatures(FeatureAll())

	// Create function: i32 double(i32 n) { return n * 2; }
	paramN := mod.LocalGet(0, TypeInt32())
	two := mod.ConstInt32(2)
	body := mod.Binary(MulInt32(), paramN, two)

	fn := mod.AddFunction("double", TypeInt32(), TypeInt32(), nil, body)
	if fn == 0 {
		t.Fatal("AddFunction returned 0")
	}
	mod.AddFunctionExport("double", "double")

	if !mod.Validate() {
		t.Error("double module should be valid")
	}

	// Check WAT
	text := mod.EmitText()
	if !strings.Contains(text, "double") {
		t.Errorf("WAT should contain 'double'")
	}
	if !strings.Contains(text, "i32.mul") {
		t.Errorf("WAT should contain 'i32.mul'")
	}

	// Check binary
	binary := mod.EmitBinary()
	if len(binary) < 8 {
		t.Fatalf("binary too short: %d bytes", len(binary))
	}
	if binary[0] != 0x00 || binary[1] != 0x61 || binary[2] != 0x73 || binary[3] != 0x6d {
		t.Errorf("binary should start with wasm magic")
	}
}

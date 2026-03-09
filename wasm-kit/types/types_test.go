package types

import (
	"testing"
)

// --- TypeKind tests ---

func TestTypeKindValues(t *testing.T) {
	if TypeKindBool != 0 {
		t.Errorf("TypeKindBool = %d, want 0", TypeKindBool)
	}
	if TypeKindI8 != 1 {
		t.Errorf("TypeKindI8 = %d, want 1", TypeKindI8)
	}
	if TypeKindI32 != 3 {
		t.Errorf("TypeKindI32 = %d, want 3", TypeKindI32)
	}
	if TypeKindI64 != 4 {
		t.Errorf("TypeKindI64 = %d, want 4", TypeKindI64)
	}
	if TypeKindF32 != 11 {
		t.Errorf("TypeKindF32 = %d, want 11", TypeKindF32)
	}
	if TypeKindF64 != 12 {
		t.Errorf("TypeKindF64 = %d, want 12", TypeKindF64)
	}
	if TypeKindV128 != 13 {
		t.Errorf("TypeKindV128 = %d, want 13", TypeKindV128)
	}
	if TypeKindVoid != 25 {
		t.Errorf("TypeKindVoid = %d, want 25", TypeKindVoid)
	}
}

// --- TypeFlags tests ---

func TestTypeFlagValues(t *testing.T) {
	if TypeFlagNone != 0 {
		t.Errorf("TypeFlagNone = %d, want 0", TypeFlagNone)
	}
	if TypeFlagSigned != 1 {
		t.Errorf("TypeFlagSigned = %d, want 1", TypeFlagSigned)
	}
	if TypeFlagUnsigned != 2 {
		t.Errorf("TypeFlagUnsigned = %d, want 2", TypeFlagUnsigned)
	}
	if TypeFlagInteger != 4 {
		t.Errorf("TypeFlagInteger = %d, want 4", TypeFlagInteger)
	}
	if TypeFlagFunction != 1<<13 {
		t.Errorf("TypeFlagFunction = %d, want %d", TypeFlagFunction, 1<<13)
	}
}

// --- NewType tests ---

func TestNewType(t *testing.T) {
	typ := NewType(TypeKindI32, TypeFlagSigned|TypeFlagInteger|TypeFlagValue, 32)
	if typ.Kind != TypeKindI32 {
		t.Errorf("Kind = %v, want TypeKindI32", typ.Kind)
	}
	if typ.Flags != TypeFlagSigned|TypeFlagInteger|TypeFlagValue {
		t.Errorf("Flags = %v, want Signed|Integer|Value", typ.Flags)
	}
	if typ.Size != 32 {
		t.Errorf("Size = %d, want 32", typ.Size)
	}
	// Non-nullable type should point to itself
	if typ.nonNullableType != typ {
		t.Error("nonNullableType should point to self for non-nullable types")
	}
}

func TestNewTypeNullable(t *testing.T) {
	typ := NewType(TypeKindExtern, TypeFlagExternal|TypeFlagReference|TypeFlagNullable, 0)
	// Nullable type should point to itself
	if typ.nullableType != typ {
		t.Error("nullableType should point to self for nullable types")
	}
}

// --- Flag tests on built-in types ---

func TestTypeI32Flags(t *testing.T) {
	if !TypeI32.Is(TypeFlagSigned | TypeFlagInteger | TypeFlagValue) {
		t.Error("TypeI32 should be Signed|Integer|Value")
	}
	if !TypeI32.IsIntegerValue() {
		t.Error("TypeI32.IsIntegerValue() should be true")
	}
	if !TypeI32.IsSignedIntegerValue() {
		t.Error("TypeI32.IsSignedIntegerValue() should be true")
	}
	if TypeI32.IsShortIntegerValue() {
		t.Error("TypeI32.IsShortIntegerValue() should be false")
	}
	if TypeI32.IsLongIntegerValue() {
		t.Error("TypeI32.IsLongIntegerValue() should be false")
	}
	if TypeI32.IsFloatValue() {
		t.Error("TypeI32.IsFloatValue() should be false")
	}
	if TypeI32.IsReference() {
		t.Error("TypeI32.IsReference() should be false")
	}
}

func TestTypeU8Flags(t *testing.T) {
	if !TypeU8.IsUnsignedIntegerValue() {
		t.Error("TypeU8.IsUnsignedIntegerValue() should be true")
	}
	if !TypeU8.IsShortIntegerValue() {
		t.Error("TypeU8.IsShortIntegerValue() should be true")
	}
	if TypeU8.IsSignedIntegerValue() {
		t.Error("TypeU8.IsSignedIntegerValue() should be false")
	}
}

func TestTypeI64Flags(t *testing.T) {
	if !TypeI64.IsLongIntegerValue() {
		t.Error("TypeI64.IsLongIntegerValue() should be true")
	}
	if TypeI64.IsShortIntegerValue() {
		t.Error("TypeI64.IsShortIntegerValue() should be false")
	}
}

func TestTypeF32Flags(t *testing.T) {
	if !TypeF32.IsFloatValue() {
		t.Error("TypeF32.IsFloatValue() should be true")
	}
	if TypeF32.IsIntegerValue() {
		t.Error("TypeF32.IsIntegerValue() should be false")
	}
	if !TypeF32.IsNumericValue() {
		t.Error("TypeF32.IsNumericValue() should be true")
	}
}

func TestTypeV128Flags(t *testing.T) {
	if !TypeV128.IsVectorValue() {
		t.Error("TypeV128.IsVectorValue() should be true")
	}
	if TypeV128.IsIntegerValue() {
		t.Error("TypeV128.IsIntegerValue() should be false")
	}
}

func TestTypeBoolFlags(t *testing.T) {
	if !TypeBool.IsBooleanValue() {
		t.Error("TypeBool.IsBooleanValue() should be true")
	}
	// Bool is integer under the hood
	if !TypeBool.IsIntegerValue() {
		t.Error("TypeBool.IsIntegerValue() should be true")
	}
	if !TypeBool.IsShortIntegerValue() {
		t.Error("TypeBool.IsShortIntegerValue() should be true")
	}
}

func TestTypeExternFlags(t *testing.T) {
	if !TypeExtern.IsExternalReference() {
		t.Error("TypeExtern.IsExternalReference() should be true")
	}
	if !TypeExtern.IsReference() {
		t.Error("TypeExtern.IsReference() should be true")
	}
	if TypeExtern.IsInternalReference() {
		t.Error("TypeExtern.IsInternalReference() should be false")
	}
}

func TestTypeIsizeVaryingFlag(t *testing.T) {
	if !TypeIsize32.IsVaryingIntegerValue() {
		t.Error("TypeIsize32.IsVaryingIntegerValue() should be true")
	}
	if !TypeIsize64.IsVaryingIntegerValue() {
		t.Error("TypeIsize64.IsVaryingIntegerValue() should be true")
	}
}

// --- Built-in type identity tests ---

func TestTypeAutoIdentity(t *testing.T) {
	if TypeAuto == TypeI32 {
		t.Error("TypeAuto must be a distinct instance from TypeI32")
	}
	if TypeAuto.Kind != TypeKindI32 {
		t.Error("TypeAuto.Kind should be TypeKindI32")
	}
}

// --- IntType tests ---

func TestIntType(t *testing.T) {
	tests := []struct {
		input    *Type
		expected *Type
	}{
		{TypeI32, TypeI32},
		{TypeI8, TypeI8},
		{TypeI16, TypeI16},
		{TypeI64, TypeI64},
		{TypeU8, TypeU8},
		{TypeU16, TypeU16},
		{TypeU32, TypeU32},
		{TypeU64, TypeU64},
		{TypeF32, TypeI32},
		{TypeF64, TypeI64},
		{TypeBool, TypeI32},
		{TypeIsize32, TypeIsize32},
		{TypeIsize64, TypeIsize64},
		{TypeUsize32, TypeUsize32},
		{TypeUsize64, TypeUsize64},
		{TypeAuto, TypeAuto},
	}
	for _, tc := range tests {
		result := tc.input.IntType()
		if result != tc.expected {
			t.Errorf("%v.IntType() = %v, want %v", tc.input.KindToString(), result.KindToString(), tc.expected.KindToString())
		}
	}
}

// --- ExceptVoid tests ---

func TestExceptVoid(t *testing.T) {
	if TypeVoid.ExceptVoid() != TypeAuto {
		t.Error("TypeVoid.ExceptVoid() should return TypeAuto")
	}
	if TypeI32.ExceptVoid() != TypeI32 {
		t.Error("TypeI32.ExceptVoid() should return TypeI32")
	}
}

// --- ByteSize tests ---

func TestByteSize(t *testing.T) {
	if TypeI8.ByteSize() != 1 {
		t.Errorf("TypeI8.ByteSize() = %d, want 1", TypeI8.ByteSize())
	}
	if TypeI16.ByteSize() != 2 {
		t.Errorf("TypeI16.ByteSize() = %d, want 2", TypeI16.ByteSize())
	}
	if TypeI32.ByteSize() != 4 {
		t.Errorf("TypeI32.ByteSize() = %d, want 4", TypeI32.ByteSize())
	}
	if TypeI64.ByteSize() != 8 {
		t.Errorf("TypeI64.ByteSize() = %d, want 8", TypeI64.ByteSize())
	}
	if TypeV128.ByteSize() != 16 {
		t.Errorf("TypeV128.ByteSize() = %d, want 16", TypeV128.ByteSize())
	}
	if TypeBool.ByteSize() != 1 {
		t.Errorf("TypeBool.ByteSize() = %d, want 1", TypeBool.ByteSize())
	}
}

// --- AlignLog2 tests ---

func TestAlignLog2(t *testing.T) {
	if TypeI8.AlignLog2() != 0 {
		t.Errorf("TypeI8.AlignLog2() = %d, want 0", TypeI8.AlignLog2())
	}
	if TypeI16.AlignLog2() != 1 {
		t.Errorf("TypeI16.AlignLog2() = %d, want 1", TypeI16.AlignLog2())
	}
	if TypeI32.AlignLog2() != 2 {
		t.Errorf("TypeI32.AlignLog2() = %d, want 2", TypeI32.AlignLog2())
	}
	if TypeI64.AlignLog2() != 3 {
		t.Errorf("TypeI64.AlignLog2() = %d, want 3", TypeI64.AlignLog2())
	}
}

// --- IsMemory tests ---

func TestIsMemory(t *testing.T) {
	memoryTypes := []*Type{
		TypeBool, TypeI8, TypeI16, TypeI32, TypeI64, TypeIsize32,
		TypeU8, TypeU16, TypeU32, TypeU64, TypeUsize32,
		TypeF32, TypeF64, TypeV128,
	}
	for _, typ := range memoryTypes {
		if !typ.IsMemory() {
			t.Errorf("%s.IsMemory() should be true", typ.KindToString())
		}
	}
	nonMemory := []*Type{TypeVoid, TypeExtern, TypeFunc, TypeAnyRef}
	for _, typ := range nonMemory {
		if typ.IsMemory() {
			t.Errorf("%s.IsMemory() should be false", typ.KindToString())
		}
	}
}

// --- Equals tests ---

func TestTypeEquals(t *testing.T) {
	// Value types: same kind = equal
	if !TypeI32.Equals(TypeI32) {
		t.Error("TypeI32 should equal itself")
	}
	if TypeI32.Equals(TypeI64) {
		t.Error("TypeI32 should not equal TypeI64")
	}
	// TypeI32.Equals(TypeAuto) is true because Equals checks structural equality (same kind).
	// They are distinguished by pointer identity (TypeI32 != TypeAuto), not by Equals.
	if !TypeI32.Equals(TypeAuto) {
		t.Error("TypeI32.Equals(TypeAuto) should be true (same kind structurally)")
	}
	if TypeI32 == TypeAuto {
		t.Error("TypeI32 and TypeAuto must be distinct instances (pointer identity)")
	}
}

// --- IsAssignableTo tests ---

func TestIsAssignableTo(t *testing.T) {
	// i8 assignable to i16, i32, i64
	if !TypeI8.IsAssignableTo(TypeI16, false) {
		t.Error("i8 should be assignable to i16")
	}
	if !TypeI8.IsAssignableTo(TypeI32, false) {
		t.Error("i8 should be assignable to i32")
	}
	if !TypeI8.IsAssignableTo(TypeI64, false) {
		t.Error("i8 should be assignable to i64")
	}
	// i32 NOT assignable to i8
	if TypeI32.IsAssignableTo(TypeI8, false) {
		t.Error("i32 should not be assignable to i8")
	}
	// f32 assignable to f64
	if !TypeF32.IsAssignableTo(TypeF64, false) {
		t.Error("f32 should be assignable to f64")
	}
	// f64 NOT assignable to f32
	if TypeF64.IsAssignableTo(TypeF32, false) {
		t.Error("f64 should not be assignable to f32")
	}
	// integers NOT assignable to floats beyond precision
	if !TypeI8.IsAssignableTo(TypeF32, false) {
		t.Error("i8 should be assignable to f32")
	}
	if !TypeI16.IsAssignableTo(TypeF32, false) {
		t.Error("i16 should be assignable to f32")
	}
	// i32 NOT assignable to f32 (32 > 23)
	if TypeI32.IsAssignableTo(TypeF32, false) {
		t.Error("i32 should not be assignable to f32 (too many bits)")
	}
	// i32 IS assignable to f64 (32 <= 52)
	if !TypeI32.IsAssignableTo(TypeF64, false) {
		t.Error("i32 should be assignable to f64")
	}
	// value NOT assignable to reference
	if TypeI32.IsAssignableTo(TypeExtern, false) {
		t.Error("i32 should not be assignable to extern ref")
	}
}

func TestIsAssignableToSignedness(t *testing.T) {
	// Without signedness check: i8 assignable to u16
	if !TypeI8.IsAssignableTo(TypeU16, false) {
		t.Error("i8 should be assignable to u16 (signedness not relevant)")
	}
	// With signedness check: i8 NOT assignable to u16
	if TypeI8.IsAssignableTo(TypeU16, true) {
		t.Error("i8 should not be assignable to u16 (signedness relevant)")
	}
	// bool is always assignable regardless of signedness
	if !TypeBool.IsAssignableTo(TypeU8, true) {
		t.Error("bool should be assignable to u8 even with signedness check")
	}
}

func TestIsAssignableToExternRefs(t *testing.T) {
	// extern ref types assignable within hierarchy
	if TypeAnyRef.IsAssignableTo(TypeExtern, false) {
		t.Error("anyref should not be assignable to extern")
	}
	// eq <: any
	if !TypeEq.IsAssignableTo(TypeAnyRef, false) {
		t.Error("eq should be assignable to anyref")
	}
}

func TestIsAssignableToVector(t *testing.T) {
	if !TypeV128.IsAssignableTo(TypeV128, false) {
		t.Error("v128 should be assignable to v128")
	}
}

// --- IsStrictlyAssignableTo tests ---

func TestIsStrictlyAssignableTo(t *testing.T) {
	if !TypeI32.IsStrictlyAssignableTo(TypeI32, false) {
		t.Error("i32 should be strictly assignable to i32")
	}
	// No widening in strict mode
	if TypeI8.IsStrictlyAssignableTo(TypeI32, false) {
		t.Error("i8 should NOT be strictly assignable to i32")
	}
}

// --- IsChangeableTo tests ---

func TestIsChangeableTo(t *testing.T) {
	if !TypeI32.IsChangeableTo(TypeU32) {
		t.Error("i32 should be changeable to u32")
	}
	if !TypeI64.IsChangeableTo(TypeU64) {
		t.Error("i64 should be changeable to u64")
	}
	if TypeI8.IsChangeableTo(TypeU8) {
		t.Error("i8 should not be changeable to u8 (< 32 bits, different signedness)")
	}
}

// --- ToUnsigned tests ---

func TestToUnsigned(t *testing.T) {
	tests := []struct {
		input    *Type
		expected *Type
	}{
		{TypeI8, TypeU8},
		{TypeI16, TypeU16},
		{TypeI32, TypeU32},
		{TypeI64, TypeU64},
		{TypeIsize32, TypeUsize32},
		{TypeIsize64, TypeUsize64},
		{TypeU32, TypeU32}, // already unsigned
		{TypeF32, TypeF32}, // not integer
	}
	for _, tc := range tests {
		result := tc.input.ToUnsigned()
		if result != tc.expected {
			t.Errorf("%v.ToUnsigned() got %v, want %v", tc.input.KindToString(), result.KindToString(), tc.expected.KindToString())
		}
	}
}

// --- AsNullable tests ---

func TestAsNullable(t *testing.T) {
	nullable := TypeExtern.AsNullable()
	if !nullable.IsNullableReference() {
		t.Error("nullable should be a nullable reference")
	}
	if nullable.NonNullableType() != TypeExtern {
		t.Error("nullable's non-nullable type should be TypeExtern")
	}
	// Calling AsNullable again should return the same instance
	nullable2 := TypeExtern.AsNullable()
	if nullable != nullable2 {
		t.Error("AsNullable should cache the result")
	}
}

func TestAsNullablePanicsOnNonReference(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("AsNullable on non-reference should panic")
		}
	}()
	TypeI32.AsNullable()
}

// --- NonNullableType / NullableType tests ---

func TestNonNullableType(t *testing.T) {
	// Non-nullable types point to themselves
	if TypeExtern.NonNullableType() != TypeExtern {
		t.Error("TypeExtern.NonNullableType() should be itself")
	}
	nullable := TypeExtern.AsNullable()
	if nullable.NonNullableType() != TypeExtern {
		t.Error("nullable extern NonNullableType should be TypeExtern")
	}
}

func TestNullableType(t *testing.T) {
	nullable := TypeExtern.NullableType()
	if nullable == nil {
		t.Error("TypeExtern.NullableType() should return non-nil")
	}
	if !nullable.IsNullableReference() {
		t.Error("NullableType should be nullable")
	}
	// Non-reference types return nil
	if TypeI32.NullableType() != nil {
		t.Error("TypeI32.NullableType() should return nil")
	}
}

// --- ComputeSmallIntegerShift / Mask tests ---

func TestComputeSmallIntegerShift(t *testing.T) {
	shift := TypeI8.ComputeSmallIntegerShift(TypeI32)
	if shift != 24 { // 32 - 8
		t.Errorf("i8 shift in i32 = %d, want 24", shift)
	}
	shift = TypeI16.ComputeSmallIntegerShift(TypeI32)
	if shift != 16 { // 32 - 16
		t.Errorf("i16 shift in i32 = %d, want 16", shift)
	}
}

func TestComputeSmallIntegerMask(t *testing.T) {
	// u8 in i32: unsigned, size=8, mask = ~0 >> (32-8) = 0xFF
	mask := TypeU8.ComputeSmallIntegerMask(TypeI32)
	if mask != 0xFF {
		t.Errorf("u8 mask in i32 = 0x%X, want 0xFF", mask)
	}
	// i8 in i32: signed, size=8-1=7, mask = ~0 >> (32-7) = 0x7F
	mask = TypeI8.ComputeSmallIntegerMask(TypeI32)
	if mask != 0x7F {
		t.Errorf("i8 mask in i32 = 0x%X, want 0x7F", mask)
	}
}

// --- KindToString tests ---

func TestKindToString(t *testing.T) {
	if TypeI32.KindToString() != "i32" {
		t.Errorf("TypeI32.KindToString() = %q, want %q", TypeI32.KindToString(), "i32")
	}
	if TypeF64.KindToString() != "f64" {
		t.Errorf("TypeF64.KindToString() = %q, want %q", TypeF64.KindToString(), "f64")
	}
	if TypeVoid.KindToString() != "void" {
		t.Errorf("TypeVoid.KindToString() = %q, want %q", TypeVoid.KindToString(), "void")
	}
}

// --- String / ToString tests ---

func TestTypeString(t *testing.T) {
	if TypeI32.String() != "i32" {
		t.Errorf("TypeI32.String() = %q, want %q", TypeI32.String(), "i32")
	}
	if TypeAuto.String() != "auto" {
		t.Errorf("TypeAuto.String() = %q, want %q", TypeAuto.String(), "auto")
	}
	if TypeVoid.String() != "void" {
		t.Errorf("TypeVoid.String() = %q, want %q", TypeVoid.String(), "void")
	}
}

func TestTypeToStringNullable(t *testing.T) {
	nullable := TypeExtern.AsNullable()
	s := nullable.ToString(false)
	expected := "ref_extern | null"
	if s != expected {
		t.Errorf("nullable extern ToString = %q, want %q", s, expected)
	}
	sWat := nullable.ToString(true)
	expectedWat := "ref_extern|null"
	if sWat != expectedWat {
		t.Errorf("nullable extern ToString(validWat) = %q, want %q", sWat, expectedWat)
	}
}

// --- TypesToString tests ---

func TestTypesToString(t *testing.T) {
	result := TypesToString([]*Type{TypeI32, TypeF64})
	if result != "i32,f64" {
		t.Errorf("TypesToString = %q, want %q", result, "i32,f64")
	}
	if TypesToString(nil) != "" {
		t.Error("TypesToString(nil) should return empty string")
	}
	if TypesToString([]*Type{}) != "" {
		t.Error("TypesToString([]) should return empty string")
	}
}

// --- CommonType tests ---

func TestCommonTypeValues(t *testing.T) {
	// i8 and i32: i32 is the common type (i8 assignable to i32)
	result := CommonType(TypeI8, TypeI32, nil, false)
	if result != TypeI32 {
		t.Errorf("CommonType(i8, i32) = %v, want i32", result)
	}
	// i32 and i64: i64 is common
	result = CommonType(TypeI32, TypeI64, nil, false)
	if result != TypeI64 {
		t.Errorf("CommonType(i32, i64) = %v, want i64", result)
	}
	// f32 and f64: f64 is common
	result = CommonType(TypeF32, TypeF64, nil, false)
	if result != TypeF64 {
		t.Errorf("CommonType(f32, f64) = %v, want f64", result)
	}
	// i32 and f64: nil (integer and float have no common type by simple assignability)
	// Actually i32 is assignable to f64 (32 <= 52)
	result = CommonType(TypeI32, TypeF64, nil, false)
	if result != TypeF64 {
		t.Errorf("CommonType(i32, f64) = %v, want f64", result)
	}
}

func TestCommonTypeNil(t *testing.T) {
	// incompatible types
	result := CommonType(TypeF32, TypeI64, nil, false)
	if result != nil {
		t.Errorf("CommonType(f32, i64) = %v, want nil", result)
	}
}

// --- Signature tests ---

// mockProgram implements ProgramReference for testing.
type mockProgram struct {
	usizeType        *Type
	uniqueSignatures map[string]*Signature
	nextSignatureId  uint32
}

func newMockProgram() *mockProgram {
	return &mockProgram{
		usizeType:        TypeUsize32,
		uniqueSignatures: make(map[string]*Signature),
		nextSignatureId:  0,
	}
}

func (m *mockProgram) GetUsizeType() *Type                      { return m.usizeType }
func (m *mockProgram) GetFunctionPrototype() interface{}         { return nil }
func (m *mockProgram) GetWrapperClasses() map[*Type]ClassReference { return nil }
func (m *mockProgram) GetUniqueSignatures() map[string]*Signature  { return m.uniqueSignatures }
func (m *mockProgram) GetNextSignatureId() uint32                  { return m.nextSignatureId }
func (m *mockProgram) SetNextSignatureId(id uint32)                { m.nextSignatureId = id }
func (m *mockProgram) ResolveClass(prototype interface{}, typeArguments []*Type) ClassReference {
	return nil
}

func TestSignatureCreate(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)

	if sig == nil {
		t.Fatal("CreateSignature returned nil")
	}
	if sig.ID != 0 {
		t.Errorf("first sig ID = %d, want 0", sig.ID)
	}
	if len(sig.ParameterTypes) != 1 {
		t.Errorf("ParameterTypes len = %d, want 1", len(sig.ParameterTypes))
	}
	if sig.ParameterTypes[0] != TypeI32 {
		t.Error("ParameterTypes[0] should be TypeI32")
	}
	if sig.ReturnType != TypeVoid {
		t.Error("ReturnType should be TypeVoid")
	}
	if sig.ThisType != nil {
		t.Error("ThisType should be nil")
	}
	if sig.RequiredParameters != 1 {
		t.Errorf("RequiredParameters = %d, want 1", sig.RequiredParameters)
	}
	if sig.HasRest {
		t.Error("HasRest should be false")
	}
	if prog.nextSignatureId != 1 {
		t.Errorf("nextSignatureId = %d, want 1", prog.nextSignatureId)
	}
}

func TestSignatureDeduplication(t *testing.T) {
	prog := newMockProgram()
	sig1 := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)
	sig2 := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)

	if sig1 != sig2 {
		t.Error("identical signatures should be deduplicated")
	}
	if prog.nextSignatureId != 1 {
		t.Errorf("nextSignatureId = %d, want 1 (no extra increment)", prog.nextSignatureId)
	}
}

func TestSignatureDifferent(t *testing.T) {
	prog := newMockProgram()
	sig1 := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)
	sig2 := CreateSignature(prog, []*Type{TypeI64}, TypeVoid, nil, 1, false)

	if sig1 == sig2 {
		t.Error("different signatures should not be deduplicated")
	}
	if prog.nextSignatureId != 2 {
		t.Errorf("nextSignatureId = %d, want 2", prog.nextSignatureId)
	}
}

func TestSignatureEquals(t *testing.T) {
	prog := newMockProgram()
	sig1 := CreateSignature(prog, []*Type{TypeI32, TypeF64}, TypeBool, nil, 2, false)
	// Create via a new program so it's a different instance
	prog2 := newMockProgram()
	sig2 := CreateSignature(prog2, []*Type{TypeI32, TypeF64}, TypeBool, nil, 2, false)

	if !sig1.Equals(sig2) {
		t.Error("structurally equal signatures should be equal")
	}
}

func TestSignatureNotEquals(t *testing.T) {
	prog := newMockProgram()
	sig1 := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)
	sig2 := CreateSignature(prog, []*Type{TypeI64}, TypeVoid, nil, 1, false)

	if sig1.Equals(sig2) {
		t.Error("different parameter types should not be equal")
	}
}

func TestSignatureEqualsWithThis(t *testing.T) {
	prog1 := newMockProgram()
	prog2 := newMockProgram()
	sig1 := CreateSignature(prog1, []*Type{}, TypeVoid, TypeI32, 0, false)
	sig2 := CreateSignature(prog2, []*Type{}, TypeVoid, TypeI32, 0, false)

	if !sig1.Equals(sig2) {
		t.Error("signatures with same this type should be equal")
	}

	prog3 := newMockProgram()
	sig3 := CreateSignature(prog3, []*Type{}, TypeVoid, nil, 0, false)
	if sig1.Equals(sig3) {
		t.Error("signature with this vs without should not be equal")
	}
}

func TestSignatureIsAssignableTo(t *testing.T) {
	prog := newMockProgram()
	sig1 := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)
	// same signature is assignable to itself
	if !sig1.IsAssignableTo(sig1, false) {
		t.Error("signature should be assignable to itself")
	}
}

func TestSignatureIsAssignableToCovariantReturn(t *testing.T) {
	prog1 := newMockProgram()
	prog2 := newMockProgram()
	sig1 := CreateSignature(prog1, []*Type{}, TypeI8, nil, 0, false)
	sig2 := CreateSignature(prog2, []*Type{}, TypeI32, nil, 0, false)

	// i8 return is assignable to i32 return (covariant)
	if !sig1.IsAssignableTo(sig2, false) {
		t.Error("() => i8 should be assignable to () => i32 (covariant return)")
	}
	// i32 return NOT assignable to i8 return
	if sig2.IsAssignableTo(sig1, false) {
		t.Error("() => i32 should not be assignable to () => i8")
	}
}

func TestSignatureIsAssignableToInvariantParams(t *testing.T) {
	prog1 := newMockProgram()
	prog2 := newMockProgram()
	sig1 := CreateSignature(prog1, []*Type{TypeI32}, TypeVoid, nil, 1, false)
	sig2 := CreateSignature(prog2, []*Type{TypeI64}, TypeVoid, nil, 1, false)

	// Parameters are invariant (pointer identity)
	if sig1.IsAssignableTo(sig2, false) {
		t.Error("(i32) => void should not be assignable to (i64) => void")
	}
}

func TestSignatureHasManagedOperands(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32, TypeF64}, TypeVoid, nil, 2, false)
	if sig.HasManagedOperands() {
		t.Error("signature with only value types should not have managed operands")
	}
}

func TestSignatureGetManagedOperandIndices(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32, TypeF64}, TypeVoid, nil, 2, false)
	indices := sig.GetManagedOperandIndices()
	if len(indices) != 0 {
		t.Errorf("expected 0 managed indices, got %d", len(indices))
	}
}

func TestSignatureHasVectorValueOperands(t *testing.T) {
	prog := newMockProgram()
	sig1 := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)
	if sig1.HasVectorValueOperands() {
		t.Error("sig with i32 param should not have vector operands")
	}

	prog2 := newMockProgram()
	sig2 := CreateSignature(prog2, []*Type{TypeV128}, TypeVoid, nil, 1, false)
	if !sig2.HasVectorValueOperands() {
		t.Error("sig with v128 param should have vector operands")
	}
}

func TestSignatureGetVectorValueOperandIndices(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32, TypeV128, TypeF64}, TypeVoid, nil, 3, false)
	indices := sig.GetVectorValueOperandIndices()
	if len(indices) != 1 || indices[0] != 1 {
		t.Errorf("vector operand indices = %v, want [1]", indices)
	}
}

func TestSignatureToString(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32, TypeF64}, TypeBool, nil, 2, false)
	s := sig.ToString(false)
	expected := "(i32, f64) => bool"
	if s != expected {
		t.Errorf("sig.ToString(false) = %q, want %q", s, expected)
	}
}

func TestSignatureToStringValidWat(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)
	s := sig.ToString(true)
	expected := "%28i32%29=>void"
	if s != expected {
		t.Errorf("sig.ToString(true) = %q, want %q", s, expected)
	}
}

func TestSignatureToStringEmpty(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{}, TypeVoid, nil, 0, false)
	s := sig.ToString(false)
	expected := "() => void"
	if s != expected {
		t.Errorf("sig.ToString(false) = %q, want %q", s, expected)
	}
}

func TestSignatureToStringWithThis(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, TypeI32, 1, false)
	s := sig.ToString(false)
	expected := "(this: i32, i32) => void"
	if s != expected {
		t.Errorf("sig.ToString(false) = %q, want %q", s, expected)
	}
}

func TestSignatureToStringWithOptional(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32, TypeF64}, TypeVoid, nil, 1, false)
	s := sig.ToString(false)
	expected := "(i32, f64?) => void"
	if s != expected {
		t.Errorf("sig.ToString(false) = %q, want %q", s, expected)
	}
}

func TestSignatureToStringWithRest(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32, TypeF64}, TypeVoid, nil, 1, true)
	s := sig.ToString(false)
	expected := "(i32, ...f64) => void"
	if s != expected {
		t.Errorf("sig.ToString(false) = %q, want %q", s, expected)
	}
}

func TestSignatureClone(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32, TypeF64}, TypeVoid, nil, 2, false)
	cloned := sig.Clone(1, true)

	if cloned.RequiredParameters != 1 {
		t.Errorf("cloned RequiredParameters = %d, want 1", cloned.RequiredParameters)
	}
	if !cloned.HasRest {
		t.Error("cloned HasRest should be true")
	}
	if cloned.ReturnType != TypeVoid {
		t.Error("cloned ReturnType should be TypeVoid")
	}
	if len(cloned.ParameterTypes) != 2 {
		t.Errorf("cloned ParameterTypes len = %d, want 2", len(cloned.ParameterTypes))
	}
}

func TestSignatureCloneDefault(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)
	cloned := sig.CloneDefault()

	if !sig.Equals(cloned) {
		t.Error("default clone should equal original")
	}
}

func TestSignatureType(t *testing.T) {
	prog := newMockProgram()
	sig := CreateSignature(prog, []*Type{TypeI32}, TypeVoid, nil, 1, false)

	sigType := sig.Type
	if sigType == nil {
		t.Fatal("sig.Type should not be nil")
	}
	if !sigType.IsReference() {
		t.Error("signature type should be a reference")
	}
	if sigType.SignatureReference != sig {
		t.Error("signature type should reference back to signature")
	}
}

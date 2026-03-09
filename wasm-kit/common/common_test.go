package common

import (
	"testing"
)

func TestCommonFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     CommonFlags
		expected uint32
	}{
		{"None", CommonFlagsNone, 0},
		{"Import", CommonFlagsImport, 1},
		{"Export", CommonFlagsExport, 2},
		{"Declare", CommonFlagsDeclare, 4},
		{"Const", CommonFlagsConst, 8},
		{"Let", CommonFlagsLet, 16},
		{"Static", CommonFlagsStatic, 32},
		{"Readonly", CommonFlagsReadonly, 64},
		{"Abstract", CommonFlagsAbstract, 128},
		{"Public", CommonFlagsPublic, 256},
		{"Private", CommonFlagsPrivate, 512},
		{"Protected", CommonFlagsProtected, 1024},
		{"Get", CommonFlagsGet, 2048},
		{"Set", CommonFlagsSet, 4096},
		{"Override", CommonFlagsOverride, 8192},
		{"DefinitelyAssigned", CommonFlagsDefinitelyAssigned, 16384},
		{"Ambient", CommonFlagsAmbient, 32768},
		{"Generic", CommonFlagsGeneric, 65536},
		{"GenericContext", CommonFlagsGenericContext, 131072},
		{"Instance", CommonFlagsInstance, 262144},
		{"Constructor", CommonFlagsConstructor, 524288},
		{"ModuleExport", CommonFlagsModuleExport, 1048576},
		{"ModuleImport", CommonFlagsModuleImport, 2097152},
		{"Resolved", CommonFlagsResolved, 4194304},
		{"Compiled", CommonFlagsCompiled, 8388608},
		{"Errored", CommonFlagsErrored, 16777216},
		{"Inlined", CommonFlagsInlined, 33554432},
		{"Scoped", CommonFlagsScoped, 67108864},
		{"Stub", CommonFlagsStub, 134217728},
		{"Overridden", CommonFlagsOverridden, 268435456},
		{"Closure", CommonFlagsClosure, 536870912},
		{"Quoted", CommonFlagsQuoted, 1073741824},
		{"InternallyNullable", CommonFlagsInternallyNullable, 2147483648},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint32(tt.flag) != tt.expected {
				t.Errorf("CommonFlags%s = %d, want %d", tt.name, uint32(tt.flag), tt.expected)
			}
		})
	}
}

func TestCommonFlagsBitwise(t *testing.T) {
	combined := CommonFlagsImport | CommonFlagsExport | CommonFlagsConst
	if combined&CommonFlagsImport == 0 {
		t.Error("expected Import flag to be set")
	}
	if combined&CommonFlagsExport == 0 {
		t.Error("expected Export flag to be set")
	}
	if combined&CommonFlagsConst == 0 {
		t.Error("expected Const flag to be set")
	}
	if combined&CommonFlagsDeclare != 0 {
		t.Error("expected Declare flag to not be set")
	}
}

func TestFeature(t *testing.T) {
	tests := []struct {
		name     string
		feature  Feature
		expected uint32
	}{
		{"None", FeatureNone, 0},
		{"SignExtension", FeatureSignExtension, 1},
		{"MutableGlobals", FeatureMutableGlobals, 2},
		{"NontrappingF2I", FeatureNontrappingF2I, 4},
		{"BulkMemory", FeatureBulkMemory, 8},
		{"Simd", FeatureSimd, 16},
		{"Threads", FeatureThreads, 32},
		{"ExceptionHandling", FeatureExceptionHandling, 64},
		{"TailCalls", FeatureTailCalls, 128},
		{"ReferenceTypes", FeatureReferenceTypes, 256},
		{"MultiValue", FeatureMultiValue, 512},
		{"GC", FeatureGC, 1024},
		{"Memory64", FeatureMemory64, 2048},
		{"RelaxedSimd", FeatureRelaxedSimd, 4096},
		{"ExtendedConst", FeatureExtendedConst, 8192},
		{"Stringref", FeatureStringref, 16384},
		{"All", FeatureAll, 32767},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint32(tt.feature) != tt.expected {
				t.Errorf("Feature%s = %d, want %d", tt.name, uint32(tt.feature), tt.expected)
			}
		})
	}
}

func TestFeatureToString(t *testing.T) {
	tests := []struct {
		feature  Feature
		expected string
	}{
		{FeatureSignExtension, "sign-extension"},
		{FeatureMutableGlobals, "mutable-globals"},
		{FeatureNontrappingF2I, "nontrapping-f2i"},
		{FeatureBulkMemory, "bulk-memory"},
		{FeatureSimd, "simd"},
		{FeatureThreads, "threads"},
		{FeatureExceptionHandling, "exception-handling"},
		{FeatureTailCalls, "tail-calls"},
		{FeatureReferenceTypes, "reference-types"},
		{FeatureMultiValue, "multi-value"},
		{FeatureGC, "gc"},
		{FeatureMemory64, "memory64"},
		{FeatureRelaxedSimd, "relaxed-simd"},
		{FeatureExtendedConst, "extended-const"},
		{FeatureStringref, "stringref"},
		{FeatureNone, ""},
	}
	for _, tt := range tests {
		result := FeatureToString(tt.feature)
		if result != tt.expected {
			t.Errorf("FeatureToString(%d) = %q, want %q", tt.feature, result, tt.expected)
		}
	}
}

func TestCommonNames(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"Empty", CommonNameEmpty, ""},
		{"I8", CommonNameI8, "i8"},
		{"I32", CommonNameI32, "i32"},
		{"I64", CommonNameI64, "i64"},
		{"U32", CommonNameU32, "u32"},
		{"Usize", CommonNameUsize, "usize"},
		{"Bool", CommonNameBool, "bool"},
		{"F32", CommonNameF32, "f32"},
		{"F64", CommonNameF64, "f64"},
		{"V128", CommonNameV128, "v128"},
		{"Void", CommonNameVoid, "void"},
		{"Null", CommonNameNull, "null"},
		{"This", CommonNameThis, "this"},
		{"Super", CommonNameSuper, "super"},
		{"Constructor", CommonNameConstructor, "constructor"},
		{"Alloc", CommonNameAlloc, "__alloc"},
		{"Realloc", CommonNameRealloc, "__realloc"},
		{"Free", CommonNameFree, "__free"},
		{"New", CommonNameNew, "__new"},
		{"Block", CommonNameBlock, "~lib/rt/common/BLOCK"},
		{"Object_", CommonNameObject_, "~lib/rt/common/OBJECT"},
		{"DefaultMemory", CommonNameDefaultMemory, "0"},
		{"DefaultTable", CommonNameDefaultTable, "0"},
		{"ASCTarget", CommonNameASCTarget, "ASC_TARGET"},
		{"ASCFeatureGC", CommonNameASCFeatureGC, "ASC_FEATURE_GC"},
		{"CapString", CommonNameCapString, "String"},
		{"Array", CommonNameArray, "Array"},
		{"Map", CommonNameMap, "Map"},
		{"Error", CommonNameError, "Error"},
		{"Abort", CommonNameAbort, "abort"},
		{"EnumToString", CommonNameEnumToString, "__enum_to_string"},
		{"RefFunc", CommonNameRefFunc, "ref_func"},
		{"RefExtern", CommonNameRefExtern, "ref_extern"},
		{"I8x16", CommonNameI8x16, "i8x16"},
		{"F64x2", CommonNameF64x2, "f64x2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("CommonName%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestPathConstants(t *testing.T) {
	if LIBRARY_PREFIX != LIBRARY_SUBST+PATH_DELIMITER {
		t.Errorf("LIBRARY_PREFIX = %q, want %q", LIBRARY_PREFIX, LIBRARY_SUBST+PATH_DELIMITER)
	}
	if PATH_DELIMITER != "/" {
		t.Errorf("PATH_DELIMITER = %q, want %q", PATH_DELIMITER, "/")
	}
	if PARENT_SUBST != ".." {
		t.Errorf("PARENT_SUBST = %q, want %q", PARENT_SUBST, "..")
	}
	if GETTER_PREFIX != "get:" {
		t.Errorf("GETTER_PREFIX = %q, want %q", GETTER_PREFIX, "get:")
	}
	if SETTER_PREFIX != "set:" {
		t.Errorf("SETTER_PREFIX = %q, want %q", SETTER_PREFIX, "set:")
	}
	if INSTANCE_DELIMITER != "#" {
		t.Errorf("INSTANCE_DELIMITER = %q, want %q", INSTANCE_DELIMITER, "#")
	}
	if STATIC_DELIMITER != "." {
		t.Errorf("STATIC_DELIMITER = %q, want %q", STATIC_DELIMITER, ".")
	}
	if INNER_DELIMITER != "~" {
		t.Errorf("INNER_DELIMITER = %q, want %q", INNER_DELIMITER, "~")
	}
	if LIBRARY_SUBST != "~lib" {
		t.Errorf("LIBRARY_SUBST = %q, want %q", LIBRARY_SUBST, "~lib")
	}
	if INDEX_SUFFIX != "/index" {
		t.Errorf("INDEX_SUFFIX = %q, want %q", INDEX_SUFFIX, "/index")
	}
	if STUB_DELIMITER != "@" {
		t.Errorf("STUB_DELIMITER = %q, want %q", STUB_DELIMITER, "@")
	}
}

func TestTarget(t *testing.T) {
	if TargetJs != 0 {
		t.Errorf("TargetJs = %d, want 0", TargetJs)
	}
	if TargetWasm32 != 1 {
		t.Errorf("TargetWasm32 = %d, want 1", TargetWasm32)
	}
	if TargetWasm64 != 2 {
		t.Errorf("TargetWasm64 = %d, want 2", TargetWasm64)
	}
}

func TestRuntime(t *testing.T) {
	if RuntimeStub != 0 {
		t.Errorf("RuntimeStub = %d, want 0", RuntimeStub)
	}
	if RuntimeMinimal != 1 {
		t.Errorf("RuntimeMinimal = %d, want 1", RuntimeMinimal)
	}
	if RuntimeIncremental != 2 {
		t.Errorf("RuntimeIncremental = %d, want 2", RuntimeIncremental)
	}
}

func TestTypeinfoFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     TypeinfoFlags
		expected uint32
	}{
		{"NONE", TypeinfoFlagsNONE, 0},
		{"ARRAYBUFFERVIEW", TypeinfoFlagsARRAYBUFFERVIEW, 1},
		{"ARRAY", TypeinfoFlagsARRAY, 2},
		{"STATICARRAY", TypeinfoFlagsSTATICARRAY, 4},
		{"SET", TypeinfoFlagsSET, 8},
		{"MAP", TypeinfoFlagsMAP, 16},
		{"POINTERFREE", TypeinfoFlagsPOINTERFREE, 32},
		{"VALUE_ALIGN_0", TypeinfoFlagsVALUE_ALIGN_0, 64},
		{"VALUE_ALIGN_1", TypeinfoFlagsVALUE_ALIGN_1, 128},
		{"VALUE_ALIGN_2", TypeinfoFlagsVALUE_ALIGN_2, 256},
		{"VALUE_ALIGN_3", TypeinfoFlagsVALUE_ALIGN_3, 512},
		{"VALUE_ALIGN_4", TypeinfoFlagsVALUE_ALIGN_4, 1024},
		{"VALUE_SIGNED", TypeinfoFlagsVALUE_SIGNED, 2048},
		{"VALUE_FLOAT", TypeinfoFlagsVALUE_FLOAT, 4096},
		{"VALUE_NULLABLE", TypeinfoFlagsVALUE_NULLABLE, 8192},
		{"VALUE_MANAGED", TypeinfoFlagsVALUE_MANAGED, 16384},
		{"KEY_ALIGN_0", TypeinfoFlagsKEY_ALIGN_0, 32768},
		{"KEY_ALIGN_1", TypeinfoFlagsKEY_ALIGN_1, 65536},
		{"KEY_ALIGN_2", TypeinfoFlagsKEY_ALIGN_2, 131072},
		{"KEY_ALIGN_3", TypeinfoFlagsKEY_ALIGN_3, 262144},
		{"KEY_ALIGN_4", TypeinfoFlagsKEY_ALIGN_4, 524288},
		{"KEY_SIGNED", TypeinfoFlagsKEY_SIGNED, 1048576},
		{"KEY_FLOAT", TypeinfoFlagsKEY_FLOAT, 2097152},
		{"KEY_NULLABLE", TypeinfoFlagsKEY_NULLABLE, 4194304},
		{"KEY_MANAGED", TypeinfoFlagsKEY_MANAGED, 8388608},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint32(tt.flag) != tt.expected {
				t.Errorf("TypeinfoFlags%s = %d, want %d", tt.name, uint32(tt.flag), tt.expected)
			}
		})
	}
}

func TestTypeinfoStruct(t *testing.T) {
	info := Typeinfo{Flags: TypeinfoFlagsARRAY | TypeinfoFlagsVALUE_SIGNED}
	if info.Flags&TypeinfoFlagsARRAY == 0 {
		t.Error("expected ARRAY flag to be set")
	}
	if info.Flags&TypeinfoFlagsVALUE_SIGNED == 0 {
		t.Error("expected VALUE_SIGNED flag to be set")
	}
	if info.Flags&TypeinfoFlagsMAP != 0 {
		t.Error("expected MAP flag to not be set")
	}
}

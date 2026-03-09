package program

import (
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// --- helpers ---

func newTestProgram() *Program {
	return NewProgram(&Options{UsizeType: types.TypeI32}, nil)
}

// --- ElementKind tests ---

func TestElementKindValues(t *testing.T) {
	tests := []struct {
		name string
		kind ElementKind
		want int32
	}{
		{"Global", ElementKindGlobal, 0},
		{"Local", ElementKindLocal, 1},
		{"Enum", ElementKindEnum, 2},
		{"EnumValue", ElementKindEnumValue, 3},
		{"FunctionPrototype", ElementKindFunctionPrototype, 4},
		{"Function", ElementKindFunction, 5},
		{"ClassPrototype", ElementKindClassPrototype, 6},
		{"Class", ElementKindClass, 7},
		{"InterfacePrototype", ElementKindInterfacePrototype, 8},
		{"Interface", ElementKindInterface, 9},
		{"PropertyPrototype", ElementKindPropertyPrototype, 10},
		{"Property", ElementKindProperty, 11},
		{"Namespace", ElementKindNamespace, 12},
		{"File", ElementKindFile, 13},
		{"TypeDefinition", ElementKindTypeDefinition, 14},
		{"IndexSignature", ElementKindIndexSignature, 15},
	}
	for _, tt := range tests {
		if tt.kind != tt.want {
			t.Errorf("ElementKind%s = %d, want %d", tt.name, tt.kind, tt.want)
		}
	}
}

// --- DecoratorFlags tests ---

func TestDecoratorFlagValues(t *testing.T) {
	tests := []struct {
		name string
		flag DecoratorFlags
		want uint32
	}{
		{"None", DecoratorFlagsNone, 0},
		{"Global", DecoratorFlagsGlobal, 1},
		{"OperatorBinary", DecoratorFlagsOperatorBinary, 2},
		{"OperatorPrefix", DecoratorFlagsOperatorPrefix, 4},
		{"OperatorPostfix", DecoratorFlagsOperatorPostfix, 8},
		{"Unmanaged", DecoratorFlagsUnmanaged, 16},
		{"Final", DecoratorFlagsFinal, 32},
		{"Inline", DecoratorFlagsInline, 64},
		{"External", DecoratorFlagsExternal, 128},
		{"ExternalJs", DecoratorFlagsExternalJs, 256},
		{"Builtin", DecoratorFlagsBuiltin, 512},
		{"Lazy", DecoratorFlagsLazy, 1024},
		{"Unsafe", DecoratorFlagsUnsafe, 2048},
	}
	for _, tt := range tests {
		if tt.flag != tt.want {
			t.Errorf("DecoratorFlags%s = %d, want %d", tt.name, tt.flag, tt.want)
		}
	}
}

func TestDecoratorFlagsFromKind(t *testing.T) {
	tests := []struct {
		kind ast.DecoratorKind
		want DecoratorFlags
	}{
		{ast.DecoratorKindGlobal, DecoratorFlagsGlobal},
		{ast.DecoratorKindOperator, DecoratorFlagsOperatorBinary},
		{ast.DecoratorKindOperatorBinary, DecoratorFlagsOperatorBinary},
		{ast.DecoratorKindOperatorPrefix, DecoratorFlagsOperatorPrefix},
		{ast.DecoratorKindOperatorPostfix, DecoratorFlagsOperatorPostfix},
		{ast.DecoratorKindUnmanaged, DecoratorFlagsUnmanaged},
		{ast.DecoratorKindFinal, DecoratorFlagsFinal},
		{ast.DecoratorKindInline, DecoratorFlagsInline},
		{ast.DecoratorKindExternal, DecoratorFlagsExternal},
		{ast.DecoratorKindExternalJs, DecoratorFlagsExternalJs},
		{ast.DecoratorKindBuiltin, DecoratorFlagsBuiltin},
		{ast.DecoratorKindLazy, DecoratorFlagsLazy},
		{ast.DecoratorKindUnsafe, DecoratorFlagsUnsafe},
	}
	for _, tt := range tests {
		got := DecoratorFlagsFromKind(tt.kind)
		if got != tt.want {
			t.Errorf("DecoratorFlagsFromKind(%d) = %d, want %d", tt.kind, got, tt.want)
		}
	}
}

// --- ConstantValueKind tests ---

func TestConstantValueKindValues(t *testing.T) {
	if ConstantValueKindNone != 0 {
		t.Errorf("ConstantValueKindNone = %d, want 0", ConstantValueKindNone)
	}
	if ConstantValueKindInteger != 1 {
		t.Errorf("ConstantValueKindInteger = %d, want 1", ConstantValueKindInteger)
	}
	if ConstantValueKindFloat != 2 {
		t.Errorf("ConstantValueKindFloat = %d, want 2", ConstantValueKindFloat)
	}
}

// --- OperatorKind tests ---

func TestOperatorKindValues(t *testing.T) {
	if OperatorKindInvalid != 0 {
		t.Error("OperatorKindInvalid should be 0")
	}
	if OperatorKindIndexedGet != 1 {
		t.Error("OperatorKindIndexedGet should be 1")
	}
	// Just verify sequential ordering of key values
	if OperatorKindAdd != 5 {
		t.Errorf("OperatorKindAdd = %d, want 5", OperatorKindAdd)
	}
}

func TestOperatorKindFromBinaryToken(t *testing.T) {
	tests := []struct {
		token tokenizer.Token
		want  OperatorKind
	}{
		{tokenizer.TokenPlus, OperatorKindAdd},
		{tokenizer.TokenPlusEquals, OperatorKindAdd},
		{tokenizer.TokenMinus, OperatorKindSub},
		{tokenizer.TokenAsterisk, OperatorKindMul},
		{tokenizer.TokenAsteriskAsterisk, OperatorKindPow},
		{tokenizer.TokenSlash, OperatorKindDiv},
		{tokenizer.TokenPercent, OperatorKindRem},
		{tokenizer.TokenAmpersand, OperatorKindBitwiseAnd},
		{tokenizer.TokenBar, OperatorKindBitwiseOr},
		{tokenizer.TokenCaret, OperatorKindBitwiseXor},
		{tokenizer.TokenLessThanLessThan, OperatorKindBitwiseShl},
		{tokenizer.TokenGreaterThanGreaterThan, OperatorKindBitwiseShr},
		{tokenizer.TokenGreaterThanGreaterThanGreaterThan, OperatorKindBitwiseShrU},
		{tokenizer.TokenEqualsEquals, OperatorKindEq},
		{tokenizer.TokenExclamationEquals, OperatorKindNe},
		{tokenizer.TokenGreaterThan, OperatorKindGt},
		{tokenizer.TokenGreaterThanEquals, OperatorKindGe},
		{tokenizer.TokenLessThan, OperatorKindLt},
		{tokenizer.TokenLessThanEquals, OperatorKindLe},
	}
	for _, tt := range tests {
		got := OperatorKindFromBinaryToken(tt.token)
		if got != tt.want {
			t.Errorf("OperatorKindFromBinaryToken(%d) = %d, want %d", tt.token, got, tt.want)
		}
	}
}

func TestOperatorKindFromBinaryTokenInvalid(t *testing.T) {
	got := OperatorKindFromBinaryToken(tokenizer.TokenEndOfFile)
	if got != OperatorKindInvalid {
		t.Errorf("expected OperatorKindInvalid for unknown token, got %d", got)
	}
}

func TestOperatorKindFromUnaryPrefixToken(t *testing.T) {
	tests := []struct {
		token tokenizer.Token
		want  OperatorKind
	}{
		{tokenizer.TokenPlus, OperatorKindPlus},
		{tokenizer.TokenMinus, OperatorKindMinus},
		{tokenizer.TokenExclamation, OperatorKindNot},
		{tokenizer.TokenTilde, OperatorKindBitwiseNot},
		{tokenizer.TokenPlusPlus, OperatorKindPrefixInc},
		{tokenizer.TokenMinusMinus, OperatorKindPrefixDec},
	}
	for _, tt := range tests {
		got := OperatorKindFromUnaryPrefixToken(tt.token)
		if got != tt.want {
			t.Errorf("OperatorKindFromUnaryPrefixToken(%d) = %d, want %d", tt.token, got, tt.want)
		}
	}
}

func TestOperatorKindFromUnaryPostfixToken(t *testing.T) {
	if OperatorKindFromUnaryPostfixToken(tokenizer.TokenPlusPlus) != OperatorKindPostfixInc {
		t.Error("++ should be PostfixInc")
	}
	if OperatorKindFromUnaryPostfixToken(tokenizer.TokenMinusMinus) != OperatorKindPostfixDec {
		t.Error("-- should be PostfixDec")
	}
	if OperatorKindFromUnaryPostfixToken(tokenizer.TokenEndOfFile) != OperatorKindInvalid {
		t.Error("unknown token should be Invalid")
	}
}

func TestOperatorKindFromDecorator(t *testing.T) {
	tests := []struct {
		kind ast.DecoratorKind
		arg  string
		want OperatorKind
	}{
		{ast.DecoratorKindOperatorBinary, "[]", OperatorKindIndexedGet},
		{ast.DecoratorKindOperatorBinary, "[]=", OperatorKindIndexedSet},
		{ast.DecoratorKindOperatorBinary, "{}", OperatorKindUncheckedIndexedGet},
		{ast.DecoratorKindOperatorBinary, "{}=", OperatorKindUncheckedIndexedSet},
		{ast.DecoratorKindOperatorBinary, "+", OperatorKindAdd},
		{ast.DecoratorKindOperatorBinary, "-", OperatorKindSub},
		{ast.DecoratorKindOperatorBinary, "*", OperatorKindMul},
		{ast.DecoratorKindOperatorBinary, "**", OperatorKindPow},
		{ast.DecoratorKindOperatorBinary, "/", OperatorKindDiv},
		{ast.DecoratorKindOperatorBinary, "%", OperatorKindRem},
		{ast.DecoratorKindOperatorBinary, "==", OperatorKindEq},
		{ast.DecoratorKindOperatorBinary, "!=", OperatorKindNe},
		{ast.DecoratorKindOperatorBinary, ">", OperatorKindGt},
		{ast.DecoratorKindOperatorBinary, ">=", OperatorKindGe},
		{ast.DecoratorKindOperatorBinary, "<", OperatorKindLt},
		{ast.DecoratorKindOperatorBinary, "<=", OperatorKindLe},
		{ast.DecoratorKindOperatorBinary, "<<", OperatorKindBitwiseShl},
		{ast.DecoratorKindOperatorBinary, ">>", OperatorKindBitwiseShr},
		{ast.DecoratorKindOperatorBinary, ">>>", OperatorKindBitwiseShrU},
		{ast.DecoratorKindOperatorBinary, "&", OperatorKindBitwiseAnd},
		{ast.DecoratorKindOperatorBinary, "|", OperatorKindBitwiseOr},
		{ast.DecoratorKindOperatorBinary, "^", OperatorKindBitwiseXor},
		{ast.DecoratorKindOperatorPrefix, "+", OperatorKindPlus},
		{ast.DecoratorKindOperatorPrefix, "-", OperatorKindMinus},
		{ast.DecoratorKindOperatorPrefix, "++", OperatorKindPrefixInc},
		{ast.DecoratorKindOperatorPrefix, "--", OperatorKindPrefixDec},
		{ast.DecoratorKindOperatorPrefix, "!", OperatorKindNot},
		{ast.DecoratorKindOperatorPrefix, "~", OperatorKindBitwiseNot},
		{ast.DecoratorKindOperatorPostfix, "++", OperatorKindPostfixInc},
		{ast.DecoratorKindOperatorPostfix, "--", OperatorKindPostfixDec},
		// Invalid cases
		{ast.DecoratorKindOperatorBinary, "", OperatorKindInvalid},
		{ast.DecoratorKindOperatorPrefix, "", OperatorKindInvalid},
		{ast.DecoratorKindOperatorPostfix, "!", OperatorKindInvalid},
	}
	for _, tt := range tests {
		got := OperatorKindFromDecorator(tt.kind, tt.arg)
		if got != tt.want {
			t.Errorf("OperatorKindFromDecorator(%d, %q) = %d, want %d", tt.kind, tt.arg, got, tt.want)
		}
	}
}

// --- ElementBase tests ---

func TestElementBaseFlags(t *testing.T) {
	e := &ElementBase{}
	InitElementBase(e, ElementKindGlobal, "test", "test", newTestProgram(), nil)

	if e.Is(common.CommonFlagsExport) {
		t.Error("should not have Export flag initially")
	}

	e.Set(common.CommonFlagsExport)
	if !e.Is(common.CommonFlagsExport) {
		t.Error("should have Export flag after Set")
	}

	e.Set(common.CommonFlagsConst)
	if !e.Is(common.CommonFlagsExport) {
		t.Error("should still have Export")
	}
	if !e.Is(common.CommonFlagsConst) {
		t.Error("should have Const")
	}
	if !e.IsAny(common.CommonFlagsExport | common.CommonFlagsPrivate) {
		t.Error("IsAny should match when at least one flag is set")
	}
	if e.IsAny(common.CommonFlagsPrivate | common.CommonFlagsProtected) {
		t.Error("IsAny should not match when no flags are set")
	}

	e.Unset(common.CommonFlagsExport)
	if e.Is(common.CommonFlagsExport) {
		t.Error("should not have Export after Unset")
	}
}

func TestElementBaseDecoratorFlags(t *testing.T) {
	e := &ElementBase{}
	InitElementBase(e, ElementKindGlobal, "test", "test", newTestProgram(), nil)

	e.decoratorFlags = DecoratorFlagsInline | DecoratorFlagsBuiltin

	if !e.HasDecorator(DecoratorFlagsInline) {
		t.Error("should have Inline decorator")
	}
	if !e.HasDecorator(DecoratorFlagsBuiltin) {
		t.Error("should have Builtin decorator")
	}
	if e.HasDecorator(DecoratorFlagsLazy) {
		t.Error("should not have Lazy decorator")
	}
	if !e.HasAnyDecorator(DecoratorFlagsInline | DecoratorFlagsLazy) {
		t.Error("HasAnyDecorator should match when at least one is set")
	}
}

func TestElementBaseGettersSetters(t *testing.T) {
	prog := newTestProgram()
	e := &ElementBase{}
	InitElementBase(e, ElementKindNamespace, "myns", "myns", prog, nil)

	if e.GetElementKind() != ElementKindNamespace {
		t.Error("wrong element kind")
	}
	if e.GetName() != "myns" {
		t.Error("wrong name")
	}
	if e.GetInternalName() != "myns" {
		t.Error("wrong internal name")
	}
	if e.GetProgram() != prog {
		t.Error("wrong program")
	}

	e.SetInternalName("renamed")
	if e.GetInternalName() != "renamed" {
		t.Error("SetInternalName failed")
	}

	if e.GetMembers() != nil {
		t.Error("members should be nil initially")
	}
	members := make(map[string]DeclaredElement)
	e.SetMembers(members)
	if e.GetMembers() == nil {
		t.Error("members should not be nil after SetMembers")
	}
}

func TestElementBaseVisibility(t *testing.T) {
	prog := newTestProgram()

	// Public element (no private/protected flags)
	pub := &ElementBase{}
	InitElementBase(pub, ElementKindGlobal, "pub", "pub", prog, nil)

	if !pub.IsPublic() {
		t.Error("element with no visibility flags should be public")
	}
	if !pub.IsImplicitlyPublic() {
		t.Error("public without explicit public flag should be implicitly public")
	}

	// Explicit public
	explicitPub := &ElementBase{}
	InitElementBase(explicitPub, ElementKindGlobal, "ep", "ep", prog, nil)
	explicitPub.Set(common.CommonFlagsPublic)
	if !explicitPub.IsPublic() {
		t.Error("explicit public should be public")
	}
	if explicitPub.IsImplicitlyPublic() {
		t.Error("explicit public should NOT be implicitly public")
	}

	// Private element
	priv := &ElementBase{}
	InitElementBase(priv, ElementKindGlobal, "priv", "priv", prog, nil)
	priv.Set(common.CommonFlagsPrivate)
	if priv.IsPublic() {
		t.Error("private element should not be public")
	}

	// VisibilityNoLessThan
	if !pub.VisibilityNoLessThan(priv) {
		t.Error("public visibility should be no less than private")
	}
	if priv.VisibilityNoLessThan(pub) {
		t.Error("private visibility should be less than public")
	}
}

func TestElementBaseString(t *testing.T) {
	e := &ElementBase{}
	InitElementBase(e, ElementKindGlobal, "myGlobal", "myGlobal", newTestProgram(), nil)
	s := e.String()
	if s != "myGlobal, kind=0" {
		t.Errorf("String() = %q", s)
	}
}

// --- MangleInternalName tests ---

func TestMangleInternalNameNilParent(t *testing.T) {
	result := MangleInternalName("test", nil, false, false)
	if result != "test" {
		t.Errorf("MangleInternalName with nil parent = %q, want %q", result, "test")
	}
}

func TestMangleInternalNameFileParent(t *testing.T) {
	prog := newTestProgram()
	file := NewFile(prog, ast.NativeSource())

	result := MangleInternalName("myFunc", file, false, false)
	expected := file.GetInternalName() + common.PATH_DELIMITER + "myFunc"
	if result != expected {
		t.Errorf("MangleInternalName(file parent) = %q, want %q", result, expected)
	}
}

func TestMangleInternalNameFileParentAsGlobal(t *testing.T) {
	prog := newTestProgram()
	file := NewFile(prog, ast.NativeSource())

	result := MangleInternalName("myFunc", file, false, true)
	if result != "myFunc" {
		t.Errorf("MangleInternalName(file parent, asGlobal) = %q, want %q", result, "myFunc")
	}
}

// --- GetDefaultParameterName tests ---

func TestGetDefaultParameterName(t *testing.T) {
	tests := []struct {
		index int32
		want  string
	}{
		{0, "$0"},
		{1, "$1"},
		{5, "$5"},
		{0, "$0"}, // cached
	}
	for _, tt := range tests {
		got := GetDefaultParameterName(tt.index)
		if got != tt.want {
			t.Errorf("GetDefaultParameterName(%d) = %q, want %q", tt.index, got, tt.want)
		}
	}
}

// --- VariableLikeBase tests ---

func TestVariableLikeBaseConstants(t *testing.T) {
	prog := newTestProgram()
	file := NewFile(prog, ast.NativeSource())
	decl := prog.MakeNativeVariableDeclaration("x", 0)

	v := &VariableLikeBase{}
	InitVariableLikeBase(v, ElementKindGlobal, "x", file, decl)

	if v.GetConstantValueKind() != ConstantValueKindNone {
		t.Error("should have no constant value initially")
	}

	v.SetConstantIntegerValue(42, types.TypeI32)
	if v.GetConstantValueKind() != ConstantValueKindInteger {
		t.Error("should have integer constant")
	}
	if v.GetConstantIntegerValue() != 42 {
		t.Errorf("integer value = %d, want 42", v.GetConstantIntegerValue())
	}
	if !v.Is(common.CommonFlagsConst) {
		t.Error("should be const after SetConstantIntegerValue")
	}
	if !v.Is(common.CommonFlagsInlined) {
		t.Error("should be inlined after SetConstantIntegerValue")
	}

	v2 := &VariableLikeBase{}
	InitVariableLikeBase(v2, ElementKindGlobal, "y", file, prog.MakeNativeVariableDeclaration("y", 0))
	v2.SetConstantFloatValue(3.14, types.TypeF64)
	if v2.GetConstantValueKind() != ConstantValueKindFloat {
		t.Error("should have float constant")
	}
	if v2.GetConstantFloatValue() != 3.14 {
		t.Errorf("float value = %f, want 3.14", v2.GetConstantFloatValue())
	}
}

// --- Program tests ---

func TestNewProgram(t *testing.T) {
	prog := newTestProgram()
	if prog == nil {
		t.Fatal("NewProgram returned nil")
	}
	if prog.Options == nil {
		t.Error("Options should not be nil")
	}
	if prog.ElementsByNameMap == nil {
		t.Error("ElementsByNameMap should not be nil")
	}
	if prog.InstancesByNameMap == nil {
		t.Error("InstancesByNameMap should not be nil")
	}
	if prog.NativeFile == nil {
		t.Error("NativeFile should not be nil")
	}
	if prog.Resolver_ == nil {
		t.Error("Resolver should not be nil")
	}
	// NativeFile should be registered
	if _, ok := prog.ElementsByNameMap[prog.NativeFile.GetInternalName()]; !ok {
		t.Error("NativeFile should be registered in ElementsByNameMap")
	}
}

func TestProgramLookup(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile

	// NativeFile should be findable
	found := prog.Lookup(file.GetInternalName())
	if found == nil {
		t.Error("should find NativeFile by internal name")
	}

	// Unknown element
	if prog.Lookup("nonexistent") != nil {
		t.Error("should return nil for unknown element")
	}
}

func TestProgramOverhead(t *testing.T) {
	prog := newTestProgram()

	if prog.BlockOverhead() != 16 {
		t.Errorf("BlockOverhead = %d, want 16", prog.BlockOverhead())
	}
	if prog.ObjectOverhead() != 4 {
		t.Errorf("ObjectOverhead (wasm32) = %d, want 4", prog.ObjectOverhead())
	}
	if prog.TotalOverhead() != 20 {
		t.Errorf("TotalOverhead (wasm32) = %d, want 20", prog.TotalOverhead())
	}

	// wasm64
	prog64 := NewProgram(&Options{UsizeType: types.TypeI64, IsWasm64: true}, nil)
	if prog64.ObjectOverhead() != 8 {
		t.Errorf("ObjectOverhead (wasm64) = %d, want 8", prog64.ObjectOverhead())
	}
	if prog64.TotalOverhead() != 24 {
		t.Errorf("TotalOverhead (wasm64) = %d, want 24", prog64.TotalOverhead())
	}
}

func TestProgramComputeBlockStart(t *testing.T) {
	prog := newTestProgram()

	tests := []struct {
		offset int32
		want   int32
	}{
		{0, 0},
		{1, 16},
		{15, 16},
		{16, 16},
		{17, 32},
		{32, 32},
	}
	for _, tt := range tests {
		got := prog.ComputeBlockStart(tt.offset)
		if got != tt.want {
			t.Errorf("ComputeBlockStart(%d) = %d, want %d", tt.offset, got, tt.want)
		}
	}
}

func TestProgramDiagnostics(t *testing.T) {
	prog := newTestProgram()
	prog.Error(100, nil, "test")

	if len(prog.Diagnostics) == 0 {
		t.Error("should have emitted a diagnostic")
	}
}

func TestProgramString(t *testing.T) {
	prog := newTestProgram()
	s := prog.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}

func TestProgramFlowProgramRef(t *testing.T) {
	prog := newTestProgram()
	ref := prog.FlowProgramRef()
	if ref == nil {
		t.Fatal("FlowProgramRef should not be nil")
	}
	// Same ref on second call
	ref2 := prog.FlowProgramRef()
	if ref != ref2 {
		t.Error("FlowProgramRef should return cached ref")
	}

	if ref.UncheckedBehaviorAlways() {
		t.Error("UncheckedBehaviorAlways should return false (stub)")
	}
}

func TestProgramMarkModuleImport(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile

	prog.MarkModuleImport("env", "memory", file)

	if moduleMap, ok := prog.ModuleImports["env"]; !ok {
		t.Error("should have env module")
	} else if moduleMap["memory"] != file {
		t.Error("should have memory element in env module")
	}
}

func TestProgramMakeNativeVariableDeclaration(t *testing.T) {
	prog := newTestProgram()
	decl := prog.MakeNativeVariableDeclaration("x", common.CommonFlagsConst)

	if decl == nil {
		t.Fatal("MakeNativeVariableDeclaration returned nil")
	}
	if decl.Name.Text != "x" {
		t.Errorf("Name.Text = %q, want %q", decl.Name.Text, "x")
	}
}

func TestProgramMakeNativeFunctionDeclaration(t *testing.T) {
	prog := newTestProgram()
	decl := prog.MakeNativeFunctionDeclaration("foo", 0)

	if decl == nil {
		t.Fatal("MakeNativeFunctionDeclaration returned nil")
	}
	if decl.Name.Text != "foo" {
		t.Errorf("Name.Text = %q, want %q", decl.Name.Text, "foo")
	}
	// Should have ambient flag
	if decl.Flags&int32(common.CommonFlagsAmbient) == 0 {
		t.Error("native function declaration should have ambient flag")
	}
	if decl.Signature == nil {
		t.Error("Signature should not be nil")
	}
}

func TestProgramMakeNativeNamespaceDeclaration(t *testing.T) {
	prog := newTestProgram()
	decl := prog.MakeNativeNamespaceDeclaration("ns", 0)

	if decl == nil {
		t.Fatal("MakeNativeNamespaceDeclaration returned nil")
	}
	if decl.Name.Text != "ns" {
		t.Error("wrong name")
	}
}

func TestProgramMakeNativeTypeDeclaration(t *testing.T) {
	prog := newTestProgram()
	decl := prog.MakeNativeTypeDeclaration("MyType", 0)

	if decl == nil {
		t.Fatal("MakeNativeTypeDeclaration returned nil")
	}
	if decl.Name.Text != "MyType" {
		t.Error("wrong name")
	}
}

// --- FunctionPrototype tests ---

func TestNewFunctionPrototype(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("myFunc", 0)

	fp := NewFunctionPrototype("myFunc", file, decl, DecoratorFlagsNone)
	if fp == nil {
		t.Fatal("NewFunctionPrototype returned nil")
	}
	if fp.GetName() != "myFunc" {
		t.Errorf("name = %q, want %q", fp.GetName(), "myFunc")
	}
	if fp.GetElementKind() != ElementKindFunctionPrototype {
		t.Error("wrong element kind")
	}
}

func TestFunctionPrototypeInstances(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("myFunc", 0)
	fp := NewFunctionPrototype("myFunc", file, decl, DecoratorFlagsNone)

	// Initially nil
	if fp.GetResolvedInstance("default") != nil {
		t.Error("should return nil for unregistered instance")
	}

	// Register a mock function - create a proper Function via NewFunction
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("myFunc", fp, nil, sig, nil)

	fp.SetResolvedInstance("default", fn)

	got := fp.GetResolvedInstance("default")
	if got != fn {
		t.Error("should return registered instance")
	}
}

// --- Function tests ---

func TestNewFunction(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)

	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	if fn == nil {
		t.Fatal("NewFunction returned nil")
	}
	if fn.GetElementKind() != ElementKindFunction {
		t.Error("wrong element kind")
	}
	if fn.Signature != sig {
		t.Error("wrong signature")
	}
	if fn.Flow == nil {
		t.Error("Flow should be initialized")
	}
	if fn.Original != fn {
		t.Error("Original should point to self")
	}
	if fn.Prototype != fp {
		t.Error("wrong prototype")
	}
}

func TestFunctionAddLocal(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	local := fn.AddLocal(types.TypeI32, "x", nil)
	if local == nil {
		t.Fatal("AddLocal returned nil")
	}
	if local.Index != 0 {
		t.Errorf("first local index = %d, want 0", local.Index)
	}
	if local.GetName() != "x" {
		t.Error("wrong local name")
	}
	if len(fn.LocalsByIndex) != 1 {
		t.Errorf("LocalsByIndex length = %d, want 1", len(fn.LocalsByIndex))
	}

	// Add another
	local2 := fn.AddLocal(types.TypeF64, "y", nil)
	if local2.Index != 1 {
		t.Errorf("second local index = %d, want 1", local2.Index)
	}
	if len(fn.LocalsByIndex) != 2 {
		t.Errorf("LocalsByIndex length = %d, want 2", len(fn.LocalsByIndex))
	}
}

func TestFunctionAddLocalDuplicatePanics(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	fn.AddLocal(types.TypeI32, "x", nil)
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic on duplicate local name")
		}
	}()
	fn.AddLocal(types.TypeI32, "x", nil) // duplicate
}

func TestFunctionGetParameterName(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	// No parameters in declaration, should fall back to default names
	name := fn.GetParameterName(0)
	if name != "$0" {
		t.Errorf("GetParameterName(0) = %q, want %q", name, "$0")
	}
}

func TestFunctionFlowBridge(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	// FlowAddLocal
	flowLocal := fn.FlowAddLocal(types.TypeI32)
	if flowLocal == nil {
		t.Fatal("FlowAddLocal returned nil")
	}

	// FlowLocalsByIndex
	locals := fn.FlowLocalsByIndex()
	if len(locals) != 1 {
		t.Errorf("FlowLocalsByIndex length = %d, want 1", len(locals))
	}

	// FlowInternalName
	if fn.FlowInternalName() == "" {
		t.Error("FlowInternalName should not be empty")
	}

	// AllocBreakId
	id := fn.AllocBreakId()
	if id != 0 {
		t.Errorf("first AllocBreakId = %d, want 0", id)
	}
	id2 := fn.AllocBreakId()
	if id2 != 1 {
		t.Errorf("second AllocBreakId = %d, want 1", id2)
	}

	// PushBreakStack / PopBreakStack
	fn.PushBreakStack(10)
	fn.PushBreakStack(20)
	popped := fn.PopBreakStack()
	if popped != 20 {
		t.Errorf("PopBreakStack = %d, want 20", popped)
	}

	// AllocInlineId
	iid := fn.AllocInlineId()
	if iid != 0 {
		t.Errorf("first AllocInlineId = %d, want 0", iid)
	}

	// FlowSignature
	if fn.FlowSignature() != sig {
		t.Error("FlowSignature should return the function's signature")
	}

	// GetFlow
	if fn.GetFlow() != fn.Flow {
		t.Error("GetFlow should return the function's flow")
	}

	// FlowProgram
	if fn.FlowProgram() == nil {
		t.Error("FlowProgram should not be nil")
	}
}

func TestFunctionFlowTruncateLocals(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	fn.AddLocal(types.TypeI32, "", nil)
	fn.AddLocal(types.TypeI32, "", nil)
	fn.AddLocal(types.TypeI32, "", nil)

	if len(fn.LocalsByIndex) != 3 {
		t.Fatalf("expected 3 locals, got %d", len(fn.LocalsByIndex))
	}

	fn.FlowTruncateLocals(1)
	if len(fn.LocalsByIndex) != 1 {
		t.Errorf("after truncate: %d locals, want 1", len(fn.LocalsByIndex))
	}
}

// --- Local tests ---

func TestNewLocal(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	local := NewLocal("myLocal", 0, types.TypeI32, fn, nil)
	if local == nil {
		t.Fatal("NewLocal returned nil")
	}
	if local.GetName() != "myLocal" {
		t.Errorf("name = %q, want %q", local.GetName(), "myLocal")
	}
	if local.OriginalName != "myLocal" {
		t.Errorf("OriginalName = %q, want %q", local.OriginalName, "myLocal")
	}
	if local.Index != 0 {
		t.Errorf("Index = %d, want 0", local.Index)
	}
	if local.GetElementKind() != ElementKindLocal {
		t.Error("wrong element kind")
	}
}

func TestNewLocalVoidPanics(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic when creating local with void type")
		}
	}()
	NewLocal("bad", 0, types.TypeVoid, fn, nil)
}

func TestLocalFlowBridge(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("testFn", common.CommonFlagsAmbient)
	fp := NewFunctionPrototype("testFn", file, decl, DecoratorFlagsNone)
	sig := types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false)
	fn := NewFunction("testFn", fp, nil, sig, nil)

	local := NewLocal("x", 5, types.TypeI32, fn, nil)

	if local.FlowIndex() != 5 {
		t.Errorf("FlowIndex = %d, want 5", local.FlowIndex())
	}
	if local.GetType() != types.TypeI32 {
		t.Error("GetType should return i32")
	}

	local.SetName("renamed")
	if local.GetName() != "renamed" {
		t.Errorf("after SetName: %q", local.GetName())
	}

	if local.DeclarationIsNative() {
		t.Error("DeclarationIsNative should return false")
	}
	// DeclarationRange and DeclarationNameRange should not panic
	_ = local.DeclarationRange()
	_ = local.DeclarationNameRange()
}

// --- Memory alignment tests ---

func TestAlSizeConstants(t *testing.T) {
	if AlSize != 16 {
		t.Errorf("AlSize = %d, want 16", AlSize)
	}
	if AlMask != 15 {
		t.Errorf("AlMask = %d, want 15", AlMask)
	}
}

// --- File tests ---

func TestNewFile(t *testing.T) {
	prog := newTestProgram()
	src := ast.NativeSource()
	file := NewFile(prog, src)

	if file == nil {
		t.Fatal("NewFile returned nil")
	}
	if file.GetElementKind() != ElementKindFile {
		t.Error("wrong element kind")
	}
	if file.Source != src {
		t.Error("wrong source")
	}
}

// --- DeclaredElementBase tests ---

func TestDeclaredElementBaseIdentifierNode(t *testing.T) {
	prog := newTestProgram()
	file := prog.NativeFile
	decl := prog.MakeNativeFunctionDeclaration("myFunc", 0)

	fp := NewFunctionPrototype("myFunc", file, decl, DecoratorFlagsNone)

	ident := fp.IdentifierNode()
	if ident == nil {
		t.Fatal("IdentifierNode should not be nil for function declaration")
	}
	if ident.Text != "myFunc" {
		t.Errorf("ident.Text = %q, want %q", ident.Text, "myFunc")
	}
}

// --- extractArgs tests ---

func TestExtractArgs(t *testing.T) {
	a0, a1, a2 := extractArgs(nil)
	if a0 != "" || a1 != "" || a2 != "" {
		t.Error("nil args should produce empty strings")
	}

	a0, a1, a2 = extractArgs([]string{"x"})
	if a0 != "x" || a1 != "" || a2 != "" {
		t.Error("single arg should pad with empty strings")
	}

	a0, a1, a2 = extractArgs([]string{"a", "b", "c", "d"})
	if a0 != "a" || a1 != "b" || a2 != "c" {
		t.Error("extra args should be ignored")
	}
}

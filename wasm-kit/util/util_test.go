package util

import (
	"math"
	"strings"
	"testing"
)

// ---------- CharCode constants ----------

func TestCharCodeValues(t *testing.T) {
	tests := []struct {
		name string
		got  int32
		want int32
	}{
		{"Null", CharCodeNull, 0},
		{"LineFeed", CharCodeLineFeed, 0x0A},
		{"CarriageReturn", CharCodeCarriageReturn, 0x0D},
		{"Space", CharCodeSpace, 0x20},
		{"Tab", CharCodeTab, 0x09},
		{"Digit0", CharCode0, 0x30},
		{"Digit9", CharCode9, 0x39},
		{"LowerA", CharCodeLowerA, 0x61},
		{"LowerZ", CharCodeLowerZ, 0x7A},
		{"UpperA", CharCodeUpperA, 0x41},
		{"UpperZ", CharCodeUpperZ, 0x5A},
		{"Underscore", CharCodeUnderscore, 0x5F},
		{"Dollar", CharCodeDollar, 0x24},
		{"Slash", CharCodeSlash, 0x2F},
		{"Dot", CharCodeDot, 0x2E},
		{"Backslash", CharCodeBackslash, 0x5C},
		{"ByteOrderMark", CharCodeByteOrderMark, 0xFEFF},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got 0x%X, want 0x%X", tc.got, tc.want)
			}
		})
	}
}

// ---------- Character classification ----------

func TestIsLineBreak(t *testing.T) {
	if !IsLineBreak(CharCodeLineFeed) {
		t.Error("LF should be line break")
	}
	if !IsLineBreak(CharCodeCarriageReturn) {
		t.Error("CR should be line break")
	}
	if !IsLineBreak(CharCodeLineSeparator) {
		t.Error("LineSeparator should be line break")
	}
	if !IsLineBreak(CharCodeParagraphSeparator) {
		t.Error("ParagraphSeparator should be line break")
	}
	if IsLineBreak(CharCodeSpace) {
		t.Error("Space should not be line break")
	}
	if IsLineBreak(CharCodeTab) {
		t.Error("Tab should not be line break")
	}
}

func TestIsWhiteSpace(t *testing.T) {
	whitespace := []int32{
		CharCodeSpace, CharCodeTab, CharCodeVerticalTab, CharCodeFormFeed,
		CharCodeNonBreakingSpace, CharCodeNextLine, CharCodeOgham,
		CharCodeEnQuad, CharCodeEmQuad, CharCodeEnSpace, CharCodeEmSpace,
		CharCodeThreePerEmSpace, CharCodeFourPerEmSpace, CharCodeSixPerEmSpace,
		CharCodeFigureSpace, CharCodePunctuationSpace, CharCodeThinSpace,
		CharCodeHairSpace, CharCodeNarrowNoBreakSpace, CharCodeMathematicalSpace,
		CharCodeIdeographicSpace, CharCodeZeroWidthSpace, CharCodeByteOrderMark,
	}
	for _, c := range whitespace {
		if !IsWhiteSpace(c) {
			t.Errorf("0x%X should be whitespace", c)
		}
	}
	if IsWhiteSpace(CharCodeLowerA) {
		t.Error("'a' should not be whitespace")
	}
	if IsWhiteSpace(CharCodeLineFeed) {
		t.Error("LF should not be whitespace")
	}
}

func TestIsDecimal(t *testing.T) {
	for c := CharCode0; c <= CharCode9; c++ {
		if !IsDecimal(c) {
			t.Errorf("0x%X should be decimal", c)
		}
	}
	if IsDecimal(CharCodeLowerA) {
		t.Error("'a' should not be decimal")
	}
	if IsDecimal(CharCode0 - 1) {
		t.Error("char before '0' should not be decimal")
	}
}

func TestIsAlpha(t *testing.T) {
	for c := CharCodeLowerA; c <= CharCodeLowerZ; c++ {
		if !IsAlpha(c) {
			t.Errorf("0x%X should be alpha", c)
		}
	}
	for c := CharCodeUpperA; c <= CharCodeUpperZ; c++ {
		if !IsAlpha(c) {
			t.Errorf("0x%X should be alpha", c)
		}
	}
	if IsAlpha(CharCode0) {
		t.Error("'0' should not be alpha")
	}
	if IsAlpha(CharCodeSpace) {
		t.Error("space should not be alpha")
	}
}

func TestIsOctal(t *testing.T) {
	for c := CharCode0; c <= CharCode7; c++ {
		if !IsOctal(c) {
			t.Errorf("0x%X should be octal", c)
		}
	}
	if IsOctal(CharCode8) {
		t.Error("'8' should not be octal")
	}
}

func TestIsHexOrDecimal(t *testing.T) {
	hex := "0123456789abcdefABCDEF"
	for _, r := range hex {
		if !IsHexOrDecimal(int32(r)) {
			t.Errorf("'%c' should be hex or decimal", r)
		}
	}
	if IsHexOrDecimal(int32('g')) {
		t.Error("'g' should not be hex")
	}
	if IsHexOrDecimal(int32('G')) {
		t.Error("'G' should not be hex")
	}
}

func TestIsIdentifierStart(t *testing.T) {
	if !IsIdentifierStart(CharCodeLowerA) {
		t.Error("'a' should be identifier start")
	}
	if !IsIdentifierStart(CharCodeUpperZ) {
		t.Error("'Z' should be identifier start")
	}
	if !IsIdentifierStart(CharCodeUnderscore) {
		t.Error("'_' should be identifier start")
	}
	if !IsIdentifierStart(CharCodeDollar) {
		t.Error("'$' should be identifier start")
	}
	if IsIdentifierStart(CharCode0) {
		t.Error("'0' should not be identifier start")
	}
	if IsIdentifierStart(CharCodeMinus) {
		t.Error("'-' should not be identifier start")
	}
	// Unicode identifier start (e.g., 'e' with accent at 0xE9 = 233)
	if !IsIdentifierStart(0xE9) {
		t.Error("U+00E9 should be identifier start")
	}
}

func TestIsIdentifierPart(t *testing.T) {
	if !IsIdentifierPart(CharCodeLowerA) {
		t.Error("'a' should be identifier part")
	}
	if !IsIdentifierPart(CharCode0) {
		t.Error("'0' should be identifier part")
	}
	if !IsIdentifierPart(CharCodeUnderscore) {
		t.Error("'_' should be identifier part")
	}
	if !IsIdentifierPart(CharCodeDollar) {
		t.Error("'$' should be identifier part")
	}
	if IsIdentifierPart(CharCodeSpace) {
		t.Error("space should not be identifier part")
	}
	if IsIdentifierPart(CharCodeMinus) {
		t.Error("'-' should not be identifier part")
	}
}

func TestIsIdentifier(t *testing.T) {
	valid := []string{"foo", "_bar", "$baz", "a1", "_", "$", "camelCase", "PascalCase"}
	for _, s := range valid {
		if !IsIdentifier(s) {
			t.Errorf("%q should be a valid identifier", s)
		}
	}

	invalid := []string{"", "1abc", "-foo", "hello world", "a-b", "a.b"}
	for _, s := range invalid {
		if IsIdentifier(s) {
			t.Errorf("%q should not be a valid identifier", s)
		}
	}
}

// ---------- Surrogate functions ----------

func TestSurrogates(t *testing.T) {
	if !IsSurrogate(0xD800) {
		t.Error("0xD800 should be surrogate")
	}
	if !IsSurrogate(0xDFFF) {
		t.Error("0xDFFF should be surrogate")
	}
	if IsSurrogate(0xD7FF) {
		t.Error("0xD7FF should not be surrogate")
	}
	if IsSurrogate(0xE000) {
		t.Error("0xE000 should not be surrogate")
	}

	if !IsHighSurrogate(0xD800) {
		t.Error("0xD800 should be high surrogate")
	}
	if IsHighSurrogate(0xDC00) {
		t.Error("0xDC00 should not be high surrogate")
	}

	if !IsLowSurrogate(0xDC00) {
		t.Error("0xDC00 should be low surrogate")
	}
	if IsLowSurrogate(0xD800) {
		t.Error("0xD800 should not be low surrogate")
	}

	// U+10000 = surrogate pair (0xD800, 0xDC00)
	cp := CombineSurrogates(0xD800, 0xDC00)
	if cp != 0x10000 {
		t.Errorf("CombineSurrogates(0xD800, 0xDC00) = 0x%X, want 0x10000", cp)
	}

	// U+1F600 = surrogate pair (0xD83D, 0xDE00)
	cp = CombineSurrogates(0xD83D, 0xDE00)
	if cp != 0x1F600 {
		t.Errorf("CombineSurrogates(0xD83D, 0xDE00) = 0x%X, want 0x1F600", cp)
	}
}

func TestNumCodeUnits(t *testing.T) {
	if NumCodeUnits(0x41) != 1 {
		t.Error("BMP character should be 1 code unit")
	}
	if NumCodeUnits(0xFFFF) != 1 {
		t.Error("U+FFFF should be 1 code unit")
	}
	if NumCodeUnits(0x10000) != 2 {
		t.Error("U+10000 should be 2 code units")
	}
	if NumCodeUnits(0x1F600) != 2 {
		t.Error("U+1F600 should be 2 code units")
	}
}

// ---------- EscapeString ----------

func TestEscapeString(t *testing.T) {
	// Basic escape sequences
	if got := EscapeString("\x00", CharCodeDoubleQuote); got != "\\0" {
		t.Errorf("null: got %q, want %q", got, "\\0")
	}
	if got := EscapeString("\b", CharCodeDoubleQuote); got != "\\b" {
		t.Errorf("backspace: got %q, want %q", got, "\\b")
	}
	if got := EscapeString("\t", CharCodeDoubleQuote); got != "\\t" {
		t.Errorf("tab: got %q, want %q", got, "\\t")
	}
	if got := EscapeString("\n", CharCodeDoubleQuote); got != "\\n" {
		t.Errorf("newline: got %q, want %q", got, "\\n")
	}
	if got := EscapeString("\v", CharCodeDoubleQuote); got != "\\v" {
		t.Errorf("vtab: got %q, want %q", got, "\\v")
	}
	if got := EscapeString("\f", CharCodeDoubleQuote); got != "\\f" {
		t.Errorf("formfeed: got %q, want %q", got, "\\f")
	}
	if got := EscapeString("\r", CharCodeDoubleQuote); got != "\\r" {
		t.Errorf("cr: got %q, want %q", got, "\\r")
	}
	if got := EscapeString("\\", CharCodeDoubleQuote); got != "\\\\" {
		t.Errorf("backslash: got %q, want %q", got, "\\\\")
	}

	// Quote-sensitive escaping
	if got := EscapeString(`"hello"`, CharCodeDoubleQuote); got != `\"hello\"` {
		t.Errorf("double quote: got %q, want %q", got, `\"hello\"`)
	}
	if got := EscapeString(`"hello"`, CharCodeSingleQuote); got != `"hello"` {
		t.Errorf("double quote with single: got %q, want %q", got, `"hello"`)
	}
	if got := EscapeString("'hello'", CharCodeSingleQuote); got != "\\'hello\\'" {
		t.Errorf("single quote: got %q, want %q", got, "\\'hello\\'")
	}
	if got := EscapeString("'hello'", CharCodeDoubleQuote); got != "'hello'" {
		t.Errorf("single quote with double: got %q, want %q", got, "'hello'")
	}
	if got := EscapeString("`hello`", CharCodeBacktick); got != "\\`hello\\`" {
		t.Errorf("backtick: got %q, want %q", got, "\\`hello\\`")
	}
	if got := EscapeString("`hello`", CharCodeDoubleQuote); got != "`hello`" {
		t.Errorf("backtick with double: got %q, want %q", got, "`hello`")
	}

	// Regular text passes through
	if got := EscapeString("hello world", CharCodeDoubleQuote); got != "hello world" {
		t.Errorf("regular text: got %q, want %q", got, "hello world")
	}
}

// ---------- Indent ----------

func TestIndent(t *testing.T) {
	tests := []struct {
		level int
		want  string
	}{
		{0, ""},
		{1, "  "},
		{2, "    "},
		{3, "      "},
		{4, "        "},
		{5, "          "},
	}
	for _, tc := range tests {
		var sb strings.Builder
		Indent(&sb, tc.level)
		if got := sb.String(); got != tc.want {
			t.Errorf("Indent(%d) = %q, want %q", tc.level, got, tc.want)
		}
	}
}

// ---------- NormalizePath ----------

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"./foo", "foo"},
		{"a/./b", "a/b"},
		{"a/b/../c", "a/c"},
		{"a/b/c/../../d", "a/d"},
		{"../a", "../a"},
		{"", "."},
		{"a/b/./c", "a/b/c"},
		{"foo", "foo"},
		{"./", "."},
		{"././foo", "foo"},
		{"a/b/c", "a/b/c"},
		{"a/b/c/..", "a/b"},
		{"a/b/c/.", "a/b/c"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizePath(tc.input)
			if got != tc.want {
				t.Errorf("NormalizePath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------- ResolvePath ----------

func TestResolvePath(t *testing.T) {
	// std/ prefix short-circuit
	if got := ResolvePath("std/builtins", "some/origin"); got != "std/builtins" {
		t.Errorf("ResolvePath std/ prefix: got %q, want %q", got, "std/builtins")
	}

	// Normal resolution
	got := ResolvePath("bar", "some/dir/file")
	if got != "some/dir/bar" {
		t.Errorf("ResolvePath(bar, some/dir/file) = %q, want %q", got, "some/dir/bar")
	}
}

// ---------- Dirname ----------

func TestDirname(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a/b/c", "a/b"},
		{"abc", "."},
		{"/", "/"},
		{"", "."},
		{"a/b", "a"},
		{"foo/bar/baz", "foo/bar"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := Dirname(tc.input)
			if got != tc.want {
				t.Errorf("Dirname(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------- BitSet ----------

func TestBitSetBasic(t *testing.T) {
	bs := NewBitSet()

	if bs.Size() != 0 {
		t.Error("new BitSet should have size 0")
	}

	bs.Add(0)
	bs.Add(1)
	bs.Add(5)
	bs.Add(31)
	bs.Add(32)
	bs.Add(100)

	if bs.Size() != 6 {
		t.Errorf("Size() = %d, want 6", bs.Size())
	}

	if !bs.Has(0) {
		t.Error("should have 0")
	}
	if !bs.Has(1) {
		t.Error("should have 1")
	}
	if !bs.Has(5) {
		t.Error("should have 5")
	}
	if !bs.Has(31) {
		t.Error("should have 31")
	}
	if !bs.Has(32) {
		t.Error("should have 32")
	}
	if !bs.Has(100) {
		t.Error("should have 100")
	}
	if bs.Has(2) {
		t.Error("should not have 2")
	}
	if bs.Has(99) {
		t.Error("should not have 99")
	}
}

func TestBitSetDelete(t *testing.T) {
	bs := NewBitSet()
	bs.Add(3)
	bs.Add(7)

	if bs.Size() != 2 {
		t.Errorf("Size() = %d, want 2", bs.Size())
	}

	bs.Delete(3)
	if bs.Has(3) {
		t.Error("should not have 3 after delete")
	}
	if bs.Size() != 1 {
		t.Errorf("Size() = %d, want 1", bs.Size())
	}
	if !bs.Has(7) {
		t.Error("should still have 7")
	}

	// Delete non-existent index should be safe
	bs.Delete(999)
}

func TestBitSetToArray(t *testing.T) {
	bs := NewBitSet()
	bs.Add(5)
	bs.Add(1)
	bs.Add(10)
	bs.Add(3)

	arr := bs.ToArray()
	expected := []int{1, 3, 5, 10}

	if len(arr) != len(expected) {
		t.Fatalf("ToArray() length = %d, want %d", len(arr), len(expected))
	}
	for i, v := range expected {
		if arr[i] != v {
			t.Errorf("ToArray()[%d] = %d, want %d", i, arr[i], v)
		}
	}
}

func TestBitSetClear(t *testing.T) {
	bs := NewBitSet()
	bs.Add(1)
	bs.Add(50)

	bs.Clear()

	if bs.Size() != 0 {
		t.Errorf("Size() after Clear = %d, want 0", bs.Size())
	}
	if bs.Has(1) {
		t.Error("should not have 1 after clear")
	}
	if bs.Has(50) {
		t.Error("should not have 50 after clear")
	}
}

func TestBitSetString(t *testing.T) {
	bs := NewBitSet()
	bs.Add(2)
	bs.Add(4)
	s := bs.String()
	if s != "BitSet{2, 4}" {
		t.Errorf("String() = %q, want %q", s, "BitSet{2, 4}")
	}
}

func TestBitSetChaining(t *testing.T) {
	bs := NewBitSet()
	bs.Add(1).Add(2).Add(3)
	if bs.Size() != 3 {
		t.Errorf("chained Add: Size() = %d, want 3", bs.Size())
	}
}

// ---------- CloneMap / MergeMaps ----------

func TestCloneMap(t *testing.T) {
	// Nil map
	var nilMap map[string]int
	cloned := CloneMap(nilMap)
	if cloned != nil {
		t.Error("CloneMap(nil) should return nil")
	}

	// Non-nil map
	m := map[string]int{"a": 1, "b": 2}
	c := CloneMap(m)
	if len(c) != 2 || c["a"] != 1 || c["b"] != 2 {
		t.Error("CloneMap did not copy correctly")
	}
	// Mutation should not affect original
	c["a"] = 99
	if m["a"] != 1 {
		t.Error("CloneMap should create independent copy")
	}
}

func TestMergeMaps(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 3, "c": 4}
	merged := MergeMaps(m1, m2)
	if len(merged) != 3 {
		t.Fatalf("MergeMaps length = %d, want 3", len(merged))
	}
	if merged["a"] != 1 {
		t.Error("merged[a] should be 1")
	}
	if merged["b"] != 3 {
		t.Error("merged[b] should be 3 (m2 wins)")
	}
	if merged["c"] != 4 {
		t.Error("merged[c] should be 4")
	}
}

// ---------- Binary read/write round-trips ----------

func TestBinaryI8(t *testing.T) {
	buf := make([]byte, 4)
	WriteI8(42, buf, 0)
	if got := ReadI8(buf, 0); got != 42 {
		t.Errorf("ReadI8 = %d, want 42", got)
	}
	WriteI8(-1, buf, 1)
	if got := ReadI8(buf, 1); got != 255 { // unsigned read
		t.Errorf("ReadI8(-1) = %d, want 255", got)
	}
}

func TestBinaryI16(t *testing.T) {
	buf := make([]byte, 4)
	WriteI16(0x1234, buf, 0)
	if got := ReadI16(buf, 0); got != 0x1234 {
		t.Errorf("ReadI16 = 0x%X, want 0x1234", got)
	}
}

func TestBinaryI32(t *testing.T) {
	buf := make([]byte, 4)
	WriteI32(0x12345678, buf, 0)
	if got := ReadI32(buf, 0); got != 0x12345678 {
		t.Errorf("ReadI32 = 0x%X, want 0x12345678", got)
	}
	WriteI32(-1, buf, 0)
	if got := ReadI32(buf, 0); got != -1 {
		t.Errorf("ReadI32(-1) = %d, want -1", got)
	}
}

func TestBinaryI64(t *testing.T) {
	buf := make([]byte, 8)
	WriteI64(0x123456789ABCDEF0, buf, 0)
	if got := ReadI64(buf, 0); got != 0x123456789ABCDEF0 {
		t.Errorf("ReadI64 = 0x%X, want 0x123456789ABCDEF0", got)
	}
}

func TestBinaryI32AsI64(t *testing.T) {
	buf := make([]byte, 8)

	// Positive signed
	WriteI32AsI64(42, buf, 0, false)
	if got := ReadI64(buf, 0); got != 42 {
		t.Errorf("WriteI32AsI64(42, signed) read as i64 = %d, want 42", got)
	}

	// Negative signed
	WriteI32AsI64(-1, buf, 0, false)
	if got := ReadI64(buf, 0); got != -1 {
		t.Errorf("WriteI32AsI64(-1, signed) read as i64 = %d, want -1", got)
	}

	// Unsigned
	WriteI32AsI64(42, buf, 0, true)
	if got := ReadI64(buf, 0); got != 42 {
		t.Errorf("WriteI32AsI64(42, unsigned) read as i64 = %d, want 42", got)
	}
}

func TestBinaryI64AsI32(t *testing.T) {
	buf := make([]byte, 4)
	WriteI64AsI32(42, buf, 0, false)
	if got := ReadI32(buf, 0); got != 42 {
		t.Errorf("WriteI64AsI32(42) = %d, want 42", got)
	}

	// Should panic on overflow (signed)
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("WriteI64AsI32 should panic on signed overflow")
			}
		}()
		WriteI64AsI32(0x100000000, buf, 0, false)
	}()

	// Should panic on overflow (unsigned)
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("WriteI64AsI32 should panic on unsigned overflow")
			}
		}()
		WriteI64AsI32(0x100000000, buf, 0, true)
	}()

	// Should panic on negative unsigned
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("WriteI64AsI32 should panic on negative unsigned")
			}
		}()
		WriteI64AsI32(-1, buf, 0, true)
	}()
}

func TestBinaryF32(t *testing.T) {
	buf := make([]byte, 4)
	WriteF32(3.14, buf, 0)
	got := ReadF32(buf, 0)
	if got != float32(3.14) {
		t.Errorf("ReadF32 = %f, want %f", got, float32(3.14))
	}

	// Special values
	WriteF32(float32(math.Inf(1)), buf, 0)
	if got := ReadF32(buf, 0); !math.IsInf(float64(got), 1) {
		t.Error("should handle +Inf")
	}
}

func TestBinaryF64(t *testing.T) {
	buf := make([]byte, 8)
	WriteF64(math.Pi, buf, 0)
	got := ReadF64(buf, 0)
	if got != math.Pi {
		t.Errorf("ReadF64 = %f, want %f", got, math.Pi)
	}

	WriteF64(math.NaN(), buf, 0)
	got = ReadF64(buf, 0)
	if !math.IsNaN(got) {
		t.Error("should handle NaN")
	}
}

func TestBinaryV128(t *testing.T) {
	buf := make([]byte, 32)
	val := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	WriteV128(val, buf, 8)
	got := ReadV128(buf, 8)
	if got != val {
		t.Errorf("ReadV128 = %v, want %v", got, val)
	}
}

// ---------- Terminal ----------

func TestTerminalColors(t *testing.T) {
	// Enable colors
	SetColorsEnabled(true)
	if !IsColorsEnabled() {
		t.Error("should be enabled")
	}

	colored := Colorize("hello", ColorRed)
	if colored != "\033[91mhello\033[0m" {
		t.Errorf("Colorize = %q, want ANSI-wrapped", colored)
	}

	// Disable colors
	prev := SetColorsEnabled(false)
	if !prev {
		t.Error("SetColorsEnabled should return previous state")
	}
	if IsColorsEnabled() {
		t.Error("should be disabled")
	}

	plain := Colorize("hello", ColorRed)
	if plain != "hello" {
		t.Errorf("Colorize with disabled = %q, want plain", plain)
	}

	// Restore
	SetColorsEnabled(true)
}

// ---------- Math ----------

func TestIsPowerOf2(t *testing.T) {
	powers := []int32{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	for _, p := range powers {
		if !IsPowerOf2(p) {
			t.Errorf("%d should be power of 2", p)
		}
	}

	nonPowers := []int32{0, 3, 5, 6, 7, 9, 10, 15, 100}
	for _, np := range nonPowers {
		if IsPowerOf2(np) {
			t.Errorf("%d should not be power of 2", np)
		}
	}
}

func TestAccuratePow64(t *testing.T) {
	if got := AccuratePow64(2, 10); got != 1024 {
		t.Errorf("AccuratePow64(2,10) = %f, want 1024", got)
	}
}

// ---------- Vector ----------

func TestV128Constants(t *testing.T) {
	for i := 0; i < 16; i++ {
		if V128Zero[i] != 0 {
			t.Errorf("V128Zero[%d] = %d, want 0", i, V128Zero[i])
		}
		if V128Ones[i] != 0xFF {
			t.Errorf("V128Ones[%d] = %d, want 0xFF", i, V128Ones[i])
		}
	}
}

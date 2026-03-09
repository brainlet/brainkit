package diagnostics

import (
	"strings"
	"testing"
)

// mockSource implements the Source interface for testing.
type mockSource struct {
	text           string
	normalizedPath string
	lineStarts     []int32
	lastColumn     int32
}

func newMockSource(path, text string) *mockSource {
	s := &mockSource{
		text:           text,
		normalizedPath: path,
	}
	// Build line starts cache
	s.lineStarts = []int32{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			s.lineStarts = append(s.lineStarts, int32(i+1))
		}
	}
	s.lineStarts = append(s.lineStarts, 0x7fffffff)
	return s
}

func (s *mockSource) SourceText() string           { return s.text }
func (s *mockSource) SourceNormalizedPath() string  { return s.normalizedPath }

func (s *mockSource) LineAt(pos int32) int32 {
	l := 0
	r := len(s.lineStarts) - 1
	for l < r {
		m := l + ((r - l) >> 1)
		if pos < s.lineStarts[m] {
			r = m
		} else if pos < s.lineStarts[m+1] {
			s.lastColumn = pos - s.lineStarts[m] + 1
			return int32(m + 1)
		} else {
			l = m + 1
		}
	}
	panic("unreachable")
}

func (s *mockSource) ColumnAt() int32 {
	return s.lastColumn
}

func TestDiagnosticCategory(t *testing.T) {
	tests := []struct {
		cat  DiagnosticCategory
		str  string
	}{
		{DiagnosticCategoryPedantic, "PEDANTIC"},
		{DiagnosticCategoryInfo, "INFO"},
		{DiagnosticCategoryWarning, "WARNING"},
		{DiagnosticCategoryError, "ERROR"},
	}
	for _, tt := range tests {
		got := DiagnosticCategoryToString(tt.cat)
		if got != tt.str {
			t.Errorf("DiagnosticCategoryToString(%d) = %q, want %q", tt.cat, got, tt.str)
		}
	}
}

func TestDiagnosticCodeToString(t *testing.T) {
	tests := []struct {
		code DiagnosticCode
		want string
	}{
		{DiagnosticCodeNotImplemented0, "Not implemented: {0}"},
		{DiagnosticCodeOperationIsUnsafe, "Operation is unsafe."},
		{DiagnosticCodeIdentifierExpected, "Identifier expected."},
		{DiagnosticCodeUnexpectedToken, "Unexpected token."},
		{DiagnosticCodeDuplicateIdentifier0, "Duplicate identifier '{0}'."},
		{DiagnosticCode(99999), ""}, // unknown code
	}
	for _, tt := range tests {
		got := DiagnosticCodeToString(tt.code)
		if got != tt.want {
			t.Errorf("DiagnosticCodeToString(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestRange(t *testing.T) {
	src := newMockSource("test.ts", "hello world\nfoo bar\n")

	t.Run("NewRange", func(t *testing.T) {
		r := NewRange(0, 5)
		r.Source = src
		if r.Start != 0 || r.End != 5 {
			t.Errorf("NewRange(0,5) = {%d,%d}, want {0,5}", r.Start, r.End)
		}
		if r.String() != "hello" {
			t.Errorf("Range.String() = %q, want %q", r.String(), "hello")
		}
	})

	t.Run("AtStart", func(t *testing.T) {
		r := NewRange(3, 8)
		r.Source = src
		at := r.AtStart()
		if at.Start != 3 || at.End != 3 {
			t.Errorf("AtStart = {%d,%d}, want {3,3}", at.Start, at.End)
		}
		if at.Source != src {
			t.Error("AtStart.Source should equal original source")
		}
	})

	t.Run("AtEnd", func(t *testing.T) {
		r := NewRange(3, 8)
		r.Source = src
		at := r.AtEnd()
		if at.Start != 8 || at.End != 8 {
			t.Errorf("AtEnd = {%d,%d}, want {8,8}", at.Start, at.End)
		}
	})

	t.Run("Equals", func(t *testing.T) {
		r1 := NewRange(0, 5)
		r1.Source = src
		r2 := NewRange(0, 5)
		r2.Source = src
		if !r1.Equals(r2) {
			t.Error("Equal ranges should be equal")
		}
		r3 := NewRange(0, 6)
		r3.Source = src
		if r1.Equals(r3) {
			t.Error("Different ranges should not be equal")
		}
	})

	t.Run("JoinRanges", func(t *testing.T) {
		r1 := NewRange(2, 5)
		r1.Source = src
		r2 := NewRange(3, 8)
		r2.Source = src
		joined := JoinRanges(r1, r2)
		if joined.Start != 2 || joined.End != 8 {
			t.Errorf("JoinRanges = {%d,%d}, want {2,8}", joined.Start, joined.End)
		}
	})

	t.Run("JoinRanges_sourceMismatch", func(t *testing.T) {
		src2 := newMockSource("other.ts", "other text")
		r1 := NewRange(0, 5)
		r1.Source = src
		r2 := NewRange(0, 5)
		r2.Source = src2
		defer func() {
			if r := recover(); r == nil {
				t.Error("JoinRanges with different sources should panic")
			}
		}()
		JoinRanges(r1, r2)
	})
}

func TestDiagnosticMessage(t *testing.T) {
	t.Run("Create_noArgs", func(t *testing.T) {
		msg := NewDiagnosticMessage(DiagnosticCodeOperationIsUnsafe, DiagnosticCategoryWarning, "", "", "")
		if msg.Code != 101 {
			t.Errorf("Code = %d, want 101", msg.Code)
		}
		if msg.Category != DiagnosticCategoryWarning {
			t.Errorf("Category = %d, want Warning", msg.Category)
		}
		if msg.Message != "Operation is unsafe." {
			t.Errorf("Message = %q, want %q", msg.Message, "Operation is unsafe.")
		}
	})

	t.Run("Create_withArgs", func(t *testing.T) {
		msg := NewDiagnosticMessage(DiagnosticCodeNotImplemented0, DiagnosticCategoryError, "feature X", "", "")
		if msg.Message != "Not implemented: feature X" {
			t.Errorf("Message = %q, want %q", msg.Message, "Not implemented: feature X")
		}
	})

	t.Run("Create_multiArgs", func(t *testing.T) {
		msg := NewDiagnosticMessage(
			DiagnosticCodeConversionFromType0To1RequiresAnExplicitCast,
			DiagnosticCategoryError,
			"i32", "f64", "",
		)
		want := "Conversion from type 'i32' to 'f64' requires an explicit cast."
		if msg.Message != want {
			t.Errorf("Message = %q, want %q", msg.Message, want)
		}
	})

	t.Run("Equals", func(t *testing.T) {
		m1 := NewDiagnosticMessage(DiagnosticCodeOperationIsUnsafe, DiagnosticCategoryWarning, "", "", "")
		m2 := NewDiagnosticMessage(DiagnosticCodeOperationIsUnsafe, DiagnosticCategoryWarning, "", "", "")
		if !m1.Equals(m2) {
			t.Error("Equal messages should be equal")
		}
	})

	t.Run("Equals_differentCode", func(t *testing.T) {
		m1 := NewDiagnosticMessage(DiagnosticCodeOperationIsUnsafe, DiagnosticCategoryWarning, "", "", "")
		m2 := NewDiagnosticMessage(DiagnosticCodeNotImplemented0, DiagnosticCategoryError, "x", "", "")
		if m1.Equals(m2) {
			t.Error("Different messages should not be equal")
		}
	})

	t.Run("WithRange", func(t *testing.T) {
		src := newMockSource("test.ts", "hello")
		rng := NewRange(0, 5)
		rng.Source = src
		msg := NewDiagnosticMessage(DiagnosticCodeOperationIsUnsafe, DiagnosticCategoryWarning, "", "", "")
		msg.WithRange(rng)
		if msg.Range == nil {
			t.Error("Range should not be nil after WithRange")
		}
	})

	t.Run("String_noRange", func(t *testing.T) {
		msg := NewDiagnosticMessage(DiagnosticCodeOperationIsUnsafe, DiagnosticCategoryWarning, "", "", "")
		s := msg.String()
		if s != "WARNING 101: Operation is unsafe." {
			t.Errorf("String() = %q, want %q", s, "WARNING 101: Operation is unsafe.")
		}
	})

	t.Run("String_withRange", func(t *testing.T) {
		src := newMockSource("test.ts", "hello world\nfoo bar\n")
		rng := NewRange(12, 15)
		rng.Source = src
		msg := NewDiagnosticMessage(DiagnosticCodeOperationIsUnsafe, DiagnosticCategoryWarning, "", "", "")
		msg.WithRange(rng)
		s := msg.String()
		// "foo" is at line 2, column 1, length 3
		if !strings.Contains(s, "test.ts") {
			t.Errorf("String should contain file path, got %q", s)
		}
		if !strings.Contains(s, "(2,") {
			t.Errorf("String should contain line number 2, got %q", s)
		}
	})
}

func TestDiagnosticEmitter(t *testing.T) {
	t.Run("EmitBasic", func(t *testing.T) {
		e := NewDiagnosticEmitter(nil)
		e.Error(DiagnosticCodeOperationIsUnsafe, nil, "", "", "")
		if len(e.Diagnostics) != 1 {
			t.Fatalf("len(Diagnostics) = %d, want 1", len(e.Diagnostics))
		}
		if e.Diagnostics[0].Category != DiagnosticCategoryError {
			t.Error("Expected error category")
		}
	})

	t.Run("AllCategories", func(t *testing.T) {
		e := NewDiagnosticEmitter(nil)
		e.Pedantic(DiagnosticCodeOperationIsUnsafe, nil, "", "", "")
		e.Info(DiagnosticCodeOperationIsUnsafe, nil, "", "", "")
		e.Warning(DiagnosticCodeOperationIsUnsafe, nil, "", "", "")
		e.Error(DiagnosticCodeOperationIsUnsafe, nil, "", "", "")
		if len(e.Diagnostics) != 4 {
			t.Fatalf("len(Diagnostics) = %d, want 4", len(e.Diagnostics))
		}
		if e.Diagnostics[0].Category != DiagnosticCategoryPedantic {
			t.Error("Expected pedantic")
		}
		if e.Diagnostics[1].Category != DiagnosticCategoryInfo {
			t.Error("Expected info")
		}
		if e.Diagnostics[2].Category != DiagnosticCategoryWarning {
			t.Error("Expected warning")
		}
		if e.Diagnostics[3].Category != DiagnosticCategoryError {
			t.Error("Expected error")
		}
	})

	t.Run("Deduplication", func(t *testing.T) {
		src := newMockSource("test.ts", "hello world")
		e := NewDiagnosticEmitter(nil)

		rng := NewRange(0, 5)
		rng.Source = src

		e.Error(DiagnosticCodeOperationIsUnsafe, rng, "", "", "")
		e.Error(DiagnosticCodeOperationIsUnsafe, rng, "", "", "")
		e.Error(DiagnosticCodeOperationIsUnsafe, rng, "", "", "")

		if len(e.Diagnostics) != 1 {
			t.Errorf("len(Diagnostics) = %d, want 1 (dedup failed)", len(e.Diagnostics))
		}
	})

	t.Run("DifferentRanges_noDedupe", func(t *testing.T) {
		src := newMockSource("test.ts", "hello world")
		e := NewDiagnosticEmitter(nil)

		r1 := NewRange(0, 5)
		r1.Source = src
		r2 := NewRange(6, 11)
		r2.Source = src

		e.Error(DiagnosticCodeOperationIsUnsafe, r1, "", "", "")
		e.Error(DiagnosticCodeOperationIsUnsafe, r2, "", "", "")

		if len(e.Diagnostics) != 2 {
			t.Errorf("len(Diagnostics) = %d, want 2", len(e.Diagnostics))
		}
	})

	t.Run("WithRelatedRange", func(t *testing.T) {
		src := newMockSource("test.ts", "hello world\nfoo bar")
		e := NewDiagnosticEmitter(nil)

		rng := NewRange(0, 5)
		rng.Source = src
		rel := NewRange(12, 15)
		rel.Source = src

		e.ErrorRelated(DiagnosticCodeOperationIsUnsafe, rng, rel, "", "", "")
		if len(e.Diagnostics) != 1 {
			t.Fatalf("len(Diagnostics) = %d, want 1", len(e.Diagnostics))
		}
		if e.Diagnostics[0].RelatedRange == nil {
			t.Error("Expected related range to be set")
		}
	})

	t.Run("SharedDiagnostics", func(t *testing.T) {
		shared := make([]*DiagnosticMessage, 0)
		e1 := NewDiagnosticEmitter(shared)
		e2 := NewDiagnosticEmitter(shared)
		// Both emitters share the same initial slice but will diverge on append
		// This matches TS behavior where diagnostics array is passed by reference
		e1.Error(DiagnosticCodeOperationIsUnsafe, nil, "", "", "")
		if len(e1.Diagnostics) != 1 {
			t.Errorf("e1 len = %d, want 1", len(e1.Diagnostics))
		}
		// e2 still has the original (empty) slice header, which is correct
		// In the real compiler, parser passes its diagnostics to the program
		e2.Error(DiagnosticCodeUnexpectedToken, nil, "", "", "")
		if len(e2.Diagnostics) != 1 {
			t.Errorf("e2 len = %d, want 1", len(e2.Diagnostics))
		}
	})
}

func TestFormatDiagnosticMessage(t *testing.T) {
	t.Run("NoRange_noColors", func(t *testing.T) {
		msg := NewDiagnosticMessage(DiagnosticCodeOperationIsUnsafe, DiagnosticCategoryWarning, "", "", "")
		formatted := FormatDiagnosticMessage(msg, false, false)
		if !strings.Contains(formatted, "WARNING") {
			t.Errorf("Expected WARNING in %q", formatted)
		}
		if !strings.Contains(formatted, " AS101") {
			t.Errorf("Expected AS101 in %q", formatted)
		}
	})

	t.Run("TSCode", func(t *testing.T) {
		msg := NewDiagnosticMessage(DiagnosticCodeIdentifierExpected, DiagnosticCategoryError, "", "", "")
		formatted := FormatDiagnosticMessage(msg, false, false)
		if !strings.Contains(formatted, " TS1003") {
			t.Errorf("Expected TS1003 in %q", formatted)
		}
	})

	t.Run("WithRange_noContext", func(t *testing.T) {
		src := newMockSource("test.ts", "hello world\nfoo bar\n")
		rng := NewRange(12, 15)
		rng.Source = src
		msg := NewDiagnosticMessage(DiagnosticCodeUnexpectedToken, DiagnosticCategoryError, "", "", "")
		msg.WithRange(rng)
		formatted := FormatDiagnosticMessage(msg, false, false)
		if !strings.Contains(formatted, "test.ts") {
			t.Errorf("Expected path in %q", formatted)
		}
		if !strings.Contains(formatted, "(2,") {
			t.Errorf("Expected line number in %q", formatted)
		}
	})

	t.Run("WithRange_withContext", func(t *testing.T) {
		src := newMockSource("test.ts", "let x = 42;\nlet y = ;\n")
		rng := NewRange(20, 21) // the ';' after empty '= '
		rng.Source = src
		msg := NewDiagnosticMessage(DiagnosticCodeUnexpectedToken, DiagnosticCategoryError, "", "", "")
		msg.WithRange(rng)
		formatted := FormatDiagnosticMessage(msg, false, true)
		// Should contain the context line and caret
		if !strings.Contains(formatted, "\u2502") {
			t.Errorf("Expected box-drawing chars in context, got %q", formatted)
		}
		if !strings.Contains(formatted, "\u2514") {
			t.Errorf("Expected corner char in context, got %q", formatted)
		}
	})
}

func TestDiagnosticCodeConstants(t *testing.T) {
	// Verify a sampling of code values match the JSON
	tests := []struct {
		code DiagnosticCode
		val  int32
	}{
		{DiagnosticCodeNotImplemented0, 100},
		{DiagnosticCodeOperationIsUnsafe, 101},
		{DiagnosticCodeConversionFromType0To1RequiresAnExplicitCast, 200},
		{DiagnosticCodeUnterminatedStringLiteral, 1002},
		{DiagnosticCodeIdentifierExpected, 1003},
		{DiagnosticCodeUnexpectedToken, 1012},
		{DiagnosticCodeDuplicateIdentifier0, 2300},
		{DiagnosticCodeCannotFindName0, 2304},
		{DiagnosticCodeType0IsNotAssignableToType1, 2322},
		{DiagnosticCodeFile0NotFound, 6054},
	}
	for _, tt := range tests {
		if int32(tt.code) != tt.val {
			t.Errorf("DiagnosticCode %d: got %d, want %d", tt.val, int32(tt.code), tt.val)
		}
	}
}

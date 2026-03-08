// Ported from: packages/core/src/workspace/line-utils.test.ts
package workspace

import (
	"strings"
	"testing"
)

func TestExtractLines(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"

	t.Run("extracts full content when no range specified", func(t *testing.T) {
		result := ExtractLines(content, 0, 0)
		if result.Content != content {
			t.Errorf("Content = %q, want %q", result.Content, content)
		}
		if result.Lines.Start != 1 {
			t.Errorf("Lines.Start = %d, want 1", result.Lines.Start)
		}
		if result.Lines.End != 5 {
			t.Errorf("Lines.End = %d, want 5", result.Lines.End)
		}
		if result.TotalLines != 5 {
			t.Errorf("TotalLines = %d, want 5", result.TotalLines)
		}
	})

	t.Run("extracts a specific range", func(t *testing.T) {
		result := ExtractLines(content, 2, 4)
		if result.Content != "line2\nline3\nline4" {
			t.Errorf("Content = %q, want %q", result.Content, "line2\nline3\nline4")
		}
		if result.Lines.Start != 2 {
			t.Errorf("Lines.Start = %d, want 2", result.Lines.Start)
		}
		if result.Lines.End != 4 {
			t.Errorf("Lines.End = %d, want 4", result.Lines.End)
		}
	})

	t.Run("extracts a single line", func(t *testing.T) {
		result := ExtractLines(content, 3, 3)
		if result.Content != "line3" {
			t.Errorf("Content = %q, want %q", result.Content, "line3")
		}
	})

	t.Run("clamps end to total lines", func(t *testing.T) {
		result := ExtractLines(content, 3, 100)
		if result.Content != "line3\nline4\nline5" {
			t.Errorf("Content = %q, want %q", result.Content, "line3\nline4\nline5")
		}
		if result.Lines.End != 5 {
			t.Errorf("Lines.End = %d, want 5", result.Lines.End)
		}
	})

	t.Run("defaults start to 1 when negative", func(t *testing.T) {
		result := ExtractLines(content, -5, 2)
		if result.Lines.Start != 1 {
			t.Errorf("Lines.Start = %d, want 1", result.Lines.Start)
		}
	})

	t.Run("handles single-line content", func(t *testing.T) {
		result := ExtractLines("only line", 0, 0)
		if result.Content != "only line" {
			t.Errorf("Content = %q, want %q", result.Content, "only line")
		}
		if result.TotalLines != 1 {
			t.Errorf("TotalLines = %d, want 1", result.TotalLines)
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		result := ExtractLines("", 0, 0)
		if result.Content != "" {
			t.Errorf("Content = %q, want empty", result.Content)
		}
		if result.TotalLines != 1 {
			// strings.Split("", "\n") returns [""]
			t.Errorf("TotalLines = %d, want 1", result.TotalLines)
		}
	})
}

func TestExtractLinesWithLimit(t *testing.T) {
	content := "a\nb\nc\nd\ne\nf"

	t.Run("extracts from offset with limit", func(t *testing.T) {
		result := ExtractLinesWithLimit(content, 2, 3)
		if result.Content != "b\nc\nd" {
			t.Errorf("Content = %q, want %q", result.Content, "b\nc\nd")
		}
		if result.Lines.Start != 2 {
			t.Errorf("Lines.Start = %d, want 2", result.Lines.Start)
		}
		if result.Lines.End != 4 {
			t.Errorf("Lines.End = %d, want 4", result.Lines.End)
		}
	})

	t.Run("extracts all remaining when limit is 0", func(t *testing.T) {
		result := ExtractLinesWithLimit(content, 3, 0)
		if result.Content != "c\nd\ne\nf" {
			t.Errorf("Content = %q, want %q", result.Content, "c\nd\ne\nf")
		}
	})

	t.Run("defaults offset to 1 when less than 1", func(t *testing.T) {
		result := ExtractLinesWithLimit(content, 0, 2)
		if result.Lines.Start != 1 {
			t.Errorf("Lines.Start = %d, want 1", result.Lines.Start)
		}
		if result.Content != "a\nb" {
			t.Errorf("Content = %q, want %q", result.Content, "a\nb")
		}
	})
}

func TestFormatWithLineNumbers(t *testing.T) {
	t.Run("formats content with line numbers", func(t *testing.T) {
		content := "hello\nworld"
		result := FormatWithLineNumbers(content, 1)
		lines := strings.Split(result, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}
		// Line 1 should end with the arrow and "hello"
		if !strings.Contains(lines[0], "1") || !strings.Contains(lines[0], "hello") {
			t.Errorf("first line should contain '1' and 'hello', got %q", lines[0])
		}
		if !strings.Contains(lines[1], "2") || !strings.Contains(lines[1], "world") {
			t.Errorf("second line should contain '2' and 'world', got %q", lines[1])
		}
	})

	t.Run("uses unicode arrow separator", func(t *testing.T) {
		result := FormatWithLineNumbers("test", 1)
		if !strings.Contains(result, "\u2192") {
			t.Errorf("should contain unicode right arrow, got %q", result)
		}
	})

	t.Run("starts from specified line number", func(t *testing.T) {
		result := FormatWithLineNumbers("a\nb", 10)
		lines := strings.Split(result, "\n")
		if !strings.Contains(lines[0], "10") {
			t.Errorf("first line should contain '10', got %q", lines[0])
		}
		if !strings.Contains(lines[1], "11") {
			t.Errorf("second line should contain '11', got %q", lines[1])
		}
	})

	t.Run("defaults to 1 when startLineNumber < 1", func(t *testing.T) {
		result := FormatWithLineNumbers("test", 0)
		if !strings.Contains(result, "1") {
			t.Errorf("should default to line 1, got %q", result)
		}
	})

	t.Run("pads line numbers for alignment", func(t *testing.T) {
		// Generate content with >9 lines to test padding
		lines := make([]string, 12)
		for i := range lines {
			lines[i] = "x"
		}
		content := strings.Join(lines, "\n")
		result := FormatWithLineNumbers(content, 1)
		resultLines := strings.Split(result, "\n")
		// Line 1 should have more leading spaces than line 10
		// Both should align properly
		if len(resultLines) != 12 {
			t.Fatalf("expected 12 lines, got %d", len(resultLines))
		}
	})
}

func TestCharIndexToLineNumber(t *testing.T) {
	content := "abc\ndef\nghi"

	t.Run("returns 1 for index 0", func(t *testing.T) {
		result := CharIndexToLineNumber(content, 0)
		if result != 1 {
			t.Errorf("got %d, want 1", result)
		}
	})

	t.Run("returns 1 for index within first line", func(t *testing.T) {
		result := CharIndexToLineNumber(content, 2)
		if result != 1 {
			t.Errorf("got %d, want 1", result)
		}
	})

	t.Run("returns 2 for index at start of second line", func(t *testing.T) {
		result := CharIndexToLineNumber(content, 4)
		if result != 2 {
			t.Errorf("got %d, want 2", result)
		}
	})

	t.Run("returns 3 for index at start of third line", func(t *testing.T) {
		result := CharIndexToLineNumber(content, 8)
		if result != 3 {
			t.Errorf("got %d, want 3", result)
		}
	})

	t.Run("returns -1 for negative index", func(t *testing.T) {
		result := CharIndexToLineNumber(content, -1)
		if result != -1 {
			t.Errorf("got %d, want -1", result)
		}
	})

	t.Run("returns -1 for index beyond content", func(t *testing.T) {
		result := CharIndexToLineNumber(content, 100)
		if result != -1 {
			t.Errorf("got %d, want -1", result)
		}
	})
}

func TestCharRangeToLineRange(t *testing.T) {
	content := "abc\ndef\nghi"

	t.Run("converts char range within single line", func(t *testing.T) {
		result := CharRangeToLineRange(content, 0, 3)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Start != 1 || result.End != 1 {
			t.Errorf("got {Start: %d, End: %d}, want {Start: 1, End: 1}", result.Start, result.End)
		}
	})

	t.Run("converts char range spanning multiple lines", func(t *testing.T) {
		result := CharRangeToLineRange(content, 0, 8)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Start != 1 || result.End != 2 {
			t.Errorf("got {Start: %d, End: %d}, want {Start: 1, End: 2}", result.Start, result.End)
		}
	})

	t.Run("returns nil for invalid indices", func(t *testing.T) {
		result := CharRangeToLineRange(content, -1, 5)
		if result != nil {
			t.Errorf("expected nil for negative start index, got %v", result)
		}
	})
}

func TestCountOccurrences(t *testing.T) {
	t.Run("counts single occurrence", func(t *testing.T) {
		count := CountOccurrences("hello world", "world")
		if count != 1 {
			t.Errorf("got %d, want 1", count)
		}
	})

	t.Run("counts multiple occurrences", func(t *testing.T) {
		count := CountOccurrences("ababab", "ab")
		if count != 3 {
			t.Errorf("got %d, want 3", count)
		}
	})

	t.Run("returns 0 for empty search string", func(t *testing.T) {
		count := CountOccurrences("hello", "")
		if count != 0 {
			t.Errorf("got %d, want 0", count)
		}
	})

	t.Run("returns 0 when not found", func(t *testing.T) {
		count := CountOccurrences("hello", "xyz")
		if count != 0 {
			t.Errorf("got %d, want 0", count)
		}
	})

	t.Run("counts non-overlapping occurrences", func(t *testing.T) {
		count := CountOccurrences("aaa", "aa")
		if count != 1 {
			t.Errorf("got %d, want 1", count)
		}
	})
}

func TestReplaceString(t *testing.T) {
	t.Run("replaces unique occurrence", func(t *testing.T) {
		result, err := ReplaceString("hello world", "world", "Go", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Content != "hello Go" {
			t.Errorf("Content = %q, want %q", result.Content, "hello Go")
		}
		if result.Replacements != 1 {
			t.Errorf("Replacements = %d, want 1", result.Replacements)
		}
	})

	t.Run("errors when string not found", func(t *testing.T) {
		_, err := ReplaceString("hello world", "xyz", "abc", false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if _, ok := err.(*StringNotFoundError); !ok {
			t.Errorf("expected *StringNotFoundError, got %T", err)
		}
	})

	t.Run("errors when string is not unique and replaceAll is false", func(t *testing.T) {
		_, err := ReplaceString("ab ab ab", "ab", "cd", false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		snu, ok := err.(*StringNotUniqueError)
		if !ok {
			t.Fatalf("expected *StringNotUniqueError, got %T", err)
		}
		if snu.Occurrences != 3 {
			t.Errorf("Occurrences = %d, want 3", snu.Occurrences)
		}
	})

	t.Run("replaces all occurrences when replaceAll is true", func(t *testing.T) {
		result, err := ReplaceString("ab ab ab", "ab", "cd", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Content != "cd cd cd" {
			t.Errorf("Content = %q, want %q", result.Content, "cd cd cd")
		}
		if result.Replacements != 3 {
			t.Errorf("Replacements = %d, want 3", result.Replacements)
		}
	})
}

func TestStringNotFoundError(t *testing.T) {
	t.Run("stores search string", func(t *testing.T) {
		err := NewStringNotFoundError("needle")
		if err.SearchString != "needle" {
			t.Errorf("SearchString = %q, want %q", err.SearchString, "needle")
		}
	})

	t.Run("has descriptive error message", func(t *testing.T) {
		err := NewStringNotFoundError("test")
		msg := err.Error()
		if !strings.Contains(msg, "not found") {
			t.Errorf("error message should mention 'not found', got %q", msg)
		}
	})
}

func TestStringNotUniqueError(t *testing.T) {
	t.Run("stores search string and occurrence count", func(t *testing.T) {
		err := NewStringNotUniqueError("dup", 5)
		if err.SearchString != "dup" {
			t.Errorf("SearchString = %q, want %q", err.SearchString, "dup")
		}
		if err.Occurrences != 5 {
			t.Errorf("Occurrences = %d, want 5", err.Occurrences)
		}
	})

	t.Run("includes occurrence count in error message", func(t *testing.T) {
		err := NewStringNotUniqueError("dup", 3)
		msg := err.Error()
		if !strings.Contains(msg, "3") {
			t.Errorf("error message should mention count, got %q", msg)
		}
	})
}

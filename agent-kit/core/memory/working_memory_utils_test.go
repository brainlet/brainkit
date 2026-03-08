// Ported from: packages/core/src/memory/working-memory-utils.test.ts
package memory

import (
	"strings"
	"testing"
	"time"
)

func TestExtractWorkingMemoryTags(t *testing.T) {
	t.Run("should extract simple working memory tags", func(t *testing.T) {
		result := ExtractWorkingMemoryTags("<working_memory>hello world</working_memory>")
		assertSliceEqual(t, result, []string{"<working_memory>hello world</working_memory>"})
	})

	t.Run("should extract multiple working memory tags", func(t *testing.T) {
		result := ExtractWorkingMemoryTags(
			"<working_memory>first</working_memory> text <working_memory>second</working_memory>",
		)
		assertSliceEqual(t, result, []string{
			"<working_memory>first</working_memory>",
			"<working_memory>second</working_memory>",
		})
	})

	t.Run("should handle multiline content", func(t *testing.T) {
		result := ExtractWorkingMemoryTags("<working_memory>line1\nline2\nline3</working_memory>")
		assertSliceEqual(t, result, []string{"<working_memory>line1\nline2\nline3</working_memory>"})
	})

	t.Run("should return nil when no tags found", func(t *testing.T) {
		result := ExtractWorkingMemoryTags("no tags here")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("should return nil when only opening tag exists", func(t *testing.T) {
		result := ExtractWorkingMemoryTags("<working_memory>unclosed")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("should handle nested angle brackets", func(t *testing.T) {
		result := ExtractWorkingMemoryTags("<working_memory>has <nested> tags</working_memory>")
		assertSliceEqual(t, result, []string{"<working_memory>has <nested> tags</working_memory>"})
	})

	t.Run("should handle empty content between tags", func(t *testing.T) {
		result := ExtractWorkingMemoryTags("<working_memory></working_memory>")
		assertSliceEqual(t, result, []string{"<working_memory></working_memory>"})
	})

	t.Run("should handle prefix and suffix text", func(t *testing.T) {
		result := ExtractWorkingMemoryTags("prefix <working_memory>content</working_memory> suffix")
		assertSliceEqual(t, result, []string{"<working_memory>content</working_memory>"})
	})
}

func TestExtractWorkingMemoryContent(t *testing.T) {
	t.Run("should extract content without tags", func(t *testing.T) {
		result := ExtractWorkingMemoryContent("<working_memory>hello world</working_memory>")
		assertEqual(t, result, "hello world")
	})

	t.Run("should return first match content only", func(t *testing.T) {
		result := ExtractWorkingMemoryContent(
			"<working_memory>first</working_memory> <working_memory>second</working_memory>",
		)
		assertEqual(t, result, "first")
	})

	t.Run("should handle multiline content", func(t *testing.T) {
		result := ExtractWorkingMemoryContent("<working_memory>line1\nline2</working_memory>")
		assertEqual(t, result, "line1\nline2")
	})

	t.Run("should return empty string when no tags found", func(t *testing.T) {
		result := ExtractWorkingMemoryContent("no tags here")
		assertEqual(t, result, "")
	})

	t.Run("should return empty string when only opening tag exists", func(t *testing.T) {
		result := ExtractWorkingMemoryContent("<working_memory>unclosed")
		assertEqual(t, result, "")
	})

	t.Run("should handle empty content", func(t *testing.T) {
		result := ExtractWorkingMemoryContent("<working_memory></working_memory>")
		assertEqual(t, result, "")
	})

	t.Run("should extract content with prefix text", func(t *testing.T) {
		result := ExtractWorkingMemoryContent("prefix <working_memory>content</working_memory>")
		assertEqual(t, result, "content")
	})
}

func TestRemoveWorkingMemoryTags(t *testing.T) {
	t.Run("should remove working memory tags", func(t *testing.T) {
		result := RemoveWorkingMemoryTags("<working_memory>secret</working_memory>")
		assertEqual(t, result, "")
	})

	t.Run("should remove tags and preserve surrounding text", func(t *testing.T) {
		result := RemoveWorkingMemoryTags("Hello <working_memory>secret</working_memory> world")
		assertEqual(t, result, "Hello  world")
	})

	t.Run("should remove multiple tags", func(t *testing.T) {
		result := RemoveWorkingMemoryTags(
			"<working_memory>a</working_memory> middle <working_memory>b</working_memory>",
		)
		assertEqual(t, result, " middle ")
	})

	t.Run("should handle text with no tags", func(t *testing.T) {
		result := RemoveWorkingMemoryTags("no tags here")
		assertEqual(t, result, "no tags here")
	})

	t.Run("should handle unclosed tags by preserving them", func(t *testing.T) {
		result := RemoveWorkingMemoryTags("before <working_memory>unclosed")
		assertEqual(t, result, "before <working_memory>unclosed")
	})

	t.Run("should remove adjacent tags", func(t *testing.T) {
		result := RemoveWorkingMemoryTags(
			"<working_memory>a</working_memory><working_memory>b</working_memory>",
		)
		assertEqual(t, result, "")
	})
}

func TestPerformanceReDoSPrevention(t *testing.T) {
	// Generate pathological input that causes O(n^2) behavior with regex.
	createPathologicalInput := func(n int) string {
		return "<working_memory>" + strings.Repeat("<working_memory>a", n)
	}

	t.Run("should handle pathological input without performance degradation", func(t *testing.T) {
		input := createPathologicalInput(5000)

		start := time.Now()
		result := ExtractWorkingMemoryTags(input)
		elapsed := time.Since(start)

		// Helper should complete in under 5ms even for large inputs.
		if elapsed > 5*time.Millisecond {
			t.Errorf("took %v, expected under 5ms", elapsed)
		}
		// Should return nil since there's no closing tag.
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("should maintain linear performance as input grows", func(t *testing.T) {
		for _, n := range []int{5000, 10000, 20000} {
			input := createPathologicalInput(n)

			start := time.Now()
			ExtractWorkingMemoryTags(input)
			elapsed := time.Since(start)

			if elapsed > 5*time.Millisecond {
				t.Errorf("n=%d took %v, expected under 5ms", n, elapsed)
			}
		}
	})
}

// --- test helpers ---

func assertEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func assertSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("length mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

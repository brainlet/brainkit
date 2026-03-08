// Ported from: packages/core/src/relevance/relevance-score-provider.test.ts
package relevance

import (
	"strings"
	"testing"
)

func TestCreateSimilarityPrompt(t *testing.T) {
	t.Run("includes query and text in prompt", func(t *testing.T) {
		prompt := CreateSimilarityPrompt("what is Go?", "Go is a programming language.")
		if !strings.Contains(prompt, "what is Go?") {
			t.Error("prompt should contain the query")
		}
		if !strings.Contains(prompt, "Go is a programming language.") {
			t.Error("prompt should contain the text")
		}
	})

	t.Run("includes scoring instruction", func(t *testing.T) {
		prompt := CreateSimilarityPrompt("q", "t")
		if !strings.Contains(prompt, "0 to 1") {
			t.Error("prompt should contain the scoring scale instruction")
		}
	})

	t.Run("includes Query: label", func(t *testing.T) {
		prompt := CreateSimilarityPrompt("my query", "my text")
		if !strings.Contains(prompt, "Query: my query") {
			t.Error("prompt should include Query: label with query value")
		}
	})

	t.Run("includes Text: label", func(t *testing.T) {
		prompt := CreateSimilarityPrompt("my query", "my text")
		if !strings.Contains(prompt, "Text: my text") {
			t.Error("prompt should include Text: label with text value")
		}
	})

	t.Run("includes relevance score instruction at end", func(t *testing.T) {
		prompt := CreateSimilarityPrompt("q", "t")
		if !strings.HasSuffix(prompt, "Relevance score (0-1):") {
			t.Error("prompt should end with relevance score instruction")
		}
	})

	t.Run("handles empty strings", func(t *testing.T) {
		prompt := CreateSimilarityPrompt("", "")
		if !strings.Contains(prompt, "Query: ") {
			t.Error("prompt should still contain Query: label")
		}
		if !strings.Contains(prompt, "Text: ") {
			t.Error("prompt should still contain Text: label")
		}
	})

	t.Run("handles special characters", func(t *testing.T) {
		prompt := CreateSimilarityPrompt("what's \"this\"?", "line1\nline2")
		if !strings.Contains(prompt, "what's \"this\"?") {
			t.Error("prompt should preserve special characters in query")
		}
		if !strings.Contains(prompt, "line1\nline2") {
			t.Error("prompt should preserve newlines in text")
		}
	})
}

func TestRelevanceScoreProviderInterface(t *testing.T) {
	t.Run("interface has GetRelevanceScore method", func(t *testing.T) {
		// Verify the interface can be satisfied by a mock implementation.
		var provider RelevanceScoreProvider = &mockRelevanceProvider{score: 0.5}
		score, err := provider.GetRelevanceScore("query", "text")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if score != 0.5 {
			t.Errorf("score = %f, want 0.5", score)
		}
	})
}

// mockRelevanceProvider is a test double for RelevanceScoreProvider.
type mockRelevanceProvider struct {
	score float64
	err   error
}

func (m *mockRelevanceProvider) GetRelevanceScore(_, _ string) (float64, error) {
	return m.score, m.err
}

// Ported from: packages/groq/src/convert-groq-usage.test.ts
package groq

import (
	"testing"
)

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }
func boolPtr(v bool) *bool    { return &v }

func TestConvertGroqUsage_NilUsage(t *testing.T) {
	t.Run("should return nil values when usage is nil", func(t *testing.T) {
		result := ConvertGroqUsage(nil)

		if result.InputTokens.Total != nil {
			t.Errorf("expected InputTokens.Total nil, got %v", result.InputTokens.Total)
		}
		if result.InputTokens.NoCache != nil {
			t.Errorf("expected InputTokens.NoCache nil, got %v", result.InputTokens.NoCache)
		}
		if result.InputTokens.CacheRead != nil {
			t.Errorf("expected InputTokens.CacheRead nil, got %v", result.InputTokens.CacheRead)
		}
		if result.InputTokens.CacheWrite != nil {
			t.Errorf("expected InputTokens.CacheWrite nil, got %v", result.InputTokens.CacheWrite)
		}
		if result.OutputTokens.Total != nil {
			t.Errorf("expected OutputTokens.Total nil, got %v", result.OutputTokens.Total)
		}
		if result.OutputTokens.Text != nil {
			t.Errorf("expected OutputTokens.Text nil, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning != nil {
			t.Errorf("expected OutputTokens.Reasoning nil, got %v", result.OutputTokens.Reasoning)
		}
		if result.Raw != nil {
			t.Errorf("expected Raw nil, got %v", result.Raw)
		}
	})
}

func TestConvertGroqUsage_BasicUsage(t *testing.T) {
	t.Run("should convert basic usage without token details", func(t *testing.T) {
		result := ConvertGroqUsage(&GroqTokenUsage{
			PromptTokens:     intPtr(20),
			CompletionTokens: intPtr(10),
		})

		if result.InputTokens.Total == nil || *result.InputTokens.Total != 20 {
			t.Errorf("expected InputTokens.Total 20, got %v", result.InputTokens.Total)
		}
		if result.InputTokens.NoCache == nil || *result.InputTokens.NoCache != 20 {
			t.Errorf("expected InputTokens.NoCache 20, got %v", result.InputTokens.NoCache)
		}
		if result.InputTokens.CacheRead != nil {
			t.Errorf("expected InputTokens.CacheRead nil, got %v", result.InputTokens.CacheRead)
		}
		if result.InputTokens.CacheWrite != nil {
			t.Errorf("expected InputTokens.CacheWrite nil, got %v", result.InputTokens.CacheWrite)
		}
		if result.OutputTokens.Total == nil || *result.OutputTokens.Total != 10 {
			t.Errorf("expected OutputTokens.Total 10, got %v", result.OutputTokens.Total)
		}
		if result.OutputTokens.Text == nil || *result.OutputTokens.Text != 10 {
			t.Errorf("expected OutputTokens.Text 10, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning != nil {
			t.Errorf("expected OutputTokens.Reasoning nil, got %v", result.OutputTokens.Reasoning)
		}
		if result.Raw == nil {
			t.Fatal("expected Raw non-nil")
		}
	})
}

func TestConvertGroqUsage_ReasoningTokens(t *testing.T) {
	t.Run("should extract reasoning tokens from completion_tokens_details", func(t *testing.T) {
		result := ConvertGroqUsage(&GroqTokenUsage{
			PromptTokens:     intPtr(79),
			CompletionTokens: intPtr(40),
			CompletionTokensDetails: &struct {
				ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
			}{
				ReasoningTokens: intPtr(21),
			},
		})

		if result.InputTokens.Total == nil || *result.InputTokens.Total != 79 {
			t.Errorf("expected InputTokens.Total 79, got %v", result.InputTokens.Total)
		}
		if result.InputTokens.NoCache == nil || *result.InputTokens.NoCache != 79 {
			t.Errorf("expected InputTokens.NoCache 79, got %v", result.InputTokens.NoCache)
		}
		if result.OutputTokens.Total == nil || *result.OutputTokens.Total != 40 {
			t.Errorf("expected OutputTokens.Total 40, got %v", result.OutputTokens.Total)
		}
		// text = 40 - 21 = 19
		if result.OutputTokens.Text == nil || *result.OutputTokens.Text != 19 {
			t.Errorf("expected OutputTokens.Text 19, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning == nil || *result.OutputTokens.Reasoning != 21 {
			t.Errorf("expected OutputTokens.Reasoning 21, got %v", result.OutputTokens.Reasoning)
		}
	})
}

func TestConvertGroqUsage_NilReasoningTokens(t *testing.T) {
	t.Run("should handle nil reasoning_tokens in completion_tokens_details", func(t *testing.T) {
		result := ConvertGroqUsage(&GroqTokenUsage{
			PromptTokens:     intPtr(20),
			CompletionTokens: intPtr(10),
			CompletionTokensDetails: &struct {
				ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
			}{
				ReasoningTokens: nil,
			},
		})

		if result.OutputTokens.Text == nil || *result.OutputTokens.Text != 10 {
			t.Errorf("expected OutputTokens.Text 10, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning != nil {
			t.Errorf("expected OutputTokens.Reasoning nil, got %v", result.OutputTokens.Reasoning)
		}
	})
}

func TestConvertGroqUsage_NilCompletionTokensDetails(t *testing.T) {
	t.Run("should handle nil completion_tokens_details", func(t *testing.T) {
		result := ConvertGroqUsage(&GroqTokenUsage{
			PromptTokens:            intPtr(20),
			CompletionTokens:        intPtr(10),
			CompletionTokensDetails: nil,
		})

		if result.OutputTokens.Text == nil || *result.OutputTokens.Text != 10 {
			t.Errorf("expected OutputTokens.Text 10, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning != nil {
			t.Errorf("expected OutputTokens.Reasoning nil, got %v", result.OutputTokens.Reasoning)
		}
	})
}

func TestConvertGroqUsage_ZeroReasoningTokens(t *testing.T) {
	t.Run("should handle zero reasoning tokens", func(t *testing.T) {
		result := ConvertGroqUsage(&GroqTokenUsage{
			PromptTokens:     intPtr(20),
			CompletionTokens: intPtr(10),
			CompletionTokensDetails: &struct {
				ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
			}{
				ReasoningTokens: intPtr(0),
			},
		})

		if result.OutputTokens.Text == nil || *result.OutputTokens.Text != 10 {
			t.Errorf("expected OutputTokens.Text 10, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning == nil || *result.OutputTokens.Reasoning != 0 {
			t.Errorf("expected OutputTokens.Reasoning 0, got %v", result.OutputTokens.Reasoning)
		}
	})
}

func TestConvertGroqUsage_AllReasoningTokens(t *testing.T) {
	t.Run("should handle all tokens being reasoning tokens", func(t *testing.T) {
		result := ConvertGroqUsage(&GroqTokenUsage{
			PromptTokens:     intPtr(20),
			CompletionTokens: intPtr(50),
			CompletionTokensDetails: &struct {
				ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
			}{
				ReasoningTokens: intPtr(50),
			},
		})

		if result.OutputTokens.Total == nil || *result.OutputTokens.Total != 50 {
			t.Errorf("expected OutputTokens.Total 50, got %v", result.OutputTokens.Total)
		}
		// text = 50 - 50 = 0
		if result.OutputTokens.Text == nil || *result.OutputTokens.Text != 0 {
			t.Errorf("expected OutputTokens.Text 0, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning == nil || *result.OutputTokens.Reasoning != 50 {
			t.Errorf("expected OutputTokens.Reasoning 50, got %v", result.OutputTokens.Reasoning)
		}
	})
}

func TestConvertGroqUsage_MissingTokenCounts(t *testing.T) {
	t.Run("should handle missing prompt_tokens and completion_tokens", func(t *testing.T) {
		result := ConvertGroqUsage(&GroqTokenUsage{})

		if result.InputTokens.Total == nil || *result.InputTokens.Total != 0 {
			t.Errorf("expected InputTokens.Total 0, got %v", result.InputTokens.Total)
		}
		if result.InputTokens.NoCache == nil || *result.InputTokens.NoCache != 0 {
			t.Errorf("expected InputTokens.NoCache 0, got %v", result.InputTokens.NoCache)
		}
		if result.OutputTokens.Total == nil || *result.OutputTokens.Total != 0 {
			t.Errorf("expected OutputTokens.Total 0, got %v", result.OutputTokens.Total)
		}
		if result.OutputTokens.Text == nil || *result.OutputTokens.Text != 0 {
			t.Errorf("expected OutputTokens.Text 0, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning != nil {
			t.Errorf("expected OutputTokens.Reasoning nil, got %v", result.OutputTokens.Reasoning)
		}
		if result.Raw == nil {
			t.Fatal("expected Raw non-nil")
		}
	})
}

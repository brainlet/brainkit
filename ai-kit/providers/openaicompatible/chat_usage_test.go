// Ported from: packages/openai-compatible/src/chat/convert-openai-compatible-chat-usage.ts (tests)
package openaicompatible

import (
	"testing"
)

func intPtr(v int) *int { return &v }

func TestConvertChatUsage(t *testing.T) {
	t.Run("should return zero usage when input is nil", func(t *testing.T) {
		usage := ConvertChatUsage(nil)

		if usage.InputTokens.Total != nil {
			t.Errorf("expected nil input total, got %v", *usage.InputTokens.Total)
		}
		if usage.OutputTokens.Total != nil {
			t.Errorf("expected nil output total, got %v", *usage.OutputTokens.Total)
		}
		if usage.Raw != nil {
			t.Errorf("expected nil raw, got %v", usage.Raw)
		}
	})

	t.Run("should convert basic usage with prompt and completion tokens", func(t *testing.T) {
		promptTokens := 10
		completionTokens := 20
		usage := ConvertChatUsage(&OpenAICompatibleTokenUsage{
			PromptTokens:     &promptTokens,
			CompletionTokens: &completionTokens,
		})

		if usage.InputTokens.Total == nil || *usage.InputTokens.Total != 10 {
			t.Errorf("expected input total 10, got %v", usage.InputTokens.Total)
		}
		if usage.OutputTokens.Total == nil || *usage.OutputTokens.Total != 20 {
			t.Errorf("expected output total 20, got %v", usage.OutputTokens.Total)
		}
	})

	t.Run("should calculate no-cache tokens as prompt minus cached", func(t *testing.T) {
		promptTokens := 100
		completionTokens := 50
		cachedTokens := 30
		usage := ConvertChatUsage(&OpenAICompatibleTokenUsage{
			PromptTokens:     &promptTokens,
			CompletionTokens: &completionTokens,
			PromptTokensDetails: &struct {
				CachedTokens *int `json:"cached_tokens,omitempty"`
			}{
				CachedTokens: &cachedTokens,
			},
		})

		if usage.InputTokens.NoCache == nil || *usage.InputTokens.NoCache != 70 {
			t.Errorf("expected no-cache 70, got %v", usage.InputTokens.NoCache)
		}
		if usage.InputTokens.CacheRead == nil || *usage.InputTokens.CacheRead != 30 {
			t.Errorf("expected cache-read 30, got %v", usage.InputTokens.CacheRead)
		}
	})

	t.Run("should calculate text tokens as completion minus reasoning", func(t *testing.T) {
		promptTokens := 10
		completionTokens := 100
		reasoningTokens := 40
		usage := ConvertChatUsage(&OpenAICompatibleTokenUsage{
			PromptTokens:     &promptTokens,
			CompletionTokens: &completionTokens,
			CompletionTokensDetails: &struct {
				ReasoningTokens          *int `json:"reasoning_tokens,omitempty"`
				AcceptedPredictionTokens *int `json:"accepted_prediction_tokens,omitempty"`
				RejectedPredictionTokens *int `json:"rejected_prediction_tokens,omitempty"`
			}{
				ReasoningTokens: &reasoningTokens,
			},
		})

		if usage.OutputTokens.Text == nil || *usage.OutputTokens.Text != 60 {
			t.Errorf("expected text tokens 60, got %v", usage.OutputTokens.Text)
		}
		if usage.OutputTokens.Reasoning == nil || *usage.OutputTokens.Reasoning != 40 {
			t.Errorf("expected reasoning tokens 40, got %v", usage.OutputTokens.Reasoning)
		}
	})

	t.Run("should include raw usage data", func(t *testing.T) {
		promptTokens := 10
		completionTokens := 20
		usage := ConvertChatUsage(&OpenAICompatibleTokenUsage{
			PromptTokens:     &promptTokens,
			CompletionTokens: &completionTokens,
		})

		if usage.Raw == nil {
			t.Fatal("expected non-nil raw usage")
		}
		rawPrompt, ok := usage.Raw["prompt_tokens"].(float64)
		if !ok {
			t.Fatalf("expected raw prompt_tokens to be float64, got %T", usage.Raw["prompt_tokens"])
		}
		if int(rawPrompt) != 10 {
			t.Errorf("expected raw prompt_tokens 10, got %v", rawPrompt)
		}
	})

	t.Run("should handle nil prompt and completion tokens", func(t *testing.T) {
		usage := ConvertChatUsage(&OpenAICompatibleTokenUsage{})

		if usage.InputTokens.Total == nil || *usage.InputTokens.Total != 0 {
			t.Errorf("expected input total 0, got %v", usage.InputTokens.Total)
		}
		if usage.OutputTokens.Total == nil || *usage.OutputTokens.Total != 0 {
			t.Errorf("expected output total 0, got %v", usage.OutputTokens.Total)
		}
	})
}

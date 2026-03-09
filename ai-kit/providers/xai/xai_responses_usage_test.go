// Ported from: packages/xai/src/responses/convert-xai-responses-usage.test.ts
package xai

import (
	"testing"
)

func TestConvertXaiResponsesUsage_BasicUsage(t *testing.T) {
	t.Run("should convert basic usage without caching or reasoning", func(t *testing.T) {
		result := convertXaiResponsesUsage(XaiResponsesUsage{
			InputTokens:  100,
			OutputTokens: 50,
		})

		if intVal(result.InputTokens.Total) != 100 {
			t.Errorf("expected InputTokens.Total 100, got %d", intVal(result.InputTokens.Total))
		}
		if intVal(result.InputTokens.NoCache) != 100 {
			t.Errorf("expected InputTokens.NoCache 100, got %d", intVal(result.InputTokens.NoCache))
		}
		if intVal(result.InputTokens.CacheRead) != 0 {
			t.Errorf("expected InputTokens.CacheRead 0, got %d", intVal(result.InputTokens.CacheRead))
		}
		if result.InputTokens.CacheWrite != nil {
			t.Errorf("expected InputTokens.CacheWrite nil, got %v", result.InputTokens.CacheWrite)
		}
		if intVal(result.OutputTokens.Total) != 50 {
			t.Errorf("expected OutputTokens.Total 50, got %d", intVal(result.OutputTokens.Total))
		}
		if intVal(result.OutputTokens.Text) != 50 {
			t.Errorf("expected OutputTokens.Text 50, got %d", intVal(result.OutputTokens.Text))
		}
		if intVal(result.OutputTokens.Reasoning) != 0 {
			t.Errorf("expected OutputTokens.Reasoning 0, got %d", intVal(result.OutputTokens.Reasoning))
		}
		if result.Raw == nil {
			t.Error("expected non-nil raw usage")
		}
	})
}

func TestConvertXaiResponsesUsage_ReasoningTokens(t *testing.T) {
	t.Run("should convert usage with reasoning tokens", func(t *testing.T) {
		reasoningTokens := 380
		result := convertXaiResponsesUsage(XaiResponsesUsage{
			InputTokens:  1941,
			OutputTokens: 583,
			OutputTokensDetails: &XaiResponsesOutputTokensDetails{
				ReasoningTokens: &reasoningTokens,
			},
		})

		if intVal(result.OutputTokens.Total) != 583 {
			t.Errorf("expected OutputTokens.Total 583, got %d", intVal(result.OutputTokens.Total))
		}
		if intVal(result.OutputTokens.Reasoning) != 380 {
			t.Errorf("expected OutputTokens.Reasoning 380, got %d", intVal(result.OutputTokens.Reasoning))
		}
		// text = 583 - 380 = 203
		if intVal(result.OutputTokens.Text) != 203 {
			t.Errorf("expected OutputTokens.Text 203, got %d", intVal(result.OutputTokens.Text))
		}
	})
}

func TestConvertXaiResponsesUsage_CachedTokens(t *testing.T) {
	t.Run("should convert usage with cached input tokens", func(t *testing.T) {
		cachedTokens := 150
		result := convertXaiResponsesUsage(XaiResponsesUsage{
			InputTokens:  200,
			OutputTokens: 50,
			InputTokensDetails: &XaiResponsesInputTokensDetails{
				CachedTokens: &cachedTokens,
			},
		})

		if intVal(result.InputTokens.Total) != 200 {
			t.Errorf("expected InputTokens.Total 200, got %d", intVal(result.InputTokens.Total))
		}
		if intVal(result.InputTokens.CacheRead) != 150 {
			t.Errorf("expected InputTokens.CacheRead 150, got %d", intVal(result.InputTokens.CacheRead))
		}
		// noCache = 200 - 150 = 50
		if intVal(result.InputTokens.NoCache) != 50 {
			t.Errorf("expected InputTokens.NoCache 50, got %d", intVal(result.InputTokens.NoCache))
		}
	})
}

func TestConvertXaiResponsesUsage_NonInclusiveReporting(t *testing.T) {
	t.Run("should handle cached_tokens exceeding input_tokens (non-inclusive reporting)", func(t *testing.T) {
		cachedTokens := 4328
		result := convertXaiResponsesUsage(XaiResponsesUsage{
			InputTokens:  4142,
			OutputTokens: 254,
			InputTokensDetails: &XaiResponsesInputTokensDetails{
				CachedTokens: &cachedTokens,
			},
		})

		// Non-inclusive: total = input_tokens + cached_tokens = 4142 + 4328 = 8470
		if intVal(result.InputTokens.Total) != 8470 {
			t.Errorf("expected InputTokens.Total 8470, got %d", intVal(result.InputTokens.Total))
		}
		if intVal(result.InputTokens.CacheRead) != 4328 {
			t.Errorf("expected InputTokens.CacheRead 4328, got %d", intVal(result.InputTokens.CacheRead))
		}
		// noCache = input_tokens = 4142 (non-inclusive)
		if intVal(result.InputTokens.NoCache) != 4142 {
			t.Errorf("expected InputTokens.NoCache 4142, got %d", intVal(result.InputTokens.NoCache))
		}
	})
}

func TestConvertXaiResponsesUsage_Combined(t *testing.T) {
	t.Run("should convert usage with both cached input and reasoning", func(t *testing.T) {
		cachedTokens := 150
		reasoningTokens := 380
		result := convertXaiResponsesUsage(XaiResponsesUsage{
			InputTokens:  200,
			OutputTokens: 583,
			InputTokensDetails: &XaiResponsesInputTokensDetails{
				CachedTokens: &cachedTokens,
			},
			OutputTokensDetails: &XaiResponsesOutputTokensDetails{
				ReasoningTokens: &reasoningTokens,
			},
		})

		if intVal(result.InputTokens.Total) != 200 {
			t.Errorf("expected InputTokens.Total 200, got %d", intVal(result.InputTokens.Total))
		}
		if intVal(result.InputTokens.CacheRead) != 150 {
			t.Errorf("expected InputTokens.CacheRead 150, got %d", intVal(result.InputTokens.CacheRead))
		}
		if intVal(result.InputTokens.NoCache) != 50 {
			t.Errorf("expected InputTokens.NoCache 50, got %d", intVal(result.InputTokens.NoCache))
		}
		if intVal(result.OutputTokens.Total) != 583 {
			t.Errorf("expected OutputTokens.Total 583, got %d", intVal(result.OutputTokens.Total))
		}
		if intVal(result.OutputTokens.Reasoning) != 380 {
			t.Errorf("expected OutputTokens.Reasoning 380, got %d", intVal(result.OutputTokens.Reasoning))
		}
		if intVal(result.OutputTokens.Text) != 203 {
			t.Errorf("expected OutputTokens.Text 203, got %d", intVal(result.OutputTokens.Text))
		}
	})
}

func TestConvertXaiResponsesUsage_RawPreserved(t *testing.T) {
	t.Run("should preserve raw usage data", func(t *testing.T) {
		cachedTokens := 2
		reasoningTokens := 317
		totalTokens := 331
		result := convertXaiResponsesUsage(XaiResponsesUsage{
			InputTokens:  12,
			OutputTokens: 319,
			TotalTokens:  &totalTokens,
			InputTokensDetails: &XaiResponsesInputTokensDetails{
				CachedTokens: &cachedTokens,
			},
			OutputTokensDetails: &XaiResponsesOutputTokensDetails{
				ReasoningTokens: &reasoningTokens,
			},
		})

		if result.Raw == nil {
			t.Fatal("expected non-nil raw usage")
		}

		if result.Raw["input_tokens"] != 12 {
			t.Errorf("expected raw input_tokens 12, got %v", result.Raw["input_tokens"])
		}
		if result.Raw["output_tokens"] != 319 {
			t.Errorf("expected raw output_tokens 319, got %v", result.Raw["output_tokens"])
		}
	})
}

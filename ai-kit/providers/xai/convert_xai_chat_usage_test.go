// Ported from: packages/xai/src/convert-xai-chat-usage.test.ts
package xai

import (
	"testing"
)

func intVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func TestConvertXaiChatUsage_BasicUsage(t *testing.T) {
	t.Run("should extract usage correctly", func(t *testing.T) {
		usage := convertXaiChatUsage(XaiChatUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		})

		if intVal(usage.InputTokens.Total) != 100 {
			t.Errorf("expected InputTokens.Total 100, got %d", intVal(usage.InputTokens.Total))
		}
		if intVal(usage.OutputTokens.Total) != 50 {
			t.Errorf("expected OutputTokens.Total 50, got %v", usage.OutputTokens.Total)
		}
	})
}

func TestConvertXaiChatUsage_ReasoningTokens(t *testing.T) {
	t.Run("should extract reasoning tokens from completion_tokens_details", func(t *testing.T) {
		reasoningTokens := 30
		usage := convertXaiChatUsage(XaiChatUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			CompletionTokensDetails: &XaiCompletionTokensDetails{
				ReasoningTokens: &reasoningTokens,
			},
		})

		if intVal(usage.OutputTokens.Reasoning) != 30 {
			t.Errorf("expected OutputTokens.Reasoning 30, got %d", intVal(usage.OutputTokens.Reasoning))
		}
		// text = completionTokens = 50 (the output total includes reasoning separately)
		if intVal(usage.OutputTokens.Text) != 50 {
			t.Errorf("expected OutputTokens.Text 50, got %d", intVal(usage.OutputTokens.Text))
		}
	})
}

func TestConvertXaiChatUsage_CachedTokens(t *testing.T) {
	t.Run("should extract cached tokens from prompt_tokens_details", func(t *testing.T) {
		cachedTokens := 25
		usage := convertXaiChatUsage(XaiChatUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			PromptTokensDetails: &XaiPromptTokensDetails{
				CachedTokens: &cachedTokens,
			},
		})

		if intVal(usage.InputTokens.CacheRead) != 25 {
			t.Errorf("expected InputTokens.CacheRead 25, got %d", intVal(usage.InputTokens.CacheRead))
		}
	})
}

func TestConvertXaiChatUsage_NonInclusiveReporting(t *testing.T) {
	t.Run("should calculate noCache tokens correctly (non-inclusive reporting)", func(t *testing.T) {
		cachedTokens := 30
		usage := convertXaiChatUsage(XaiChatUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			PromptTokensDetails: &XaiPromptTokensDetails{
				CachedTokens: &cachedTokens,
			},
		})

		// noCache = total - cacheRead = 100 - 30 = 70
		if intVal(usage.InputTokens.NoCache) != 70 {
			t.Errorf("expected InputTokens.NoCache 70, got %d", intVal(usage.InputTokens.NoCache))
		}
	})
}

func TestConvertXaiChatUsage_NullDetails(t *testing.T) {
	t.Run("should handle nil prompt_tokens_details and completion_tokens_details", func(t *testing.T) {
		usage := convertXaiChatUsage(XaiChatUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		})

		if intVal(usage.InputTokens.CacheRead) != 0 {
			t.Errorf("expected InputTokens.CacheRead 0, got %d", intVal(usage.InputTokens.CacheRead))
		}
		if intVal(usage.OutputTokens.Reasoning) != 0 {
			t.Errorf("expected OutputTokens.Reasoning 0, got %d", intVal(usage.OutputTokens.Reasoning))
		}
		if intVal(usage.InputTokens.NoCache) != 100 {
			t.Errorf("expected InputTokens.NoCache 100, got %d", intVal(usage.InputTokens.NoCache))
		}
		if intVal(usage.OutputTokens.Text) != 50 {
			t.Errorf("expected OutputTokens.Text 50, got %d", intVal(usage.OutputTokens.Text))
		}
	})
}

func TestConvertXaiChatUsage_ZeroReasoningTokens(t *testing.T) {
	t.Run("should handle zero reasoning tokens", func(t *testing.T) {
		reasoningTokens := 0
		usage := convertXaiChatUsage(XaiChatUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			CompletionTokensDetails: &XaiCompletionTokensDetails{
				ReasoningTokens: &reasoningTokens,
			},
		})

		if intVal(usage.OutputTokens.Reasoning) != 0 {
			t.Errorf("expected OutputTokens.Reasoning 0, got %d", intVal(usage.OutputTokens.Reasoning))
		}
		if intVal(usage.OutputTokens.Text) != 50 {
			t.Errorf("expected OutputTokens.Text 50, got %d", intVal(usage.OutputTokens.Text))
		}
	})
}

func TestConvertXaiChatUsage_PreservesRaw(t *testing.T) {
	t.Run("should preserve raw usage data", func(t *testing.T) {
		reasoningTokens := 30
		cachedTokens := 25
		usage := convertXaiChatUsage(XaiChatUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			PromptTokensDetails: &XaiPromptTokensDetails{
				CachedTokens: &cachedTokens,
			},
			CompletionTokensDetails: &XaiCompletionTokensDetails{
				ReasoningTokens: &reasoningTokens,
			},
		})

		if usage.Raw == nil {
			t.Fatal("expected Raw to be non-nil")
		}
		if usage.Raw["prompt_tokens"] != 100 {
			t.Errorf("expected raw prompt_tokens 100, got %v", usage.Raw["prompt_tokens"])
		}
		if usage.Raw["completion_tokens"] != 50 {
			t.Errorf("expected raw completion_tokens 50, got %v", usage.Raw["completion_tokens"])
		}
	})
}

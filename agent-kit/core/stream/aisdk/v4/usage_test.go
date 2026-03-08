// Ported from: packages/core/src/stream/aisdk/v4/usage.test.ts
package v4

import (
	"testing"
)

func TestConvertV4Usage(t *testing.T) {
	t.Run("should return empty UsageStats for nil usage", func(t *testing.T) {
		result := ConvertV4Usage(nil)
		if result.InputTokens != nil {
			t.Errorf("expected nil inputTokens, got %v", result.InputTokens)
		}
		if result.OutputTokens != nil {
			t.Errorf("expected nil outputTokens, got %v", result.OutputTokens)
		}
	})

	t.Run("should convert V4 usage to UsageStats", func(t *testing.T) {
		result := ConvertV4Usage(&LanguageModelUsageV4{
			PromptTokens:     10,
			CompletionTokens: 20,
		})
		if result.InputTokens == nil {
			t.Fatal("expected non-nil inputTokens")
		}
		if *result.InputTokens != 10 {
			t.Errorf("expected inputTokens 10, got %d", *result.InputTokens)
		}
		if result.OutputTokens == nil {
			t.Fatal("expected non-nil outputTokens")
		}
		if *result.OutputTokens != 20 {
			t.Errorf("expected outputTokens 20, got %d", *result.OutputTokens)
		}
	})

	t.Run("should handle zero token counts", func(t *testing.T) {
		result := ConvertV4Usage(&LanguageModelUsageV4{
			PromptTokens:     0,
			CompletionTokens: 0,
		})
		if result.InputTokens == nil {
			t.Fatal("expected non-nil inputTokens")
		}
		if *result.InputTokens != 0 {
			t.Errorf("expected inputTokens 0, got %d", *result.InputTokens)
		}
		if *result.OutputTokens != 0 {
			t.Errorf("expected outputTokens 0, got %d", *result.OutputTokens)
		}
	})
}

func TestLanguageModelUsageV4(t *testing.T) {
	t.Run("should hold token counts", func(t *testing.T) {
		totalTokens := 30
		usage := LanguageModelUsageV4{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      &totalTokens,
		}
		if usage.PromptTokens != 10 {
			t.Errorf("expected promptTokens 10, got %d", usage.PromptTokens)
		}
		if usage.CompletionTokens != 20 {
			t.Errorf("expected completionTokens 20, got %d", usage.CompletionTokens)
		}
		if *usage.TotalTokens != 30 {
			t.Errorf("expected totalTokens 30, got %d", *usage.TotalTokens)
		}
	})

	t.Run("should allow nil totalTokens", func(t *testing.T) {
		usage := LanguageModelUsageV4{
			PromptTokens:     5,
			CompletionTokens: 10,
		}
		if usage.TotalTokens != nil {
			t.Error("expected nil totalTokens")
		}
	})
}

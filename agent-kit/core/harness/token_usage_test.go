// Ported from: packages/core/src/harness/token-usage.test.ts
package harness

import (
	"testing"
)

func TestStepFinishTokenUsageExtraction(t *testing.T) {
	t.Skip("not yet implemented - requires processStream method which handles step-finish chunks and extracts token usage")

	// The TS tests verify:
	// 1. Token usage extraction from AI SDK v5/v6 format (inputTokens/outputTokens)
	// 2. Token usage extraction from legacy v4 format (promptTokens/completionTokens)
	// 3. Accumulation of token usage across multiple step-finish chunks
	//
	// All rely on the private processStream() method which is not yet ported to Go.
	// When processStream is implemented, the following subtests should be enabled:
	//
	// t.Run("extracts token usage from AI SDK v5/v6 format", func(t *testing.T) { ... })
	// t.Run("extracts token usage from legacy v4 format", func(t *testing.T) { ... })
	// t.Run("accumulates token usage across multiple step-finish chunks", func(t *testing.T) { ... })
}

func TestGetTokenUsage(t *testing.T) {
	t.Run("returns zero token usage initially", func(t *testing.T) {
		h, err := New(HarnessConfig{
			ID: "test-harness",
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}

		usage := h.GetTokenUsage()
		if usage.PromptTokens != 0 {
			t.Errorf("expected PromptTokens = 0, got %d", usage.PromptTokens)
		}
		if usage.CompletionTokens != 0 {
			t.Errorf("expected CompletionTokens = 0, got %d", usage.CompletionTokens)
		}
		if usage.TotalTokens != 0 {
			t.Errorf("expected TotalTokens = 0, got %d", usage.TotalTokens)
		}
	})

	t.Run("reflects manually set token usage", func(t *testing.T) {
		h, err := New(HarnessConfig{
			ID: "test-harness",
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}

		// Directly set tokenUsage (accessible within same package)
		h.mu.Lock()
		h.tokenUsage = TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
		h.mu.Unlock()

		usage := h.GetTokenUsage()
		if usage.PromptTokens != 100 {
			t.Errorf("expected PromptTokens = 100, got %d", usage.PromptTokens)
		}
		if usage.CompletionTokens != 50 {
			t.Errorf("expected CompletionTokens = 50, got %d", usage.CompletionTokens)
		}
		if usage.TotalTokens != 150 {
			t.Errorf("expected TotalTokens = 150, got %d", usage.TotalTokens)
		}
	})

	t.Run("resets on CreateThread", func(t *testing.T) {
		h, err := New(HarnessConfig{
			ID: "test-harness",
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}

		// Set some token usage
		h.mu.Lock()
		h.tokenUsage = TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
		h.mu.Unlock()

		// Creating a new thread should reset token usage
		_, err = h.CreateThread("")
		if err != nil {
			t.Fatalf("CreateThread failed: %v", err)
		}

		usage := h.GetTokenUsage()
		if usage.PromptTokens != 0 || usage.CompletionTokens != 0 || usage.TotalTokens != 0 {
			t.Errorf("expected zero token usage after CreateThread, got %+v", usage)
		}
	})
}

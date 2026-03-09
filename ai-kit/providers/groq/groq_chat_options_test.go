// Ported from: packages/groq/src/groq-chat-options.test.ts
package groq

import (
	"testing"
)

func TestGroqLanguageModelOptions_ReasoningEffort(t *testing.T) {
	t.Run("accepts valid reasoningEffort values", func(t *testing.T) {
		validValues := []string{"none", "default", "low", "medium", "high"}

		for _, value := range validValues {
			result, err := GroqLanguageModelOptionsSchema.Validate(map[string]any{
				"reasoningEffort": value,
			})
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", value, err)
			}
			if !result.Success {
				t.Errorf("expected success for reasoningEffort %q", value)
			}
			if result.Value.ReasoningEffort == nil || *result.Value.ReasoningEffort != value {
				t.Errorf("expected reasoningEffort %q, got %v", value, result.Value.ReasoningEffort)
			}
		}
	})

	t.Run("allows reasoningEffort to be undefined", func(t *testing.T) {
		result, err := GroqLanguageModelOptionsSchema.Validate(map[string]any{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Success {
			t.Error("expected success for empty options")
		}
		if result.Value.ReasoningEffort != nil {
			t.Errorf("expected reasoningEffort nil, got %v", result.Value.ReasoningEffort)
		}
	})
}

func TestGroqLanguageModelOptions_CombinedOptions(t *testing.T) {
	t.Run("accepts reasoningEffort with other valid options", func(t *testing.T) {
		result, err := GroqLanguageModelOptionsSchema.Validate(map[string]any{
			"reasoningEffort":   "high",
			"parallelToolCalls": true,
			"user":              "test-user",
			"structuredOutputs": false,
			"serviceTier":       "flex",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Success {
			t.Error("expected success for combined options")
		}
		if result.Value.ReasoningEffort == nil || *result.Value.ReasoningEffort != "high" {
			t.Errorf("expected reasoningEffort 'high', got %v", result.Value.ReasoningEffort)
		}
		if result.Value.ParallelToolCalls == nil || *result.Value.ParallelToolCalls != true {
			t.Errorf("expected parallelToolCalls true, got %v", result.Value.ParallelToolCalls)
		}
		if result.Value.User == nil || *result.Value.User != "test-user" {
			t.Errorf("expected user 'test-user', got %v", result.Value.User)
		}
	})
}

func TestGroqLanguageModelOptions_AllVariants(t *testing.T) {
	t.Run("validates all reasoningEffort variants individually", func(t *testing.T) {
		variants := []string{"none", "default", "low", "medium", "high"}

		for _, variant := range variants {
			result, err := GroqLanguageModelOptionsSchema.Validate(map[string]any{
				"reasoningEffort": variant,
			})
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", variant, err)
			}
			if !result.Success {
				t.Errorf("expected success for variant %q", variant)
			}
			if result.Value.ReasoningEffort == nil || *result.Value.ReasoningEffort != variant {
				t.Errorf("expected reasoningEffort %q, got %v", variant, result.Value.ReasoningEffort)
			}
		}
	})
}

func TestGroqLanguageModelOptions_TypeInference(t *testing.T) {
	t.Run("infers GroqLanguageModelOptions type correctly", func(t *testing.T) {
		medium := "medium"
		ptc := false
		options := GroqLanguageModelOptions{
			ReasoningEffort:   &medium,
			ParallelToolCalls: &ptc,
		}

		if options.ReasoningEffort == nil || *options.ReasoningEffort != "medium" {
			t.Errorf("expected reasoningEffort 'medium', got %v", options.ReasoningEffort)
		}
		if options.ParallelToolCalls == nil || *options.ParallelToolCalls != false {
			t.Errorf("expected parallelToolCalls false, got %v", options.ParallelToolCalls)
		}
	})
}

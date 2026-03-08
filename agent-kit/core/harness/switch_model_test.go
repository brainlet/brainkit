// Ported from: packages/core/src/harness/switch-model.test.ts
package harness

import (
	"testing"
)

func TestHarnessSwitchModel(t *testing.T) {
	t.Run("tracks model selection via modelUseCountTracker", func(t *testing.T) {
		var trackedModelID string
		trackCount := 0

		h, err := New(HarnessConfig{
			ID: "test-harness",
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
			ModelUseCountTracker: func(modelID string) {
				trackedModelID = modelID
				trackCount++
			},
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}

		err = h.SwitchModel("openai/gpt-5.3-codex", "", "")
		if err != nil {
			t.Fatalf("SwitchModel failed: %v", err)
		}

		if trackCount != 1 {
			t.Errorf("expected tracker called 1 time, got %d", trackCount)
		}
		if trackedModelID != "openai/gpt-5.3-codex" {
			t.Errorf("expected tracked modelID = %q, got %q", "openai/gpt-5.3-codex", trackedModelID)
		}
	})

	t.Run("emits model_changed event", func(t *testing.T) {
		h, err := New(HarnessConfig{
			ID: "test-harness",
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}

		var receivedEvent HarnessEvent
		h.Subscribe(func(event HarnessEvent) {
			if event.Type == "model_changed" {
				receivedEvent = event
			}
		})

		err = h.SwitchModel("anthropic/claude-sonnet-4-20250514", "", "")
		if err != nil {
			t.Fatalf("SwitchModel failed: %v", err)
		}

		if receivedEvent.Type != "model_changed" {
			t.Errorf("expected model_changed event, got %q", receivedEvent.Type)
		}
		if receivedEvent.ModelID != "anthropic/claude-sonnet-4-20250514" {
			t.Errorf("expected ModelID = %q, got %q", "anthropic/claude-sonnet-4-20250514", receivedEvent.ModelID)
		}
	})

	t.Run("updates current model ID in state", func(t *testing.T) {
		h, err := New(HarnessConfig{
			ID: "test-harness",
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}

		err = h.SwitchModel("openai/gpt-4o", "", "")
		if err != nil {
			t.Fatalf("SwitchModel failed: %v", err)
		}

		if got := h.GetCurrentModelID(); got != "openai/gpt-4o" {
			t.Errorf("GetCurrentModelID() = %q, want %q", got, "openai/gpt-4o")
		}
	})

	t.Run("does not call tracker when no tracker configured", func(t *testing.T) {
		h, err := New(HarnessConfig{
			ID: "test-harness",
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
			// No ModelUseCountTracker
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}

		// Should not panic
		err = h.SwitchModel("openai/gpt-4o", "", "")
		if err != nil {
			t.Fatalf("SwitchModel failed: %v", err)
		}
	})
}

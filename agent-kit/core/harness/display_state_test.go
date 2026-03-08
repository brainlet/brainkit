// Ported from: packages/core/src/harness/display-state.test.ts
package harness

import (
	"testing"
)

func createTestHarness(t *testing.T) *Harness {
	t.Helper()
	h, err := New(HarnessConfig{
		ID: "test-harness",
		Modes: []HarnessMode{
			{ID: "default", Name: "Default", Default: true},
		},
	})
	if err != nil {
		t.Fatalf("failed to create harness: %v", err)
	}
	return h
}

func TestDefaultDisplayState(t *testing.T) {
	t.Run("returns a fresh display state with correct defaults", func(t *testing.T) {
		ds := DefaultDisplayState()
		if ds.IsRunning {
			t.Error("expected IsRunning to be false")
		}
		if ds.CurrentMessage != nil {
			t.Error("expected CurrentMessage to be nil")
		}
		if ds.TokenUsage.PromptTokens != 0 || ds.TokenUsage.CompletionTokens != 0 || ds.TokenUsage.TotalTokens != 0 {
			t.Errorf("expected zero token usage, got %+v", ds.TokenUsage)
		}
		if len(ds.ActiveTools) != 0 {
			t.Errorf("expected empty ActiveTools, got %d entries", len(ds.ActiveTools))
		}
		if len(ds.ToolInputBuffers) != 0 {
			t.Errorf("expected empty ToolInputBuffers, got %d entries", len(ds.ToolInputBuffers))
		}
		if ds.PendingApproval != nil {
			t.Error("expected PendingApproval to be nil")
		}
		if ds.PendingQuestion != nil {
			t.Error("expected PendingQuestion to be nil")
		}
		if ds.PendingPlanApproval != nil {
			t.Error("expected PendingPlanApproval to be nil")
		}
		if len(ds.ActiveSubagents) != 0 {
			t.Errorf("expected empty ActiveSubagents, got %d entries", len(ds.ActiveSubagents))
		}
		if ds.OMProgress.Status != OMStatusIdle {
			t.Errorf("expected OMProgress.Status = %q, got %q", OMStatusIdle, ds.OMProgress.Status)
		}
		if ds.OMProgress.PendingTokens != 0 {
			t.Errorf("expected OMProgress.PendingTokens = 0, got %d", ds.OMProgress.PendingTokens)
		}
		if ds.OMProgress.Threshold != 30000 {
			t.Errorf("expected OMProgress.Threshold = 30000, got %d", ds.OMProgress.Threshold)
		}
		if len(ds.ModifiedFiles) != 0 {
			t.Errorf("expected empty ModifiedFiles, got %d entries", len(ds.ModifiedFiles))
		}
		if ds.Tasks != nil {
			t.Errorf("expected Tasks to be nil, got %v", ds.Tasks)
		}
		if ds.PreviousTasks != nil {
			t.Errorf("expected PreviousTasks to be nil, got %v", ds.PreviousTasks)
		}
		if ds.BufferingMessages {
			t.Error("expected BufferingMessages to be false")
		}
		if ds.BufferingObservations {
			t.Error("expected BufferingObservations to be false")
		}
	})

	t.Run("returns independent instances", func(t *testing.T) {
		ds1 := DefaultDisplayState()
		ds2 := DefaultDisplayState()
		ds1.Tasks = append(ds1.Tasks, TaskItem{Content: "test", Status: "pending", ActiveForm: "Testing"})
		if len(ds2.Tasks) != 0 {
			t.Errorf("expected ds2.Tasks to remain empty, got %d", len(ds2.Tasks))
		}
	})
}

func TestHarnessGetDisplayState(t *testing.T) {
	t.Run("returns display state with correct initial values", func(t *testing.T) {
		h := createTestHarness(t)
		ds := h.GetDisplayState()
		if ds.IsRunning {
			t.Error("expected IsRunning to be false")
		}
		if ds.CurrentMessage != nil {
			t.Error("expected CurrentMessage to be nil")
		}
		if ds.TokenUsage.PromptTokens != 0 || ds.TokenUsage.CompletionTokens != 0 || ds.TokenUsage.TotalTokens != 0 {
			t.Errorf("expected zero token usage, got %+v", ds.TokenUsage)
		}
		if len(ds.ActiveTools) != 0 {
			t.Errorf("expected empty ActiveTools, got %d entries", len(ds.ActiveTools))
		}
		if ds.PendingApproval != nil {
			t.Error("expected PendingApproval to be nil")
		}
		if ds.PendingQuestion != nil {
			t.Error("expected PendingQuestion to be nil")
		}
		if ds.PendingPlanApproval != nil {
			t.Error("expected PendingPlanApproval to be nil")
		}
		if len(ds.ActiveSubagents) != 0 {
			t.Errorf("expected empty ActiveSubagents, got %d entries", len(ds.ActiveSubagents))
		}
		if len(ds.ModifiedFiles) != 0 {
			t.Errorf("expected empty ModifiedFiles, got %d entries", len(ds.ModifiedFiles))
		}
	})
}

func TestAgentLifecycleDisplayState(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestMessageStreamingDisplayState(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestToolLifecycleDisplayState(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestModifiedFilesTracking(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestInteractivePrompts(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestSubagentLifecycleDisplayState(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestUsageUpdate(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestTaskUpdated(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestOMEventTransitions(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestStateChangedThresholdSyncing(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestResetThreadDisplayState(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestDisplayStateChangedEmission(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

func TestFullLifecycleIntegration(t *testing.T) {
	t.Skip("not yet implemented - requires emit() to update display state from events (display state reducer not ported)")
}

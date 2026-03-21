package harness

import "testing"

func TestDisplayState_SubagentFlow(t *testing.T) {
	ds := NewDisplayState()

	updateDisplayState(ds, HarnessEvent{
		Type: EventSubagentStart, ToolCallID: "tc1", AgentType: "explore", Task: "find auth code", ModelID: "openai/gpt-4o-mini",
	})
	if sa, ok := ds.ActiveSubagents["tc1"]; !ok || sa.AgentType != "explore" {
		t.Fatal("should have active subagent")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventSubagentTextDelta, ToolCallID: "tc1", Text: "Found it "})
	updateDisplayState(ds, HarnessEvent{Type: EventSubagentTextDelta, ToolCallID: "tc1", Text: "in auth.go"})
	if ds.ActiveSubagents["tc1"].TextDelta != "Found it in auth.go" {
		t.Errorf("text = %q", ds.ActiveSubagents["tc1"].TextDelta)
	}

	updateDisplayState(ds, HarnessEvent{
		Type: EventSubagentToolStart, ToolCallID: "tc1", SubToolCallID: "stc1", ToolName: "view", Args: map[string]any{"path": "auth.go"},
	})
	if len(ds.ActiveSubagents["tc1"].ToolCalls) != 1 {
		t.Fatal("should have 1 tool call")
	}

	updateDisplayState(ds, HarnessEvent{
		Type: EventSubagentToolEnd, ToolCallID: "tc1", SubToolCallID: "stc1", ToolName: "view", Result: "file content",
	})
	if ds.ActiveSubagents["tc1"].ToolCalls[0].Status != "completed" {
		t.Error("sub-tool should be completed")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventSubagentEnd, ToolCallID: "tc1", Text: "Auth is in auth.go", Duration: 3000})
	if ds.ActiveSubagents["tc1"].Status != "completed" {
		t.Error("subagent should be completed")
	}
	if ds.ActiveSubagents["tc1"].Duration != 3000 {
		t.Errorf("duration = %d, want 3000", ds.ActiveSubagents["tc1"].Duration)
	}
}

func TestDisplayState_TokenAccumulation(t *testing.T) {
	ds := NewDisplayState()

	updateDisplayState(ds, HarnessEvent{Type: EventUsageUpdate, Usage: &TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}})
	updateDisplayState(ds, HarnessEvent{Type: EventUsageUpdate, Usage: &TokenUsage{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300}})

	if ds.TokenUsage.PromptTokens != 300 {
		t.Errorf("prompt tokens = %d, want 300", ds.TokenUsage.PromptTokens)
	}
	if ds.TokenUsage.CompletionTokens != 150 {
		t.Errorf("completion tokens = %d, want 150", ds.TokenUsage.CompletionTokens)
	}
	if ds.TokenUsage.TotalTokens != 450 {
		t.Errorf("total tokens = %d, want 450", ds.TokenUsage.TotalTokens)
	}
}

func TestDisplayState_OMProgress(t *testing.T) {
	ds := NewDisplayState()

	updateDisplayState(ds, HarnessEvent{Type: EventOMStatus, Status: "idle", ObservationThreshold: 5, ReflectionThreshold: 3})
	if ds.OMProgress == nil || ds.OMProgress.Status != "idle" {
		t.Fatal("OM progress should be set")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventOMObservationStart, MessageCount: 5})
	if !ds.OMProgress.GeneratingObservation {
		t.Error("should be generating observation")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventOMObservationEnd})
	if ds.OMProgress.GeneratingObservation {
		t.Error("should not be generating observation")
	}
	if ds.OMProgress.TotalObservations != 1 {
		t.Errorf("total observations = %d, want 1", ds.OMProgress.TotalObservations)
	}

	updateDisplayState(ds, HarnessEvent{Type: EventOMReflectionStart})
	if !ds.OMProgress.GeneratingReflection {
		t.Error("should be generating reflection")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventOMReflectionEnd})
	if ds.OMProgress.TotalReflections != 1 {
		t.Errorf("total reflections = %d, want 1", ds.OMProgress.TotalReflections)
	}
}

func TestDisplayState_Buffering(t *testing.T) {
	ds := NewDisplayState()

	updateDisplayState(ds, HarnessEvent{Type: EventOMBufferingStart, Target: "messages"})
	if !ds.BufferingMessages {
		t.Error("should be buffering messages")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventOMBufferingEnd, Target: "messages"})
	if ds.BufferingMessages {
		t.Error("should not be buffering messages")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventOMBufferingStart, Target: "observations"})
	if !ds.BufferingObservations {
		t.Error("should be buffering observations")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventOMBufferingFailed, Target: "observations"})
	if ds.BufferingObservations {
		t.Error("should not be buffering observations after failure")
	}
}

func TestDisplayState_ModifiedFilesTracking(t *testing.T) {
	ds := NewDisplayState()

	updateDisplayState(ds, HarnessEvent{Type: EventToolEnd, ToolCallID: "tc1", ToolName: "write_file", Args: map[string]any{"path": "main.go"}})
	if _, ok := ds.ModifiedFiles["main.go"]; !ok {
		t.Fatal("write_file should track")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventToolEnd, ToolCallID: "tc2", ToolName: "string_replace_lsp", Args: map[string]any{"path": "main.go"}})
	if ds.ModifiedFiles["main.go"].Operations != 2 {
		t.Errorf("operations = %d, want 2", ds.ModifiedFiles["main.go"].Operations)
	}

	updateDisplayState(ds, HarnessEvent{Type: EventToolEnd, ToolCallID: "tc3", ToolName: "read_file", Args: map[string]any{"path": "other.go"}})
	if _, ok := ds.ModifiedFiles["other.go"]; ok {
		t.Error("read_file should not track")
	}
}

func TestDisplayState_StateChanged_OMThresholds(t *testing.T) {
	ds := NewDisplayState()
	ds.OMProgress = &OMProgressState{ObservationThreshold: 5, ReflectionThreshold: 3}

	updateDisplayState(ds, HarnessEvent{
		Type:        EventStateChanged,
		ChangedKeys: []string{"observationThreshold"},
		State:       map[string]any{"observationThreshold": float64(10)},
	})
	if ds.OMProgress.ObservationThreshold != 10 {
		t.Errorf("observation threshold = %d, want 10", ds.OMProgress.ObservationThreshold)
	}
}

package brainkit

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Event type serialization
// ---------------------------------------------------------------------------

func TestHarnessEvent_Unmarshal(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect HarnessEventType
	}{
		{"agent_start", `{"type":"agent_start","threadId":"t1","runId":"r1","content":"hello","modeId":"build","modelId":"openai/gpt-4o"}`, EventAgentStart},
		{"agent_end", `{"type":"agent_end","threadId":"t1","text":"done","finishReason":"stop"}`, EventAgentEnd},
		{"message_update", `{"type":"message_update","messageId":"m1","text":"hello "}`, EventMessageUpdate},
		{"tool_start", `{"type":"tool_start","toolCallId":"tc1","toolName":"write_file","args":{"path":"main.go"}}`, EventToolStart},
		{"tool_approval_required", `{"type":"tool_approval_required","toolCallId":"tc1","toolName":"write_file","category":"edit"}`, EventToolApprovalRequired},
		{"ask_question", `{"type":"ask_question","questionId":"q1","question":"What port?","options":["8080","3000"]}`, EventAskQuestion},
		{"task_updated", `{"type":"task_updated","tasks":[{"title":"Build API","status":"pending"}]}`, EventTaskUpdated},
		{"om_status", `{"type":"om_status","status":"idle","observationThreshold":5}`, EventOMStatus},
		{"error", `{"type":"error","error":"something failed","fatal":true}`, EventError},
		{"subagent_start", `{"type":"subagent_start","toolCallId":"tc1","agentType":"explore","task":"find auth"}`, EventSubagentStart},
		{"shell_output", `{"type":"shell_output","toolCallId":"tc1","stream":"stdout","data":"hello\n"}`, EventShellOutput},
		{"plan_approval_required", `{"type":"plan_approval_required","planId":"p1","plan":"Step 1: ..."}`, EventPlanApprovalRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event HarnessEvent
			err := json.Unmarshal([]byte(tt.input), &event)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if event.Type != tt.expect {
				t.Errorf("type = %q, want %q", event.Type, tt.expect)
			}
		})
	}
}

func TestHarnessEvent_UnknownType(t *testing.T) {
	var event HarnessEvent
	err := json.Unmarshal([]byte(`{"type":"future_event","data":"something"}`), &event)
	if err != nil {
		t.Fatalf("should not error on unknown type: %v", err)
	}
	if event.Type != "future_event" {
		t.Errorf("type = %q, want %q", event.Type, "future_event")
	}
}

func TestHarnessEvent_ToolArgs(t *testing.T) {
	var event HarnessEvent
	err := json.Unmarshal([]byte(`{"type":"tool_start","toolCallId":"tc1","toolName":"write_file","args":{"path":"main.go","content":"package main"}}`), &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Args["path"] != "main.go" {
		t.Errorf("args.path = %v", event.Args["path"])
	}
	if event.Args["content"] != "package main" {
		t.Errorf("args.content = %v", event.Args["content"])
	}
}

func TestHarnessEvent_TaskList(t *testing.T) {
	var event HarnessEvent
	err := json.Unmarshal([]byte(`{"type":"task_updated","tasks":[{"title":"Build API","description":"Create endpoints","status":"pending"},{"title":"Test","status":"completed"}]}`), &event)
	if err != nil {
		t.Fatal(err)
	}
	if len(event.Tasks) != 2 {
		t.Fatalf("tasks len = %d, want 2", len(event.Tasks))
	}
	if event.Tasks[0].Title != "Build API" {
		t.Errorf("tasks[0].title = %q", event.Tasks[0].Title)
	}
	if event.Tasks[1].Status != "completed" {
		t.Errorf("tasks[1].status = %q", event.Tasks[1].Status)
	}
}

// ---------------------------------------------------------------------------
// DisplayState
// ---------------------------------------------------------------------------

func TestDisplayState_NewAndClone(t *testing.T) {
	ds := NewDisplayState()
	if ds.ActiveTools == nil || ds.ToolInputBuffers == nil || ds.ActiveSubagents == nil || ds.ModifiedFiles == nil {
		t.Fatal("maps should be initialized")
	}

	ds.IsRunning = true
	ds.ActiveTools["tc1"] = &ActiveToolState{ToolName: "write_file", Status: "running"}
	ds.Tasks = []HarnessTask{{Title: "task1", Status: "pending"}}

	c := ds.clone()
	if !c.IsRunning {
		t.Error("clone should preserve IsRunning")
	}
	if _, ok := c.ActiveTools["tc1"]; !ok {
		t.Error("clone should copy ActiveTools")
	}

	// Mutate original
	ds.ActiveTools["tc1"].Status = "completed"
	if c.ActiveTools["tc1"].Status != "running" {
		t.Error("clone must be a deep copy — mutating original affected clone")
	}
}

func TestDisplayState_CloneNil(t *testing.T) {
	var ds *DisplayState
	c := ds.clone()
	if c != nil {
		t.Error("clone of nil should be nil")
	}
}

func TestTokenUsage_Zero(t *testing.T) {
	var tu TokenUsage
	if tu.TotalTokens != 0 || tu.PromptTokens != 0 || tu.CompletionTokens != 0 {
		t.Error("zero value should have all zeros")
	}
}

// ---------------------------------------------------------------------------
// Config validation
// ---------------------------------------------------------------------------

func TestValidateHarnessConfig_MissingID(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{})
	if err == nil || err.Error() != "harness: ID is required" {
		t.Errorf("expected ID error, got: %v", err)
	}
}

func TestValidateHarnessConfig_NoModes(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{ID: "test"})
	if err == nil {
		t.Error("expected error for no modes")
	}
}

func TestValidateHarnessConfig_NoDefault(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:    "test",
		Modes: []ModeConfig{{ID: "a", AgentName: "agent1"}},
	})
	if err == nil {
		t.Error("expected error for no default mode")
	}
}

func TestValidateHarnessConfig_MultipleDefaults(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID: "test",
		Modes: []ModeConfig{
			{ID: "a", Default: true, AgentName: "agent1"},
			{ID: "b", Default: true, AgentName: "agent1"},
		},
	})
	if err == nil {
		t.Error("expected error for multiple default modes")
	}
}

func TestValidateHarnessConfig_MissingAgentName(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:    "test",
		Modes: []ModeConfig{{ID: "a", Default: true}},
	})
	if err == nil {
		t.Error("expected error for missing AgentName")
	}
}

func TestValidateHarnessConfig_Valid(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:    "test",
		Modes: []ModeConfig{{ID: "build", Name: "Build", Default: true, AgentName: "coder"}},
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateHarnessConfig_SubagentNoTools(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:        "test",
		Modes:     []ModeConfig{{ID: "build", Default: true, AgentName: "coder"}},
		Subagents: []HarnessSubagentConfig{{ID: "explore"}},
	})
	if err == nil {
		t.Error("expected error for subagent with no tools")
	}
}

func TestValidateHarnessConfig_ValidWithSubagents(t *testing.T) {
	err := validateHarnessConfig(HarnessConfig{
		ID:    "test",
		Modes: []ModeConfig{{ID: "build", Default: true, AgentName: "coder"}},
		Subagents: []HarnessSubagentConfig{
			{ID: "explore", AllowedTools: []string{"view", "search"}},
		},
	})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Display state machine
// ---------------------------------------------------------------------------

func TestDisplayState_AgentLifecycle(t *testing.T) {
	ds := NewDisplayState()

	updateDisplayState(ds, HarnessEvent{Type: EventAgentStart})
	if !ds.IsRunning {
		t.Error("should be running after agent_start")
	}

	ds.ActiveTools["tc1"] = &ActiveToolState{ToolName: "test", Status: "running"}
	ds.PendingApproval = &PendingApproval{ToolCallID: "tc1"}
	ds.PendingQuestion = &PendingQuestion{QuestionID: "q1"}
	ds.PendingPlanApproval = &PendingPlanApproval{PlanID: "p1"}
	ds.ActiveSubagents["sa1"] = &ActiveSubagentState{AgentType: "explore"}

	updateDisplayState(ds, HarnessEvent{Type: EventAgentEnd})
	if ds.IsRunning {
		t.Error("should not be running after agent_end")
	}
	if len(ds.ActiveTools) != 0 {
		t.Error("active tools should be cleared")
	}
	if ds.PendingApproval != nil {
		t.Error("pending approval should be cleared")
	}
	if ds.PendingQuestion != nil {
		t.Error("pending question should be cleared")
	}
	if ds.PendingPlanApproval != nil {
		t.Error("pending plan should be cleared")
	}
	if len(ds.ActiveSubagents) != 0 {
		t.Error("active subagents should be cleared")
	}
	if ds.CurrentMessage != nil {
		t.Error("current message should be cleared")
	}
}

func TestDisplayState_MessageFlow(t *testing.T) {
	ds := NewDisplayState()

	updateDisplayState(ds, HarnessEvent{Type: EventMessageStart, MessageID: "m1"})
	if ds.CurrentMessage == nil || ds.CurrentMessage.ID != "m1" {
		t.Fatal("current message should be set")
	}
	if ds.CurrentMessage.Role != "assistant" {
		t.Errorf("role = %q, want assistant", ds.CurrentMessage.Role)
	}

	updateDisplayState(ds, HarnessEvent{Type: EventMessageUpdate, Text: "hello "})
	updateDisplayState(ds, HarnessEvent{Type: EventMessageUpdate, Text: "world"})
	if ds.CurrentMessage.Text != "hello world" {
		t.Errorf("text = %q, want %q", ds.CurrentMessage.Text, "hello world")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventMessageEnd, Text: "hello world!"})
	if ds.CurrentMessage.Text != "hello world!" {
		t.Errorf("final text = %q, want %q", ds.CurrentMessage.Text, "hello world!")
	}
}

func TestDisplayState_ToolFlow(t *testing.T) {
	ds := NewDisplayState()

	updateDisplayState(ds, HarnessEvent{Type: EventToolInputStart, ToolCallID: "tc1", ToolName: "write_file"})
	if _, ok := ds.ToolInputBuffers["tc1"]; !ok {
		t.Fatal("should have input buffer")
	}

	updateDisplayState(ds, HarnessEvent{Type: EventToolInputDelta, ToolCallID: "tc1", Delta: `{"path"`})
	updateDisplayState(ds, HarnessEvent{Type: EventToolInputDelta, ToolCallID: "tc1", Delta: `:"main.go"}`})
	if ds.ToolInputBuffers["tc1"].Text != `{"path":"main.go"}` {
		t.Errorf("buffer = %q", ds.ToolInputBuffers["tc1"].Text)
	}

	updateDisplayState(ds, HarnessEvent{Type: EventToolInputEnd, ToolCallID: "tc1", ToolName: "write_file", Args: map[string]any{"path": "main.go"}})
	if _, ok := ds.ToolInputBuffers["tc1"]; ok {
		t.Error("input buffer should be cleared after input_end")
	}
	if _, ok := ds.ActiveTools["tc1"]; !ok {
		t.Fatal("should have active tool after input_end")
	}
	if ds.ActiveTools["tc1"].Status != "running" {
		t.Errorf("status = %q, want running", ds.ActiveTools["tc1"].Status)
	}

	updateDisplayState(ds, HarnessEvent{Type: EventToolEnd, ToolCallID: "tc1", ToolName: "write_file", Result: "ok", Duration: 150})
	if ds.ActiveTools["tc1"].Status != "completed" {
		t.Errorf("status = %q, want completed", ds.ActiveTools["tc1"].Status)
	}
	if ds.ActiveTools["tc1"].Duration != 150 {
		t.Errorf("duration = %d, want 150", ds.ActiveTools["tc1"].Duration)
	}
	if _, ok := ds.ModifiedFiles["main.go"]; !ok {
		t.Error("write_file should track modified file")
	}
}

func TestDisplayState_ToolError(t *testing.T) {
	ds := NewDisplayState()
	updateDisplayState(ds, HarnessEvent{Type: EventToolStart, ToolCallID: "tc1", ToolName: "execute_command", Args: map[string]any{"command": "ls"}})
	updateDisplayState(ds, HarnessEvent{Type: EventToolEnd, ToolCallID: "tc1", ToolName: "execute_command", IsError: true, Result: "permission denied"})
	if ds.ActiveTools["tc1"].Status != "error" {
		t.Errorf("status = %q, want error", ds.ActiveTools["tc1"].Status)
	}
	if !ds.ActiveTools["tc1"].IsError {
		t.Error("IsError should be true")
	}
}

func TestDisplayState_ShellOutput(t *testing.T) {
	ds := NewDisplayState()
	updateDisplayState(ds, HarnessEvent{Type: EventToolStart, ToolCallID: "tc1", ToolName: "execute_command"})
	updateDisplayState(ds, HarnessEvent{Type: EventShellOutput, ToolCallID: "tc1", Stream: "stdout", Data: "line1\n"})
	updateDisplayState(ds, HarnessEvent{Type: EventShellOutput, ToolCallID: "tc1", Stream: "stderr", Data: "warn\n"})
	if len(ds.ActiveTools["tc1"].ShellOutput) != 2 {
		t.Fatalf("shell output len = %d, want 2", len(ds.ActiveTools["tc1"].ShellOutput))
	}
	if ds.ActiveTools["tc1"].ShellOutput[0].Stream != "stdout" {
		t.Error("first chunk should be stdout")
	}
	if ds.ActiveTools["tc1"].ShellOutput[1].Stream != "stderr" {
		t.Error("second chunk should be stderr")
	}
}

func TestDisplayState_ToolApproval(t *testing.T) {
	ds := NewDisplayState()
	updateDisplayState(ds, HarnessEvent{
		Type: EventToolApprovalRequired, ToolCallID: "tc1", ToolName: "execute_command",
		Args: map[string]any{"command": "rm -rf /"}, Category: "execute",
	})
	if ds.PendingApproval == nil {
		t.Fatal("should have pending approval")
	}
	if ds.PendingApproval.Category != "execute" {
		t.Errorf("category = %q, want execute", ds.PendingApproval.Category)
	}
	if ds.PendingApproval.ToolName != "execute_command" {
		t.Errorf("toolName = %q", ds.PendingApproval.ToolName)
	}
}

func TestDisplayState_AskQuestion(t *testing.T) {
	ds := NewDisplayState()
	updateDisplayState(ds, HarnessEvent{
		Type: EventAskQuestion, QuestionID: "q1", Question: "What port?", Options: []string{"8080", "3000"},
	})
	if ds.PendingQuestion == nil {
		t.Fatal("should have pending question")
	}
	if ds.PendingQuestion.Question != "What port?" {
		t.Errorf("question = %q", ds.PendingQuestion.Question)
	}
	if len(ds.PendingQuestion.Options) != 2 {
		t.Errorf("options = %v", ds.PendingQuestion.Options)
	}
}

func TestDisplayState_PlanApproval(t *testing.T) {
	ds := NewDisplayState()
	updateDisplayState(ds, HarnessEvent{Type: EventPlanApprovalRequired, PlanID: "p1", Plan: "Step 1: Build"})
	if ds.PendingPlanApproval == nil {
		t.Fatal("should have pending plan")
	}
	updateDisplayState(ds, HarnessEvent{Type: EventPlanApproved, PlanID: "p1"})
	if ds.PendingPlanApproval != nil {
		t.Error("plan should be cleared after approval")
	}
}

func TestDisplayState_TaskUpdate(t *testing.T) {
	ds := NewDisplayState()
	ds.Tasks = []HarnessTask{{Title: "old", Status: "completed"}}

	updateDisplayState(ds, HarnessEvent{
		Type:  EventTaskUpdated,
		Tasks: []HarnessTask{{Title: "new", Status: "pending"}},
	})
	if len(ds.PreviousTasks) != 1 || ds.PreviousTasks[0].Title != "old" {
		t.Error("previous tasks should contain old tasks")
	}
	if len(ds.Tasks) != 1 || ds.Tasks[0].Title != "new" {
		t.Error("tasks should be replaced")
	}
}

func TestDisplayState_ThreadReset(t *testing.T) {
	ds := NewDisplayState()
	ds.ActiveTools["tc1"] = &ActiveToolState{ToolName: "test"}
	ds.Tasks = []HarnessTask{{Title: "task1"}}
	ds.OMProgress = &OMProgressState{Status: "observing"}
	ds.PendingApproval = &PendingApproval{ToolCallID: "tc1"}
	ds.BufferingMessages = true

	updateDisplayState(ds, HarnessEvent{Type: EventThreadChanged, ThreadID: "t2"})

	if len(ds.ActiveTools) != 0 {
		t.Error("active tools should be cleared")
	}
	if ds.Tasks != nil {
		t.Error("tasks should be cleared")
	}
	if ds.OMProgress != nil {
		t.Error("OM progress should be cleared")
	}
	if ds.PendingApproval != nil {
		t.Error("pending approval should be cleared")
	}
	if ds.BufferingMessages {
		t.Error("buffering should be cleared")
	}
}

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

	// write_file tracks
	updateDisplayState(ds, HarnessEvent{Type: EventToolEnd, ToolCallID: "tc1", ToolName: "write_file", Args: map[string]any{"path": "main.go"}})
	if _, ok := ds.ModifiedFiles["main.go"]; !ok {
		t.Fatal("write_file should track")
	}

	// string_replace_lsp tracks
	updateDisplayState(ds, HarnessEvent{Type: EventToolEnd, ToolCallID: "tc2", ToolName: "string_replace_lsp", Args: map[string]any{"path": "main.go"}})
	if ds.ModifiedFiles["main.go"].Operations != 2 {
		t.Errorf("operations = %d, want 2", ds.ModifiedFiles["main.go"].Operations)
	}

	// read_file does NOT track
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

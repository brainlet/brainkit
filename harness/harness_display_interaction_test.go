package harness

import "testing"

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

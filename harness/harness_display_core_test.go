package harness

import "testing"

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

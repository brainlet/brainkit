package messages

import "encoding/json"

// ── Requests ──

type WorkflowRunMsg struct {
	Name  string `json:"name"`
	Input any    `json:"input"`
}

func (WorkflowRunMsg) BusTopic() string { return "workflows.run" }

type WorkflowResumeMsg struct {
	RunID  string `json:"runId"`
	StepID string `json:"stepId,omitempty"`
	Data   any    `json:"data"`
}

func (WorkflowResumeMsg) BusTopic() string { return "workflows.resume" }

type WorkflowCancelMsg struct {
	RunID string `json:"runId"`
}

func (WorkflowCancelMsg) BusTopic() string { return "workflows.cancel" }

type WorkflowStatusMsg struct {
	RunID string `json:"runId"`
}

func (WorkflowStatusMsg) BusTopic() string { return "workflows.status" }

// ── Responses ──

type WorkflowRunResp struct {
	ResultMeta
	Result json.RawMessage `json:"result"`
}

func (WorkflowRunResp) BusTopic() string { return "workflows.run.result" }

type WorkflowResumeResp struct {
	ResultMeta
	Result json.RawMessage `json:"result"`
}

func (WorkflowResumeResp) BusTopic() string { return "workflows.resume.result" }

type WorkflowCancelResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (WorkflowCancelResp) BusTopic() string { return "workflows.cancel.result" }

type WorkflowStatusResp struct {
	ResultMeta
	Status string `json:"status"`
	Step   string `json:"step"`
}

func (WorkflowStatusResp) BusTopic() string { return "workflows.status.result" }

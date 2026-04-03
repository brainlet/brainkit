package messages

import "encoding/json"

// ── Workflow Lifecycle ──

type WorkflowStartMsg struct {
	Name      string          `json:"name"`
	InputData json.RawMessage `json:"inputData,omitempty"`
}

func (WorkflowStartMsg) BusTopic() string { return "workflow.start" }

type WorkflowStartResp struct {
	ResultMeta
	RunID  string          `json:"runId"`
	Status string          `json:"status"`
	Steps  json.RawMessage `json:"steps,omitempty"`
}

type WorkflowStartAsyncMsg struct {
	Name      string          `json:"name"`
	InputData json.RawMessage `json:"inputData,omitempty"`
}

func (WorkflowStartAsyncMsg) BusTopic() string { return "workflow.startAsync" }

type WorkflowStartAsyncResp struct {
	ResultMeta
	RunID string `json:"runId"`
}

type WorkflowStatusMsg struct {
	Name  string `json:"name"`
	RunID string `json:"runId"`
}

func (WorkflowStatusMsg) BusTopic() string { return "workflow.status" }

type WorkflowStatusResp struct {
	ResultMeta
	RunID  string          `json:"runId"`
	Status string          `json:"status"`
	Steps  json.RawMessage `json:"steps,omitempty"`
}

type WorkflowResumeMsg struct {
	Name       string          `json:"name"`
	RunID      string          `json:"runId"`
	Step       string          `json:"step,omitempty"`
	ResumeData json.RawMessage `json:"resumeData,omitempty"`
}

func (WorkflowResumeMsg) BusTopic() string { return "workflow.resume" }

type WorkflowResumeResp struct {
	ResultMeta
	Status string          `json:"status"`
	Steps  json.RawMessage `json:"steps,omitempty"`
}

type WorkflowCancelMsg struct {
	Name  string `json:"name"`
	RunID string `json:"runId"`
}

func (WorkflowCancelMsg) BusTopic() string { return "workflow.cancel" }

type WorkflowCancelResp struct {
	ResultMeta
	Cancelled bool `json:"cancelled"`
}

type WorkflowListMsg struct{}

func (WorkflowListMsg) BusTopic() string { return "workflow.list" }

type WorkflowListResp struct {
	ResultMeta
	Workflows []WorkflowInfo `json:"workflows"`
}

type WorkflowInfo struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	HasInput  bool   `json:"hasInput"`
	HasOutput bool   `json:"hasOutput"`
}

// ── Workflow Runs (query + restart) ──

type WorkflowRunsMsg struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

func (WorkflowRunsMsg) BusTopic() string { return "workflow.runs" }

type WorkflowRunsResp struct {
	ResultMeta
	Runs  json.RawMessage `json:"runs"`
	Total int             `json:"total"`
}

type WorkflowRestartMsg struct {
	Name  string `json:"name"`
	RunID string `json:"runId"`
}

func (WorkflowRestartMsg) BusTopic() string { return "workflow.restart" }

type WorkflowRestartResp struct {
	ResultMeta
	Status string          `json:"status"`
	Steps  json.RawMessage `json:"steps,omitempty"`
}

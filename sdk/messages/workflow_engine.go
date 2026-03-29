package messages

import "encoding/json"

// ── Workflow Execution ──

type WorkflowRunMsg struct {
	WorkflowID string          `json:"workflowId"`
	Input      json.RawMessage `json:"input,omitempty"`
}

func (WorkflowRunMsg) BusTopic() string { return "workflow.run" }

type WorkflowRunResp struct {
	ResultMeta
	RunID  string `json:"runId"`
	Status string `json:"status"`
}

type WorkflowStatusMsg struct {
	RunID string `json:"runId"`
}

func (WorkflowStatusMsg) BusTopic() string { return "workflow.status" }

type WorkflowStatusResp struct {
	ResultMeta
	RunID       string `json:"runId"`
	WorkflowID  string `json:"workflowId"`
	Status      string `json:"status"`
	CurrentStep int    `json:"currentStep"`
	StartedAt   string `json:"startedAt"`
	Error       string `json:"error,omitempty"`
	Output      string `json:"output,omitempty"`
}

type WorkflowCancelMsg struct {
	RunID string `json:"runId"`
}

func (WorkflowCancelMsg) BusTopic() string { return "workflow.cancel" }

type WorkflowCancelResp struct {
	ResultMeta
	Cancelled bool `json:"cancelled"`
}

type WorkflowListMsg struct {
	WorkflowID string `json:"workflowId,omitempty"`
}

func (WorkflowListMsg) BusTopic() string { return "workflow.list" }

type WorkflowListResp struct {
	ResultMeta
	Runs []WorkflowRunInfo `json:"runs"`
}

type WorkflowRunInfo struct {
	RunID       string `json:"runId"`
	WorkflowID  string `json:"workflowId"`
	Status      string `json:"status"`
	CurrentStep int    `json:"currentStep"`
	StartedAt   string `json:"startedAt"`
	Error       string `json:"error,omitempty"`
}

type WorkflowHistoryMsg struct {
	RunID string `json:"runId"`
}

func (WorkflowHistoryMsg) BusTopic() string { return "workflow.history" }

type WorkflowHistoryResp struct {
	ResultMeta
	Entries json.RawMessage `json:"entries"` // []JournalEntry as JSON
}

// ── Automations (workflow + plugin deps + admin code + manifest) ──

type AutomationDeployMsg struct {
	Path           string          `json:"path,omitempty"`
	Manifest       json.RawMessage `json:"manifest,omitempty"`
	WorkflowSource string          `json:"workflowSource,omitempty"`
	AdminSource    string          `json:"adminSource,omitempty"`
}

func (AutomationDeployMsg) BusTopic() string { return "automation.deploy" }

type AutomationDeployResp struct {
	ResultMeta
	Deployed   bool   `json:"deployed"`
	WorkflowID string `json:"workflowId"`
}

type AutomationTeardownMsg struct {
	Name string `json:"name"`
}

func (AutomationTeardownMsg) BusTopic() string { return "automation.teardown" }

type AutomationTeardownResp struct {
	ResultMeta
	Removed bool `json:"removed"`
}

type AutomationListMsg struct{}

func (AutomationListMsg) BusTopic() string { return "automation.list" }

type AutomationListResp struct {
	ResultMeta
	Automations []AutomationInfo `json:"automations"`
}

type AutomationInfo struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	ActiveRuns int    `json:"activeRuns"`
}

type AutomationInfoMsg struct {
	Name string `json:"name"`
}

func (AutomationInfoMsg) BusTopic() string { return "automation.info" }

type AutomationInfoResp struct {
	ResultMeta
	Manifest json.RawMessage `json:"manifest"`
	Status   string          `json:"status"`
}

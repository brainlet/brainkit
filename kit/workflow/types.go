package workflow

import (
	"encoding/json"
	"time"
)

// WorkflowDef describes a registered workflow.
type WorkflowDef struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Binary      []byte          `json:"-"`           // compiled WASM
	EntryFunc   string          `json:"entryFunc"`   // exported function name
	Triggers    []TriggerDef    `json:"triggers"`
	Timeout     time.Duration   `json:"timeout"`     // total workflow timeout
	MaxRetries  int             `json:"maxRetries"`
}

// TriggerDef describes what starts a workflow.
type TriggerDef struct {
	Type       string `json:"type"`       // "bus", "schedule", "manual"
	Topic      string `json:"topic"`
	Expression string `json:"expression"` // for schedule: "every 5m"
}

// RunStatus is the current state of a workflow run.
type RunStatus string

const (
	RunRunning   RunStatus = "running"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
	RunSuspended RunStatus = "suspended"
	RunReplaying RunStatus = "replaying"
	RunCancelled RunStatus = "cancelled"
)

// WorkflowRun tracks one execution of a workflow.
type WorkflowRun struct {
	WorkflowID      string          `json:"workflowId"`
	RunID           string          `json:"runId"`
	Status          RunStatus       `json:"status"`
	Input           json.RawMessage `json:"input"`
	Output          string          `json:"output,omitempty"`
	CurrentStep     int             `json:"currentStep"`
	StartedAt       time.Time       `json:"startedAt"`
	CompletedAt     *time.Time      `json:"completedAt,omitempty"`
	SuspendedEvent  string          `json:"suspendedEvent,omitempty"`  // topic waiting for
	SuspendedTimeout int            `json:"suspendedTimeout,omitempty"` // timeout seconds
	Error           string          `json:"error,omitempty"`
	RetryCount      int             `json:"retryCount"`
}

// JournalEntry records one step's execution.
type JournalEntry struct {
	StepName    string          `json:"stepName"`
	StepIndex   int             `json:"stepIndex"`
	Status      string          `json:"status"` // "completed", "failed", "pending", "suspended"
	Calls       []HostCallRecord `json:"calls"`
	StartedAt   time.Time       `json:"startedAt"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
	Error       string          `json:"error,omitempty"`
}

// HostCallRecord records one host function call within a step.
type HostCallRecord struct {
	Function string          `json:"function"` // "ai.generate", "telegram.send"
	Args     json.RawMessage `json:"args"`
	Result   json.RawMessage `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
	Duration time.Duration   `json:"duration"`
}

// HostFunctionDef describes a host function available to workflows.
type HostFunctionDef struct {
	Module      string      `json:"module"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Params      []HostParam `json:"params"`
	Returns     string      `json:"returns"` // "string", "i32", "i64", "void"
	PluginName  string      `json:"pluginName,omitempty"`
	PluginTopic string      `json:"pluginTopic,omitempty"`
}

// HostParam describes a parameter of a host function.
type HostParam struct {
	Name string `json:"name"`
	Type string `json:"type"` // "string", "i32", "i64", "f64"
}

// RunInfo is the public view of a workflow run.
type RunInfo struct {
	RunID       string    `json:"runId"`
	WorkflowID  string    `json:"workflowId"`
	Status      string    `json:"status"`
	CurrentStep int       `json:"currentStep"`
	StartedAt   string    `json:"startedAt"`
	Error       string    `json:"error,omitempty"`
}

package harness

import "encoding/json"

// HarnessEventType identifies the kind of harness event.
type HarnessEventType string

const (
	// Agent lifecycle
	EventAgentStart HarnessEventType = "agent_start"
	EventAgentEnd   HarnessEventType = "agent_end"

	// Mode & Model
	EventModeChanged  HarnessEventType = "mode_changed"
	EventModelChanged HarnessEventType = "model_changed"

	// Thread
	EventThreadChanged HarnessEventType = "thread_changed"
	EventThreadCreated HarnessEventType = "thread_created"
	EventThreadDeleted HarnessEventType = "thread_deleted"

	// Message streaming
	EventMessageStart  HarnessEventType = "message_start"
	EventMessageUpdate HarnessEventType = "message_update"
	EventMessageEnd    HarnessEventType = "message_end"

	// Tool execution
	EventToolStart            HarnessEventType = "tool_start"
	EventToolApprovalRequired HarnessEventType = "tool_approval_required"
	EventToolInputStart       HarnessEventType = "tool_input_start"
	EventToolInputDelta       HarnessEventType = "tool_input_delta"
	EventToolInputEnd         HarnessEventType = "tool_input_end"
	EventToolUpdate           HarnessEventType = "tool_update"
	EventToolEnd              HarnessEventType = "tool_end"
	EventShellOutput          HarnessEventType = "shell_output"

	// Interactive
	EventAskQuestion          HarnessEventType = "ask_question"
	EventPlanApprovalRequired HarnessEventType = "plan_approval_required"
	EventPlanApproved         HarnessEventType = "plan_approved"

	// Subagent
	EventSubagentStart        HarnessEventType = "subagent_start"
	EventSubagentTextDelta    HarnessEventType = "subagent_text_delta"
	EventSubagentToolStart    HarnessEventType = "subagent_tool_start"
	EventSubagentToolEnd      HarnessEventType = "subagent_tool_end"
	EventSubagentEnd          HarnessEventType = "subagent_end"
	EventSubagentModelChanged HarnessEventType = "subagent_model_changed"

	// Observational Memory
	EventOMStatus            HarnessEventType = "om_status"
	EventOMObservationStart  HarnessEventType = "om_observation_start"
	EventOMObservationEnd    HarnessEventType = "om_observation_end"
	EventOMObservationFailed HarnessEventType = "om_observation_failed"
	EventOMReflectionStart   HarnessEventType = "om_reflection_start"
	EventOMReflectionEnd     HarnessEventType = "om_reflection_end"
	EventOMReflectionFailed  HarnessEventType = "om_reflection_failed"
	EventOMBufferingStart    HarnessEventType = "om_buffering_start"
	EventOMBufferingEnd      HarnessEventType = "om_buffering_end"
	EventOMBufferingFailed   HarnessEventType = "om_buffering_failed"
	EventOMActivation        HarnessEventType = "om_activation"
	EventOMModelChanged      HarnessEventType = "om_model_changed"

	// Workspace
	EventWorkspaceStatusChanged HarnessEventType = "workspace_status_changed"
	EventWorkspaceReady         HarnessEventType = "workspace_ready"
	EventWorkspaceError         HarnessEventType = "workspace_error"

	// State & System
	EventStateChanged        HarnessEventType = "state_changed"
	EventDisplayStateChanged HarnessEventType = "display_state_changed"
	EventTaskUpdated         HarnessEventType = "task_updated"
	EventUsageUpdate         HarnessEventType = "usage_update"
	EventFollowUpQueued      HarnessEventType = "follow_up_queued"
	EventInfo                HarnessEventType = "info"
	EventError               HarnessEventType = "error"
)

// HarnessEvent is the Go representation of every Harness event.
// Fields are populated based on Type. Raw preserves original JSON for passthrough.
type HarnessEvent struct {
	Type HarnessEventType `json:"type"`
	Raw  json.RawMessage  `json:"-"`

	// Common identifiers
	ThreadID   string `json:"threadId,omitempty"`
	RunID      string `json:"runId,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	MessageID  string `json:"messageId,omitempty"`
	QuestionID string `json:"questionId,omitempty"`
	PlanID     string `json:"planId,omitempty"`
	ModeID     string `json:"modeId,omitempty"`
	ModelID    string `json:"modelId,omitempty"`
	AgentType  string `json:"agentType,omitempty"`

	// Text content
	Text         string `json:"text,omitempty"`
	Content      string `json:"content,omitempty"`
	Reasoning    string `json:"reasoning,omitempty"`
	Question     string `json:"question,omitempty"`
	Plan         string `json:"plan,omitempty"`
	ErrorMessage string `json:"error,omitempty"`
	Message      any    `json:"message,omitempty"`
	Delta        string `json:"delta,omitempty"`
	Title        string `json:"title,omitempty"`

	// Structured data
	Args        map[string]any `json:"args,omitempty"`
	Result      any            `json:"result,omitempty"`
	Usage       *TokenUsage    `json:"usage,omitempty"`
	State       map[string]any `json:"state,omitempty"`
	Tasks       []HarnessTask  `json:"tasks,omitempty"`
	Options     []string       `json:"options,omitempty"`
	ChangedKeys []string       `json:"changedKeys,omitempty"`

	// Flags
	IsError      bool   `json:"isError,omitempty"`
	Fatal        bool   `json:"fatal,omitempty"`
	Default      bool   `json:"default,omitempty"`
	Role         string `json:"role,omitempty"`
	ModeName     string `json:"modeName,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Category     string `json:"category,omitempty"`
	FinishReason string `json:"finishReason,omitempty"`
	Stream       string `json:"stream,omitempty"`
	Data         string `json:"data,omitempty"`
	Status       string `json:"status,omitempty"`
	Target       string `json:"target,omitempty"`

	// Numeric
	Duration    int `json:"duration,omitempty"`
	QueueLength int `json:"queueLength,omitempty"`

	// OM-specific
	MessageCount                 int      `json:"messageCount,omitempty"`
	ObservationCount             int      `json:"observationCount,omitempty"`
	Observations                 []string `json:"observations,omitempty"`
	Reflections                  []string `json:"reflections,omitempty"`
	MessagesSinceLastObservation int      `json:"messagesSinceLastObservation,omitempty"`
	MessagesSinceLastReflection  int      `json:"messagesSinceLastReflection,omitempty"`
	ObservationThreshold         int      `json:"observationThreshold,omitempty"`
	ReflectionThreshold          int      `json:"reflectionThreshold,omitempty"`
	TotalObservations            int      `json:"totalObservations,omitempty"`
	TotalReflections             int      `json:"totalReflections,omitempty"`

	// Subagent-specific
	SubToolCallID string `json:"subToolCallId,omitempty"`
	Task          string `json:"task,omitempty"`

	// Display state (only on display_state_changed)
	DisplayStatePayload *DisplayState `json:"displayState,omitempty"`
}

// TokenUsage tracks LLM token consumption.
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

package harness

import (
	"encoding/json"
	"time"
)

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

// ---------------------------------------------------------------------------
// HarnessEvent — flat struct for all 41 event types
// ---------------------------------------------------------------------------

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
	Message      any    `json:"message,omitempty"` // string for info/error, object for message_* events
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
	Stream       string `json:"stream,omitempty"` // stdout|stderr
	Data         string `json:"data,omitempty"`   // shell output data
	Status       string `json:"status,omitempty"` // workspace/OM status
	Target       string `json:"target,omitempty"` // buffering target

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

// ---------------------------------------------------------------------------
// Shared data types
// ---------------------------------------------------------------------------

// TokenUsage tracks LLM token consumption.
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// HarnessTask represents a task tracked by the Harness.
type HarnessTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` // "pending" | "in_progress" | "completed"
}

// Mode represents a Harness mode (e.g., build, plan, fast).
type Mode struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Default        bool   `json:"default"`
	DefaultModelID string `json:"defaultModelId"`
	Color          string `json:"color"`
}

// HarnessThread represents a conversation thread.
type HarnessThread struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	CreatedAt  string         `json:"createdAt"`
	UpdatedAt  string         `json:"updatedAt"`
	ResourceID string         `json:"resourceId"`
	Metadata   map[string]any `json:"metadata"`
}

// HarnessMessage represents a message in a thread.
type HarnessMessage struct {
	ID        string         `json:"id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	CreatedAt string         `json:"createdAt"`
	ThreadID  string         `json:"threadId"`
	Metadata  map[string]any `json:"metadata"`
}

// AvailableModel describes a model the Harness can use.
type AvailableModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	IsAuth   bool   `json:"isAuth"`
	UseCount int    `json:"useCount"`
	IsCustom bool   `json:"isCustom"`
}

// HarnessSession describes the current Harness session.
type HarnessSession struct {
	CurrentThreadID string          `json:"currentThreadId"`
	CurrentModeID   string          `json:"currentModeId"`
	Threads         []HarnessThread `json:"threads"`
}

// PermissionRules tracks persistent permission policies.
type PermissionRules struct {
	Categories map[string]string `json:"categories"` // category -> policy
	Tools      map[string]string `json:"tools"`      // toolName -> policy
}

// SessionGrants tracks temporary session-level permission grants.
type SessionGrants struct {
	Categories []string `json:"categories"`
	Tools      []string `json:"tools"`
}

// ToolApprovalDecision is the user's response to a tool approval request.
type ToolApprovalDecision string

const (
	ToolApprove             ToolApprovalDecision = "approve"
	ToolDecline             ToolApprovalDecision = "decline"
	ToolAlwaysAllowCategory ToolApprovalDecision = "always_allow_category"
)

// PlanResponse is the user's response to a plan approval request.
type PlanResponse struct {
	Action   string `json:"action"`   // "approve" | "reject"
	Feedback string `json:"feedback"` // optional feedback on rejection
}

// FileAttachment represents a file attached to a message.
type FileAttachment struct {
	URI      string `json:"uri,omitempty"`
	Base64   string `json:"base64,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Name     string `json:"name,omitempty"`
}

// ---------------------------------------------------------------------------
// DisplayState — canonical UI state
// ---------------------------------------------------------------------------

// DisplayState is the canonical UI state, updated on every event.
type DisplayState struct {
	IsRunning             bool                              `json:"isRunning"`
	CurrentMessage        *CurrentMessage                   `json:"currentMessage"`
	TokenUsage            TokenUsage                        `json:"tokenUsage"`
	ActiveTools           map[string]*ActiveToolState        `json:"activeTools"`
	ToolInputBuffers      map[string]*ToolInputBuffer        `json:"toolInputBuffers"`
	PendingApproval       *PendingApproval                   `json:"pendingApproval"`
	PendingQuestion       *PendingQuestion                   `json:"pendingQuestion"`
	PendingPlanApproval   *PendingPlanApproval               `json:"pendingPlanApproval"`
	ActiveSubagents       map[string]*ActiveSubagentState     `json:"activeSubagents"`
	OMProgress            *OMProgressState                   `json:"omProgress"`
	BufferingMessages     bool                               `json:"bufferingMessages"`
	BufferingObservations bool                               `json:"bufferingObservations"`
	ModifiedFiles         map[string]*ModifiedFileState      `json:"modifiedFiles"`
	Tasks                 []HarnessTask                      `json:"tasks"`
	PreviousTasks         []HarnessTask                      `json:"previousTasks"`
}

// NewDisplayState creates a fresh display state with initialized maps.
func NewDisplayState() *DisplayState {
	return &DisplayState{
		ActiveTools:      make(map[string]*ActiveToolState),
		ToolInputBuffers: make(map[string]*ToolInputBuffer),
		ActiveSubagents:  make(map[string]*ActiveSubagentState),
		ModifiedFiles:    make(map[string]*ModifiedFileState),
	}
}

type CurrentMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Text      string `json:"text"`
	Reasoning string `json:"reasoning"`
}

type ActiveToolState struct {
	ToolName    string         `json:"toolName"`
	Args        map[string]any `json:"args"`
	Status      string         `json:"status"` // "running" | "completed" | "error"
	Result      any            `json:"result"`
	IsError     bool           `json:"isError"`
	StartTime   time.Time      `json:"startTime"`
	Duration    int            `json:"duration"`
	ShellOutput []ShellChunk   `json:"shellOutput"`
}

type ShellChunk struct {
	Stream string `json:"stream"` // "stdout" | "stderr"
	Data   string `json:"data"`
}

type ToolInputBuffer struct {
	ToolName string `json:"toolName"`
	Text     string `json:"text"`
}

type PendingApproval struct {
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Args       map[string]any `json:"args"`
	Category   string         `json:"category"`
}

type PendingQuestion struct {
	QuestionID string   `json:"questionId"`
	Question   string   `json:"question"`
	Options    []string `json:"options"`
}

type PendingPlanApproval struct {
	PlanID string `json:"planId"`
	Plan   string `json:"plan"`
	Title  string `json:"title"`
}

type ActiveSubagentState struct {
	AgentType string             `json:"agentType"`
	Task      string             `json:"task"`
	ModelID   string             `json:"modelId"`
	Status    string             `json:"status"`
	TextDelta string             `json:"textDelta"`
	ToolCalls []SubagentToolCall `json:"toolCalls"`
	Result    string             `json:"result"`
	Duration  int                `json:"duration"`
	IsError   bool               `json:"isError"`
}

type SubagentToolCall struct {
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Args       map[string]any `json:"args"`
	Result     any            `json:"result"`
	IsError    bool           `json:"isError"`
	Status     string         `json:"status"`
}

type OMProgressState struct {
	Status                       string `json:"status"`
	MessagesSinceLastObservation int    `json:"messagesSinceLastObservation"`
	MessagesSinceLastReflection  int    `json:"messagesSinceLastReflection"`
	ObservationThreshold         int    `json:"observationThreshold"`
	ReflectionThreshold          int    `json:"reflectionThreshold"`
	TotalObservations            int    `json:"totalObservations"`
	TotalReflections             int    `json:"totalReflections"`
	BufferedCount                int    `json:"bufferedCount"`
	GeneratingObservation        bool   `json:"generatingObservation"`
	GeneratingReflection         bool   `json:"generatingReflection"`
	CurrentCycle                 string `json:"currentCycle"`
}

type ModifiedFileState struct {
	Path          string    `json:"path"`
	Operations    int       `json:"operations"`
	FirstModified time.Time `json:"firstModified"`
}

// clone returns a deep copy of the DisplayState.
func (ds *DisplayState) clone() *DisplayState {
	if ds == nil {
		return nil
	}
	c := *ds
	c.ActiveTools = make(map[string]*ActiveToolState, len(ds.ActiveTools))
	for k, v := range ds.ActiveTools {
		vv := *v
		vv.ShellOutput = append([]ShellChunk(nil), v.ShellOutput...)
		c.ActiveTools[k] = &vv
	}
	c.ToolInputBuffers = make(map[string]*ToolInputBuffer, len(ds.ToolInputBuffers))
	for k, v := range ds.ToolInputBuffers {
		vv := *v
		c.ToolInputBuffers[k] = &vv
	}
	c.ActiveSubagents = make(map[string]*ActiveSubagentState, len(ds.ActiveSubagents))
	for k, v := range ds.ActiveSubagents {
		vv := *v
		vv.ToolCalls = append([]SubagentToolCall(nil), v.ToolCalls...)
		c.ActiveSubagents[k] = &vv
	}
	c.ModifiedFiles = make(map[string]*ModifiedFileState, len(ds.ModifiedFiles))
	for k, v := range ds.ModifiedFiles {
		vv := *v
		c.ModifiedFiles[k] = &vv
	}
	c.Tasks = append([]HarnessTask(nil), ds.Tasks...)
	c.PreviousTasks = append([]HarnessTask(nil), ds.PreviousTasks...)
	if ds.CurrentMessage != nil {
		cm := *ds.CurrentMessage
		c.CurrentMessage = &cm
	}
	if ds.PendingApproval != nil {
		pa := *ds.PendingApproval
		c.PendingApproval = &pa
	}
	if ds.PendingQuestion != nil {
		pq := *ds.PendingQuestion
		c.PendingQuestion = &pq
	}
	if ds.PendingPlanApproval != nil {
		pp := *ds.PendingPlanApproval
		c.PendingPlanApproval = &pp
	}
	if ds.OMProgress != nil {
		om := *ds.OMProgress
		c.OMProgress = &om
	}
	return &c
}

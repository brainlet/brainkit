// Ported from: packages/core/src/harness/types.ts
package harness

import (
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent"
	model "github.com/brainlet/brainkit/agent-kit/core/llm/model"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/workspace"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// Agent is a stub for ../agent.Agent.
// The real agent.Agent is a struct (not interface). Kept as an interface so the harness
// can hold any agent-like value without depending on agent struct internals.
// These methods represent the minimum useful contract for identification and
// pass-through. Replace with *agent.Agent when harness calls agent methods directly.
type Agent interface {
	// GetID returns the agent's unique identifier.
	GetID() string
	// GetName returns the agent's display name.
	GetName() string
}

// ToolsInput is imported from agent/types.
type ToolsInput = agent.ToolsInput

// AgentInstructions is imported from agent.
type AgentInstructions = agent.AgentInstructions

// MastraLanguageModel is imported from llm/model.
type MastraLanguageModel = model.MastraLanguageModel

// LoopStopWhen is a stub for ../loop/types.StopCondition.
// Stub: real loop.StopCondition = interface{} = any. Same shape; kept local
// to avoid dependency on loop package from harness.
type LoopStopWhen = any

// MastraMemory is a stub for ../memory/memory.MastraMemory.
// Stub: real memory.MastraMemory is an interface with many methods (ID, GetThreadById,
// SaveMessages, etc). Using = any avoids coupling harness to full memory interface.
type MastraMemory = any

// MastraCompositeStore is a stub for ../storage/base.MastraCompositeStore.
// Stub: real type is a struct with MastraBase embedding + many methods.
// Using = any avoids coupling harness to full storage interface.
type MastraCompositeStore = any

// DynamicArgument is a stub for ../types.DynamicArgument.
// Stub: real type has static/resolver/isDynamic fields with DynamicArgumentFunc[T]
// that takes DynamicArgumentContext. This stub uses simplified Value/Resolver fields.
// Shape mismatch (different field names and resolver signature).
type DynamicArgument[T any] struct {
	Value    T
	Resolver func() (T, error)
}

// Workspace is imported from workspace.
type Workspace = workspace.Workspace

// WorkspaceConfig is imported from workspace.
type WorkspaceConfig = workspace.WorkspaceConfig

// WorkspaceStatus is imported from workspace.
type WorkspaceStatus = workspace.WorkspaceStatus

// RequestContext is imported from requestcontext.
type RequestContext = requestcontext.RequestContext

// =============================================================================
// Heartbeat Handlers
// =============================================================================

// HeartbeatHandler is a periodic task that the Harness runs on a timer.
// Heartbeat handlers start during Init() and are cleaned up on StopHeartbeats().
type HeartbeatHandler struct {
	// ID is a unique identifier for this handler (used for dedup and logging).
	ID string
	// IntervalMs is the interval in milliseconds between invocations.
	IntervalMs int
	// Handler is the function to run on each tick.
	Handler func() error
	// Immediate controls whether to run the handler immediately on start (default: true).
	Immediate *bool
	// Shutdown is called when the handler is removed or all heartbeats are stopped.
	Shutdown func() error
}

// =============================================================================
// Harness Configuration
// =============================================================================

// HarnessMode represents configuration for a single agent mode within the harness.
// Each mode represents a different "personality" or capability set.
type HarnessMode struct {
	// ID is a unique identifier for this mode (e.g., "plan", "build", "review").
	ID string
	// Name is the human-readable name for display.
	Name string
	// Default indicates whether this is the default mode when harness starts.
	Default bool
	// DefaultModelID is the default model ID for this mode (e.g., "anthropic/claude-sonnet-4-20250514").
	DefaultModelID string
	// Color is a hex color for the mode indicator (e.g., "#7c3aed").
	Color string
	// Agent is the agent for this mode. In Go we use the interface directly.
	// For dynamic agent resolution based on state, use AgentFactory.
	Agent Agent
	// AgentFactory is a function that receives harness state and returns an agent.
	// Used when the agent depends on current state. Mutually exclusive with Agent.
	AgentFactory func(state map[string]any) Agent
}

// =============================================================================
// Subagents
// =============================================================================

// HarnessSubagent defines a subagent that the Harness can spawn via the built-in subagent tool.
// Each subagent runs as a fresh Agent with constrained tools and its own instructions.
type HarnessSubagent struct {
	// ID is a unique identifier for this subagent type (e.g., "explore", "plan", "execute").
	ID string
	// Name is a human-readable name shown in tool output (e.g., "Explore").
	Name string
	// Description of what this subagent does (used in auto-generated tool description).
	Description string
	// Instructions that guide the agent's behavior.
	Instructions AgentInstructions
	// Tools this subagent has direct access to.
	Tools ToolsInput
	// AllowedHarnessTools are tool IDs to pull from the harness's shared tools config.
	AllowedHarnessTools []string
	// DefaultModelID is the default model ID for this subagent type.
	DefaultModelID string
	// MaxSteps is an optional maximum number of steps for this subagent's execution loop.
	MaxSteps *int
	// StopWhen is an optional stop condition for this subagent's execution loop.
	StopWhen LoopStopWhen
}

// HarnessConfig holds configuration for creating a Harness instance.
type HarnessConfig struct {
	// ID is a unique identifier for this harness instance.
	ID string
	// ResourceID for grouping threads (e.g., project identifier).
	ResourceID string
	// Storage backend for persistence (threads, messages, state).
	Storage MastraCompositeStore
	// InitialState values.
	InitialState map[string]any
	// Memory configuration (shared across all modes).
	Memory MastraMemory
	// Modes are the available agent modes.
	Modes []HarnessMode
	// Tools available to all agents across all modes.
	Tools ToolsInput
	// Workspace configuration.
	WorkspaceConfig *WorkspaceConfig
	// Workspace is a pre-constructed workspace instance.
	Workspace *Workspace
	// HeartbeatHandlers are periodic heartbeat handlers started during Init().
	HeartbeatHandlers []HeartbeatHandler
	// IDGenerator is a custom ID generator for threads, messages, etc.
	IDGenerator func() string
	// ModelAuthChecker is a custom auth checker for model providers.
	ModelAuthChecker ModelAuthChecker
	// ModelUseCountProvider provides per-model use counts.
	ModelUseCountProvider ModelUseCountProvider
	// ModelUseCountTracker is called when a model is selected via SwitchModel().
	ModelUseCountTracker ModelUseCountTracker
	// CustomModelCatalogProvider provides additional model catalog entries.
	CustomModelCatalogProvider CustomModelCatalogProvider
	// Subagents are subagent definitions.
	Subagents []HarnessSubagent
	// ResolveModel converts a model ID string to a language model instance.
	ResolveModel func(modelID string) MastraLanguageModel
	// OMConfig holds Observational Memory configuration defaults.
	OMConfig *HarnessOMConfig
	// ToolCategoryResolver maps tool names to permission categories.
	ToolCategoryResolver func(toolName string) *ToolCategory
	// ThreadLock provides optional thread locking callbacks.
	ThreadLock *ThreadLock
}

// ThreadLock provides thread locking callbacks for preventing concurrent access.
type ThreadLock struct {
	Acquire func(threadID string) error
	Release func(threadID string) error
}

// HarnessOMConfig holds default configuration for Observational Memory.
type HarnessOMConfig struct {
	// DefaultObserverModelID is the default model ID for the observer agent.
	DefaultObserverModelID string
	// DefaultReflectorModelID is the default model ID for the reflector agent.
	DefaultReflectorModelID string
	// DefaultObservationThreshold is the default observation threshold in tokens.
	DefaultObservationThreshold *int
	// DefaultReflectionThreshold is the default reflection threshold in tokens.
	DefaultReflectionThreshold *int
}

// =============================================================================
// Permissions
// =============================================================================

// ToolCategory represents a tool category for permission grouping.
type ToolCategory string

const (
	ToolCategoryRead    ToolCategory = "read"
	ToolCategoryEdit    ToolCategory = "edit"
	ToolCategoryExecute ToolCategory = "execute"
	ToolCategoryMCP     ToolCategory = "mcp"
	ToolCategoryOther   ToolCategory = "other"
)

// PermissionPolicy represents a permission policy for a tool or category.
type PermissionPolicy string

const (
	PermissionPolicyAllow PermissionPolicy = "allow"
	PermissionPolicyAsk   PermissionPolicy = "ask"
	PermissionPolicyDeny  PermissionPolicy = "deny"
)

// PermissionRules holds permission rules for controlling tool approval behavior.
type PermissionRules struct {
	Categories map[ToolCategory]PermissionPolicy
	Tools      map[string]PermissionPolicy
}

// =============================================================================
// Model Discovery
// =============================================================================

// ModelAuthStatus represents auth status for a model's provider.
type ModelAuthStatus struct {
	HasAuth      bool
	APIKeyEnvVar string
}

// AvailableModel holds info about an available model from the provider registry.
type AvailableModel struct {
	// ID is the full model ID (e.g., "anthropic/claude-sonnet-4-20250514").
	ID string
	// Provider is the provider prefix (e.g., "anthropic").
	Provider string
	// ModelName is the model name without provider prefix.
	ModelName string
	// HasAPIKey indicates whether the provider has valid authentication.
	HasAPIKey bool
	// APIKeyEnvVar is the environment variable for the provider's API key.
	APIKeyEnvVar string
	// UseCount is the number of times this model has been used.
	UseCount int
}

// CustomAvailableModel represents additional model entries supplied by the app layer.
type CustomAvailableModel struct {
	ID           string
	Provider     string
	ModelName    string
	HasAPIKey    bool
	APIKeyEnvVar string
}

// CustomModelCatalogProvider provides additional model catalog entries.
type CustomModelCatalogProvider func() ([]CustomAvailableModel, error)

// ModelAuthChecker is a custom auth checker for model providers.
// Returns true if authenticated, false if not, nil to fall back to default check.
type ModelAuthChecker func(provider string) *bool

// ModelUseCountProvider provides per-model use counts for sorting.
type ModelUseCountProvider func() map[string]int

// ModelUseCountTracker is called when a model is selected via SwitchModel().
type ModelUseCountTracker func(modelID string)

// =============================================================================
// Harness State
// =============================================================================

// HarnessThread holds thread metadata stored in the harness.
type HarnessThread struct {
	ID         string
	ResourceID string
	Title      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	TokenUsage *TokenUsage
	Metadata   map[string]any
}

// HarnessSession holds session info for the current harness instance.
type HarnessSession struct {
	CurrentThreadID string
	CurrentModeID   string
	Threads         []HarnessThread
}

// =============================================================================
// Events
// =============================================================================

// TokenUsage holds token usage statistics from the model.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// =============================================================================
// Observational Memory Progress
// =============================================================================

// OMStatus represents the status of the Observational Memory system.
type OMStatus string

const (
	OMStatusIdle       OMStatus = "idle"
	OMStatusObserving  OMStatus = "observing"
	OMStatusReflecting OMStatus = "reflecting"
)

// OMBufferedStatus represents the status of a buffered OM operation.
type OMBufferedStatus string

const (
	OMBufferedStatusIdle     OMBufferedStatus = "idle"
	OMBufferedStatusRunning  OMBufferedStatus = "running"
	OMBufferedStatusComplete OMBufferedStatus = "complete"
)

// OMProgressState holds the full progress state for Observational Memory.
type OMProgressState struct {
	Status                   OMStatus
	PendingTokens            int
	Threshold                int
	ThresholdPercent         float64
	ObservationTokens        int
	ReflectionThreshold      int
	ReflectionThresholdPercent float64
	Buffered                 OMBuffered
	GenerationCount          int
	StepNumber               int
	CycleID                  string
	StartTime                *int64
	PreReflectionTokens      int
}

// OMBuffered holds buffered state for observations and reflection.
type OMBuffered struct {
	Observations OMBufferedObservations
	Reflection   OMBufferedReflection
}

// OMBufferedObservations holds buffered observation state.
type OMBufferedObservations struct {
	Status                   OMBufferedStatus
	Chunks                   int
	MessageTokens            int
	ProjectedMessageRemoval  int
	ObservationTokens        int
}

// OMBufferedReflection holds buffered reflection state.
type OMBufferedReflection struct {
	Status                OMBufferedStatus
	InputObservationTokens int
	ObservationTokens      int
}

// =============================================================================
// Display State
// =============================================================================

// ActiveToolState holds the state of an active tool execution.
type ActiveToolState struct {
	Name          string
	Args          any
	Status        string // "streaming_input" | "running" | "completed" | "error"
	PartialResult string
	Result        any
	IsError       bool
	ShellOutput   string
}

// ActiveSubagentState holds the state of an active subagent execution.
type ActiveSubagentState struct {
	AgentType  string
	Task       string
	ModelID    string
	ToolCalls  []SubagentToolCall
	TextDelta  string
	Status     string // "running" | "completed" | "error"
	DurationMs *int
	Result     string
}

// SubagentToolCall represents a tool call made by a subagent.
type SubagentToolCall struct {
	Name    string
	IsError bool
}

// TaskItem represents a task in the task list.
type TaskItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"` // "pending" | "in_progress" | "completed"
	ActiveForm string `json:"activeForm"`
}

// HarnessDisplayState is the canonical display state maintained by the Harness.
// This is the single source of truth for what to display.
type HarnessDisplayState struct {
	IsRunning           bool
	CurrentMessage      *HarnessMessage
	TokenUsage          TokenUsage
	ActiveTools         map[string]*ActiveToolState
	ToolInputBuffers    map[string]*ToolInputBuffer
	PendingApproval     *PendingApproval
	PendingQuestion     *PendingQuestion
	PendingPlanApproval *PendingPlanApproval
	ActiveSubagents     map[string]*ActiveSubagentState
	OMProgress          OMProgressState
	BufferingMessages   bool
	BufferingObservations bool
	ModifiedFiles       map[string]*ModifiedFile
	Tasks               []TaskItem
	PreviousTasks       []TaskItem
}

// ToolInputBuffer holds partial JSON buffers for tools whose arguments are being streamed.
type ToolInputBuffer struct {
	Text     string
	ToolName string
}

// PendingApproval represents a tool awaiting user approval.
type PendingApproval struct {
	ToolCallID string
	ToolName   string
	Args       any
}

// PendingQuestion represents a question from the agent awaiting user answer.
type PendingQuestion struct {
	QuestionID string
	Question   string
	Options    []QuestionOption
}

// QuestionOption represents an option for a question.
type QuestionOption struct {
	Label       string
	Description string
}

// PendingPlanApproval represents a plan awaiting user approval.
type PendingPlanApproval struct {
	PlanID string
	Title  string
	Plan   string
}

// ModifiedFile tracks file modifications by tool executions.
type ModifiedFile struct {
	Operations    []string
	FirstModified time.Time
}

// DefaultDisplayState creates the default/initial HarnessDisplayState.
func DefaultDisplayState() HarnessDisplayState {
	return HarnessDisplayState{
		IsRunning:           false,
		CurrentMessage:      nil,
		TokenUsage:          TokenUsage{},
		ActiveTools:         make(map[string]*ActiveToolState),
		ToolInputBuffers:    make(map[string]*ToolInputBuffer),
		PendingApproval:     nil,
		PendingQuestion:     nil,
		PendingPlanApproval: nil,
		ActiveSubagents:     make(map[string]*ActiveSubagentState),
		OMProgress:          DefaultOMProgressState(),
		BufferingMessages:   false,
		BufferingObservations: false,
		ModifiedFiles:       make(map[string]*ModifiedFile),
		Tasks:               nil,
		PreviousTasks:       nil,
	}
}

// DefaultOMProgressState creates the default OM progress state.
func DefaultOMProgressState() OMProgressState {
	return OMProgressState{
		Status:              OMStatusIdle,
		PendingTokens:       0,
		Threshold:           30000,
		ThresholdPercent:    0,
		ObservationTokens:   0,
		ReflectionThreshold: 40000,
		ReflectionThresholdPercent: 0,
		Buffered: OMBuffered{
			Observations: OMBufferedObservations{
				Status: OMBufferedStatusIdle,
			},
			Reflection: OMBufferedReflection{
				Status: OMBufferedStatusIdle,
			},
		},
		GenerationCount:     0,
		StepNumber:          0,
		PreReflectionTokens: 0,
	}
}

// =============================================================================
// Events
// =============================================================================

// HarnessEvent represents events emitted by the harness that UIs can subscribe to.
// In Go, we use a struct with a Type discriminator instead of a TypeScript union.
type HarnessEvent struct {
	Type string `json:"type"`

	// mode_changed
	ModeID         string `json:"modeId,omitempty"`
	PreviousModeID string `json:"previousModeId,omitempty"`

	// model_changed
	ModelID string `json:"modelId,omitempty"`
	Scope   string `json:"scope,omitempty"` // "global" | "thread" | "mode"

	// thread_changed / thread_created / thread_deleted
	ThreadID         string         `json:"threadId,omitempty"`
	PreviousThreadID string         `json:"previousThreadId,omitempty"`
	Thread           *HarnessThread `json:"thread,omitempty"`

	// state_changed
	State       map[string]any `json:"state,omitempty"`
	ChangedKeys []string       `json:"changedKeys,omitempty"`

	// agent_end
	Reason string `json:"reason,omitempty"` // "complete" | "aborted" | "error"

	// message events
	Message *HarnessMessage `json:"message,omitempty"`

	// tool events
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	Args       any    `json:"args,omitempty"`
	Result     any    `json:"result,omitempty"`
	IsError    bool   `json:"isError,omitempty"`

	// tool_input_delta
	ArgsTextDelta string `json:"argsTextDelta,omitempty"`

	// shell_output
	Output string `json:"output,omitempty"`
	Stream string `json:"stream,omitempty"` // "stdout" | "stderr"

	// usage_update
	Usage *TokenUsage `json:"usage,omitempty"`

	// info / error
	InfoMessage  string `json:"infoMessage,omitempty"`
	Error        error  `json:"error,omitempty"`
	ErrorType    string `json:"errorType,omitempty"`
	Retryable    bool   `json:"retryable,omitempty"`
	RetryDelay   int    `json:"retryDelay,omitempty"`

	// follow_up_queued
	Count int `json:"count,omitempty"`

	// workspace events
	WorkspaceStatus string `json:"workspaceStatus,omitempty"`
	WorkspaceID     string `json:"workspaceId,omitempty"`
	WorkspaceName   string `json:"workspaceName,omitempty"`

	// ask_question
	QuestionID string           `json:"questionId,omitempty"`
	Question   string           `json:"question,omitempty"`
	Options    []QuestionOption `json:"options,omitempty"`

	// plan_approval_required
	PlanID string `json:"planId,omitempty"`
	Title  string `json:"title,omitempty"`
	Plan   string `json:"plan,omitempty"`

	// subagent events
	AgentType     string `json:"agentType,omitempty"`
	Task          string `json:"task,omitempty"`
	TextDelta     string `json:"textDelta,omitempty"`
	SubToolName   string `json:"subToolName,omitempty"`
	SubToolArgs   any    `json:"subToolArgs,omitempty"`
	SubToolResult any    `json:"subToolResult,omitempty"`
	DurationMs    int    `json:"durationMs,omitempty"`

	// task_updated
	Tasks []TaskItem `json:"tasks,omitempty"`

	// display_state_changed
	DisplayState *HarnessDisplayState `json:"displayState,omitempty"`
}

// HarnessEventListener is a listener function for harness events.
type HarnessEventListener func(event HarnessEvent)

// =============================================================================
// Messages
// =============================================================================

// HarnessMessage is a simplified message type for UI consumption.
type HarnessMessage struct {
	ID           string
	Role         string // "user" | "assistant" | "system"
	Content      []HarnessMessageContent
	CreatedAt    time.Time
	StopReason   string // "complete" | "tool_use" | "aborted" | "error"
	ErrorMessage string
}

// HarnessMessageContent represents a content block within a harness message.
// Uses a Type discriminator with type-specific fields.
type HarnessMessageContent struct {
	Type string `json:"type"`

	// text
	Text string `json:"text,omitempty"`

	// thinking
	Thinking string `json:"thinking,omitempty"`

	// tool_call / tool_result
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	ToolArgs any  `json:"args,omitempty"`
	ToolResult any `json:"result,omitempty"`
	IsError bool   `json:"isError,omitempty"`

	// image
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`

	// file
	MediaType string `json:"mediaType,omitempty"`
	Filename  string `json:"filename,omitempty"`

	// om_observation_start / om_observation_end
	TokensToObserve  int    `json:"tokensToObserve,omitempty"`
	OperationType    string `json:"operationType,omitempty"`
	TokensObserved   int    `json:"tokensObserved,omitempty"`
	ObservationTokens int   `json:"observationTokens,omitempty"`
	ObservationDurationMs int `json:"observationDurationMs,omitempty"`
	Observations     string `json:"observations,omitempty"`
	CurrentTask      string `json:"currentTask,omitempty"`
	SuggestedResponse string `json:"suggestedResponse,omitempty"`

	// om_observation_failed
	ObservationError string `json:"observationError,omitempty"`
	TokensAttempted  int    `json:"tokensAttempted,omitempty"`
}

// =============================================================================
// Request Context
// =============================================================================

// HarnessRequestContext is harness-specific context set on the RequestContext
// under the "harness" key.
type HarnessRequestContext struct {
	// HarnessID is the harness instance ID.
	HarnessID string
	// State is the current harness state (read-only snapshot).
	State map[string]any
	// GetState returns the current harness state (live, not snapshot).
	GetState func() map[string]any
	// SetState updates harness state.
	SetState func(updates map[string]any) error
	// ThreadID is the current thread ID.
	ThreadID string
	// ResourceID is the current resource ID.
	ResourceID string
	// ModeID is the current mode ID.
	ModeID string
	// AbortSignal for the current operation. Go idiom is context.Context cancellation.
	// Kept as any for 1:1 TS port fidelity. Should be replaced with context.Context when wiring.
	AbortSignal any
	// WorkspaceRef is the workspace instance (if configured on the Harness).
	WorkspaceRef *Workspace
	// EmitEvent emits a harness event (used by tools to forward events).
	EmitEvent func(event HarnessEvent)
	// RegisterQuestion registers a pending question resolver.
	RegisterQuestion func(questionID string, resolve func(answer string))
	// RegisterPlanApproval registers a pending plan approval resolver.
	RegisterPlanApproval func(planID string, resolve func(action string, feedback string))
	// GetSubagentModelID gets the configured subagent model ID for a specific agent type.
	GetSubagentModelID func(agentType string) string
}

// Ported from: packages/core/src/agent/agent.types.ts
package agent

// Re-exported types from loop/network/validation for convenience.
// MISMATCH: cannot alias to loop/network types because they reference different MastraScorer:
//   agent.MastraScorer:  interface{ ID() string; Name() string }
//   network.MastraScorer: interface{ ID() string; Name() string; Run(ctx, ScorerRunInput) (*ScorerRunResult, error) }
// CompletionConfig.Scorers field type depends on the local MastraScorer definition.
// No circular dependency (loop doesn't import agent), but shape mismatch prevents wiring.

// CompletionConfig configures network/stream completion scoring.
// Scorers evaluate whether the task is complete after each iteration.
type CompletionConfig struct {
	// Scorers to run to determine if the task is complete.
	// Each scorer should return 0 (not complete) or 1 (complete).
	Scorers []MastraScorer `json:"-"`
	// Strategy is how to combine scorer results:
	//   "all" - All scorers must pass (score = 1) (default)
	//   "any" - At least one scorer must pass
	Strategy string `json:"strategy,omitempty"`
	// Timeout is the maximum time for all scorers (ms). Default: 600000 (10 minutes).
	Timeout int `json:"timeout,omitempty"`
	// Parallel controls whether scorers run in parallel. Default: true.
	Parallel *bool `json:"parallel,omitempty"`
	// OnComplete is called after scorers run with results.
	OnComplete func(result CompletionRunResult) error `json:"-"`
	// SuppressFeedback suppresses the completion feedback message from being saved to memory.
	SuppressFeedback bool `json:"suppressFeedback,omitempty"`
}

// CompletionRunResult is the result of running completion checks.
type CompletionRunResult struct {
	// Complete indicates whether the task is complete (based on strategy).
	Complete bool `json:"complete"`
	// CompletionReason is the reason for completion/failure.
	CompletionReason string `json:"completionReason,omitempty"`
	// Scorers contains individual scorer results.
	Scorers []ScorerResult `json:"scorers"`
	// TotalDuration is the total duration of all checks in milliseconds.
	TotalDuration int64 `json:"totalDuration"`
	// TimedOut indicates whether checks timed out.
	TimedOut bool `json:"timedOut"`
}

// ScorerResult is the result of running a single scorer.
type ScorerResult struct {
	Score       float64 `json:"score"`
	Passed      bool    `json:"passed"`
	Reason      string  `json:"reason,omitempty"`
	ScorerID    string  `json:"scorerId"`
	ScorerName  string  `json:"scorerName"`
	Duration    int64   `json:"duration"`
	FinalResult string  `json:"finalResult,omitempty"`
}

// IsTaskCompleteConfig is an alias for CompletionConfig.
type IsTaskCompleteConfig = CompletionConfig

// IsTaskCompleteRunResult is an alias for CompletionRunResult.
type IsTaskCompleteRunResult = CompletionRunResult

// StreamIsTaskCompleteConfig is the isTaskComplete scoring configuration
// for stream/generate. Reuses IsTaskCompleteConfig for consistency.
type StreamIsTaskCompleteConfig = IsTaskCompleteConfig

// ============================================================================
// Delegation Hook Types
// ============================================================================

// PrimitiveType enumerates the valid primitive types for delegation.
type PrimitiveType string

const (
	PrimitiveTypeAgent    PrimitiveType = "agent"
	PrimitiveTypeWorkflow PrimitiveType = "workflow"
)

// MessageFilterContext is the context passed to the messageFilter callback.
// Contains everything needed to decide which parent messages to share with the sub-agent.
type MessageFilterContext struct {
	// Messages is the full unfiltered messages from the parent agent's conversation history.
	Messages []MastraDBMessage `json:"messages"`
	// PrimitiveID is the ID of the primitive being delegated to.
	PrimitiveID string `json:"primitiveId"`
	// PrimitiveType is the type of primitive being delegated to.
	PrimitiveType PrimitiveType `json:"primitiveType"`
	// Prompt is the prompt being sent to the sub-agent (after any onDelegationStart modifications).
	Prompt string `json:"prompt"`
	// Iteration is the current iteration number (1-based).
	Iteration int `json:"iteration"`
	// RunID is the ID of the current run.
	RunID string `json:"runId"`
	// ThreadID is the current thread ID (if using memory).
	ThreadID string `json:"threadId,omitempty"`
	// ResourceID is the resource ID (if using memory).
	ResourceID string `json:"resourceId,omitempty"`
	// ParentAgentID is the parent agent's ID.
	ParentAgentID string `json:"parentAgentId"`
	// ParentAgentName is the parent agent's name.
	ParentAgentName string `json:"parentAgentName"`
	// ToolCallID is the tool call ID from the LLM.
	ToolCallID string `json:"toolCallId"`
}

// DelegationStartParams holds additional parameters from the tool call.
type DelegationStartParams struct {
	ThreadID     string `json:"threadId,omitempty"`
	ResourceID   string `json:"resourceId,omitempty"`
	Instructions string `json:"instructions,omitempty"`
	MaxSteps     *int   `json:"maxSteps,omitempty"`
}

// DelegationStartContext is the context passed to the onDelegationStart hook.
// Contains information about the sub-agent or workflow being called.
type DelegationStartContext struct {
	// PrimitiveID is the ID of the delegated primitive (agent or workflow).
	PrimitiveID string `json:"primitiveId"`
	// PrimitiveType is the type of primitive being delegated to.
	PrimitiveType PrimitiveType `json:"primitiveType"`
	// Prompt is the prompt being sent to the sub-agent/workflow.
	Prompt string `json:"prompt"`
	// Params contains additional parameters from the tool call.
	Params DelegationStartParams `json:"params"`
	// Iteration is the current iteration number (1-based).
	Iteration int `json:"iteration"`
	// RunID is the ID of the current run.
	RunID string `json:"runId"`
	// ThreadID is the current thread ID (if using memory).
	ThreadID string `json:"threadId,omitempty"`
	// ResourceID is the resource ID (if using memory).
	ResourceID string `json:"resourceId,omitempty"`
	// ParentAgentID is the parent agent's ID.
	ParentAgentID string `json:"parentAgentId"`
	// ParentAgentName is the parent agent's name.
	ParentAgentName string `json:"parentAgentName"`
	// ToolCallID is the tool call ID from the LLM.
	ToolCallID string `json:"toolCallId"`
	// Messages accumulated so far.
	Messages []MastraDBMessage `json:"messages"`
}

// DelegationStartResult is the result returned from onDelegationStart hook.
type DelegationStartResult struct {
	// Proceed indicates whether to proceed with the delegation (default: true).
	Proceed *bool `json:"proceed,omitempty"`
	// RejectionReason is used when Proceed is false.
	RejectionReason string `json:"rejectionReason,omitempty"`
	// ModifiedPrompt to send to the sub-agent (optional).
	ModifiedPrompt string `json:"modifiedPrompt,omitempty"`
	// ModifiedInstructions for the sub-agent (optional).
	ModifiedInstructions string `json:"modifiedInstructions,omitempty"`
	// ModifiedMaxSteps for the sub-agent (optional).
	ModifiedMaxSteps *int `json:"modifiedMaxSteps,omitempty"`
}

// OnDelegationStartHandler is the handler for delegation start events.
// Return a result to modify or reject delegation, or nil to proceed as-is.
type OnDelegationStartHandler func(ctx DelegationStartContext) (*DelegationStartResult, error)

// DelegationResult holds the result from a sub-agent/workflow execution.
type DelegationResult struct {
	Text                string `json:"text"`
	SubAgentThreadID    string `json:"subAgentThreadId,omitempty"`
	SubAgentResourceID  string `json:"subAgentResourceId,omitempty"`
}

// DelegationCompleteContext is the context passed to the onDelegationComplete hook.
type DelegationCompleteContext struct {
	// PrimitiveID is the ID of the delegated primitive.
	PrimitiveID string `json:"primitiveId"`
	// PrimitiveType is the type of primitive.
	PrimitiveType PrimitiveType `json:"primitiveType"`
	// Prompt is the prompt that was sent.
	Prompt string `json:"prompt"`
	// Result from the sub-agent/workflow.
	Result DelegationResult `json:"result"`
	// Duration of the delegation in milliseconds.
	Duration int64 `json:"duration"`
	// Success indicates whether the delegation succeeded.
	Success bool `json:"success"`
	// Error if the delegation failed.
	Error error `json:"-"`
	// Iteration is the current iteration number (1-based).
	Iteration int `json:"iteration"`
	// RunID is the ID of the current run.
	RunID string `json:"runId"`
	// ToolCallID is the tool call ID from the LLM.
	ToolCallID string `json:"toolCallId"`
	// ParentAgentID is the parent agent's ID.
	ParentAgentID string `json:"parentAgentId"`
	// ParentAgentName is the parent agent's name.
	ParentAgentName string `json:"parentAgentName"`
	// Messages accumulated so far (including the delegation result).
	Messages []MastraDBMessage `json:"messages"`
	// Bail stops all other concurrent delegations.
	// Only relevant when multiple tool calls are executed concurrently.
	Bail func() `json:"-"`
}

// DelegationCompleteResult is the result returned from onDelegationComplete hook.
type DelegationCompleteResult struct {
	// Feedback is optional feedback to add to the conversation.
	Feedback string `json:"feedback,omitempty"`
}

// OnDelegationCompleteHandler is the handler for delegation complete events.
type OnDelegationCompleteHandler func(ctx DelegationCompleteContext) (*DelegationCompleteResult, error)

// ============================================================================
// Iteration Hook Types
// ============================================================================

// IterationToolCall represents a tool call made in an iteration.
type IterationToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// IterationToolResult represents a tool result from an iteration.
type IterationToolResult struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Result any    `json:"result"`
	Error  error  `json:"-"`
}

// IterationCompleteContext is the context passed to the onIterationComplete hook.
type IterationCompleteContext struct {
	// Iteration is the current iteration number (1-based).
	Iteration int `json:"iteration"`
	// MaxIterations is the maximum iterations allowed.
	MaxIterations *int `json:"maxIterations,omitempty"`
	// Text is the text output from this iteration.
	Text string `json:"text"`
	// ToolCalls made in this iteration.
	ToolCalls []IterationToolCall `json:"toolCalls"`
	// ToolResults from this iteration.
	ToolResults []IterationToolResult `json:"toolResults"`
	// IsFinal indicates whether this is the final iteration.
	IsFinal bool `json:"isFinal"`
	// FinishReason is the reason the model stopped.
	FinishReason string `json:"finishReason"`
	// RunID is the ID of the current run.
	RunID string `json:"runId"`
	// ThreadID is the current thread ID (if using memory).
	ThreadID string `json:"threadId,omitempty"`
	// ResourceID is the resource ID (if using memory).
	ResourceID string `json:"resourceId,omitempty"`
	// AgentID is the agent identifier.
	AgentID string `json:"agentId"`
	// AgentName is the agent name.
	AgentName string `json:"agentName"`
	// Messages is all messages in the conversation.
	Messages []MastraDBMessage `json:"messages"`
}

// IterationCompleteResult is the result returned from onIterationComplete hook.
type IterationCompleteResult struct {
	// Continue controls whether to continue to the next iteration.
	//   - true: Continue to next iteration
	//   - false: Stop processing (even if model wants to continue)
	//   - nil: Let the model decide
	Continue *bool `json:"continue,omitempty"`
	// Feedback message to add to the conversation before the next iteration.
	Feedback string `json:"feedback,omitempty"`
}

// OnIterationCompleteHandler is the handler for iteration complete events.
type OnIterationCompleteHandler func(ctx IterationCompleteContext) (*IterationCompleteResult, error)

// ============================================================================
// Delegation Configuration
// ============================================================================

// MessageFilterFunc is the type for the message filter callback.
type MessageFilterFunc func(ctx MessageFilterContext) ([]MastraDBMessage, error)

// DelegationConfig holds configuration for delegation behavior during execution.
type DelegationConfig struct {
	// OnDelegationStart is called before a subagent is executed.
	// Can reject or modify the delegation.
	OnDelegationStart OnDelegationStartHandler `json:"-"`

	// OnDelegationComplete is called after a subagent execution completes.
	// Can provide feedback or stop processing.
	OnDelegationComplete OnDelegationCompleteHandler `json:"-"`

	// MessageFilter controls which parent messages are passed to each subagent
	// as conversation context. Receives the full parent message history along
	// with delegation metadata, and returns the messages that should be forwarded.
	// Runs after onDelegationStart so the prompt reflects any modifications made there.
	MessageFilter MessageFilterFunc `json:"-"`
}

// ============================================================================
// NetworkRoutingConfig
// ============================================================================

// NetworkRoutingConfig configures the routing agent's behavior.
type NetworkRoutingConfig struct {
	// AdditionalInstructions are appended to the routing agent's system prompt.
	AdditionalInstructions string `json:"additionalInstructions,omitempty"`
	// VerboseIntrospection includes verbose reasoning about why primitives were/weren't selected.
	// Default: false.
	VerboseIntrospection bool `json:"verboseIntrospection,omitempty"`
}

// ============================================================================
// NetworkOptions
// ============================================================================

// NetworkIterationCompleteContext is the context for network iteration complete callbacks.
type NetworkIterationCompleteContext struct {
	Iteration     int    `json:"iteration"`
	PrimitiveID   string `json:"primitiveId"`
	PrimitiveType string `json:"primitiveType"` // "agent" | "workflow" | "tool" | "none"
	Result        string `json:"result"`
	IsComplete    bool   `json:"isComplete"`
}

// NetworkOptions holds full configuration options for agent.network() execution.
type NetworkOptions struct {
	// Memory configures conversation persistence and retrieval.
	Memory *AgentMemoryOption `json:"memory,omitempty"`

	// AutoResumeSuspendedTools indicates whether to automatically resume suspended tools.
	AutoResumeSuspendedTools bool `json:"autoResumeSuspendedTools,omitempty"`

	// RunID is a unique identifier for this execution run.
	RunID string `json:"runId,omitempty"`

	// RequestContext holds dynamic configuration and state.
	RequestContext *RequestContext `json:"requestContext,omitempty"`

	// MaxSteps is the maximum number of iterations to run.
	MaxSteps *int `json:"maxSteps,omitempty"`

	// ModelSettings holds model-specific settings like temperature, maxTokens, topP, etc.
	ModelSettings any `json:"modelSettings,omitempty"`

	// Routing configures how primitives are selected.
	Routing *NetworkRoutingConfig `json:"routing,omitempty"`

	// Completion configures when the task is considered done.
	Completion *CompletionConfig `json:"completion,omitempty"`

	// OnIterationComplete is called after each iteration completes.
	OnIterationComplete func(ctx NetworkIterationCompleteContext) error `json:"-"`

	// StructuredOutput configures structured output for the network's final result.
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`

	// OnStepFinish is called after each LLM step within a sub-agent execution.
	OnStepFinish any `json:"-"`

	// OnError is called when an error occurs during sub-agent execution.
	OnError func(err error) error `json:"-"`

	// OnAbort is called when streaming is aborted.
	OnAbort func(event any) error `json:"-"`

	// AbortSignal to abort the streaming operation.
	// In Go, use context.Context cancellation instead.
	AbortSignal any `json:"-"`

	// Embedded observability context (partial).
	ObservabilityContext
}

// MultiPrimitiveExecutionOptions is deprecated; use NetworkOptions instead.
// Deprecated: Use NetworkOptions.
type MultiPrimitiveExecutionOptions = NetworkOptions

// ============================================================================
// AgentExecutionOptions
// ============================================================================

// AgentExecutionOptionsBase holds the base execution options shared by
// generate() and stream() in vNext mode.
type AgentExecutionOptionsBase struct {
	// Instructions to override the agent's default instructions for this execution.
	Instructions SystemMessage `json:"instructions,omitempty"`

	// System is a custom system message to include in the prompt.
	System SystemMessage `json:"system,omitempty"`

	// Context holds additional context messages (ModelMessage).
	Context []any `json:"context,omitempty"`

	// Memory configures conversation persistence and retrieval.
	Memory *AgentMemoryOption `json:"memory,omitempty"`

	// RunID is a unique identifier for this execution run.
	RunID string `json:"runId,omitempty"`

	// SavePerStep saves messages incrementally after each stream step completes. Default: false.
	SavePerStep bool `json:"savePerStep,omitempty"`

	// RequestContext holds dynamic configuration and state.
	RequestContext *RequestContext `json:"requestContext,omitempty"`

	// MaxSteps is the maximum number of steps to run.
	MaxSteps *int `json:"maxSteps,omitempty"`

	// StopWhen holds conditions for stopping execution (e.g., step count, token limit).
	StopWhen any `json:"stopWhen,omitempty"`

	// ProviderOptions holds provider-specific options passed to the language model.
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`

	// OnStepFinish is called after each execution step.
	OnStepFinish any `json:"-"`
	// OnFinish is called when execution completes.
	OnFinish any `json:"-"`
	// OnChunk is called for each streaming chunk received.
	OnChunk any `json:"-"`
	// OnError is called when an error occurs during streaming.
	OnError func(err error) error `json:"-"`
	// OnAbort is called when streaming is aborted.
	OnAbort func(event any) error `json:"-"`
	// ActiveTools lists tools that are active for this execution.
	ActiveTools []string `json:"activeTools,omitempty"`
	// AbortSignal to abort the streaming operation.
	// In Go, use context.Context cancellation instead.
	AbortSignal any `json:"-"`

	// InputProcessors to use for this execution (overrides agent's default).
	InputProcessors []InputProcessorOrWorkflow `json:"inputProcessors,omitempty"`
	// OutputProcessors to use for this execution (overrides agent's default).
	OutputProcessors []OutputProcessorOrWorkflow `json:"outputProcessors,omitempty"`
	// MaxProcessorRetries overrides agent's default maxProcessorRetries.
	MaxProcessorRetries *int `json:"maxProcessorRetries,omitempty"`

	// Toolsets are additional tool sets for this execution.
	Toolsets ToolsetsInput `json:"toolsets,omitempty"`
	// ClientTools are client-side tools available during execution.
	ClientTools ToolsInput `json:"clientTools,omitempty"`
	// ToolChoice controls tool selection strategy: "auto", "none", "required", or specific tools.
	ToolChoice any `json:"toolChoice,omitempty"`

	// ModelSettings holds model-specific settings like temperature, maxTokens, topP, etc.
	ModelSettings any `json:"modelSettings,omitempty"`

	// Scorers are evaluation scorers to run on the execution results.
	Scorers any `json:"scorers,omitempty"`
	// ReturnScorerData indicates whether to return detailed scoring data. Default: false.
	ReturnScorerData bool `json:"returnScorerData,omitempty"`
	// TracingOptions for starting new traces.
	TracingOptions *TracingOptions `json:"tracingOptions,omitempty"`

	// PrepareStep is a callback function called before each step of multi-step execution.
	PrepareStep any `json:"-"`

	// IsTaskComplete is the scoring configuration for supervisor patterns.
	// Scorers evaluate whether the task is complete after each iteration.
	IsTaskComplete *StreamIsTaskCompleteConfig `json:"isTaskComplete,omitempty"`

	// RequireToolApproval requires approval for all tool calls.
	RequireToolApproval bool `json:"requireToolApproval,omitempty"`

	// AutoResumeSuspendedTools automatically resumes suspended tools.
	AutoResumeSuspendedTools bool `json:"autoResumeSuspendedTools,omitempty"`

	// ToolCallConcurrency is the maximum number of concurrent tool calls.
	// Default: 1 when approval may be required, otherwise 10.
	ToolCallConcurrency *int `json:"toolCallConcurrency,omitempty"`

	// IncludeRawChunks includes raw chunks in the stream output (not available on all model providers).
	IncludeRawChunks bool `json:"includeRawChunks,omitempty"`

	// OnIterationComplete is called after each iteration (LLM call) completes.
	// Can control whether to continue and inject feedback.
	OnIterationComplete OnIterationCompleteHandler `json:"-"`

	// Delegation configures sub-agent and workflow tool call delegation.
	Delegation *DelegationConfig `json:"delegation,omitempty"`

	// Embedded observability context (partial).
	ObservabilityContext
}

// AgentExecutionOptions extends AgentExecutionOptionsBase with optional structured output.
// In TypeScript this used conditional types; in Go we use a single struct with optional field.
type AgentExecutionOptions struct {
	AgentExecutionOptionsBase

	// StructuredOutput configures structured output for this execution.
	// When set, the agent generates a structured response matching the schema.
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
}

// InnerAgentExecutionOptions extends AgentExecutionOptionsBase with internal execution fields.
type InnerAgentExecutionOptions struct {
	AgentExecutionOptionsBase

	// OutputWriter writes structured event chunks.
	OutputWriter OutputWriter `json:"-"`
	// Messages is the message list input for this execution.
	Messages MessageListInput `json:"messages"`
	// MethodType is the agent method type (generate, stream, etc.).
	MethodType AgentMethodType `json:"methodType"`
	// Model is an internal model override for when structuredOutput.model is used with maxSteps=1.
	Model any `json:"model,omitempty"`
	// ResumeContext holds data for resuming a suspended execution.
	ResumeContext *ResumeContext `json:"resumeContext,omitempty"`
	// ToolCallID is an optional tool call identifier.
	ToolCallID string `json:"toolCallId,omitempty"`

	// StructuredOutput configures structured output for this execution.
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
}

// ResumeContext holds data needed to resume a suspended execution.
type ResumeContext struct {
	ResumeData any `json:"resumeData"`
	Snapshot   any `json:"snapshot"`
}

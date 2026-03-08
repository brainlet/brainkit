// Ported from: packages/core/src/observability/types/tracing.ts
package types

import (
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ============================================================================
// Span Types
// ============================================================================

// SpanType enumerates AI-specific span types with their associated metadata.
type SpanType string

const (
	// SpanTypeAgentRun is the root span for agent processes.
	SpanTypeAgentRun SpanType = "agent_run"
	// SpanTypeGeneric is a generic span for custom operations.
	SpanTypeGeneric SpanType = "generic"
	// SpanTypeModelGeneration represents model generation with model calls, token usage, prompts, completions.
	SpanTypeModelGeneration SpanType = "model_generation"
	// SpanTypeModelStep represents a single model execution step within a generation (one API call).
	SpanTypeModelStep SpanType = "model_step"
	// SpanTypeModelChunk represents an individual model streaming chunk/event.
	SpanTypeModelChunk SpanType = "model_chunk"
	// SpanTypeMCPToolCall represents MCP (Model Context Protocol) tool execution.
	SpanTypeMCPToolCall SpanType = "mcp_tool_call"
	// SpanTypeProcessorRun represents input or output processor execution.
	SpanTypeProcessorRun SpanType = "processor_run"
	// SpanTypeToolCall represents function/tool execution with inputs, outputs, errors.
	SpanTypeToolCall SpanType = "tool_call"
	// SpanTypeWorkflowRun is the root span for workflow processes.
	SpanTypeWorkflowRun SpanType = "workflow_run"
	// SpanTypeWorkflowStep represents workflow step execution with step status, data flow.
	SpanTypeWorkflowStep SpanType = "workflow_step"
	// SpanTypeWorkflowConditional represents workflow conditional execution.
	SpanTypeWorkflowConditional SpanType = "workflow_conditional"
	// SpanTypeWorkflowConditionalEval represents individual condition evaluation within conditional.
	SpanTypeWorkflowConditionalEval SpanType = "workflow_conditional_eval"
	// SpanTypeWorkflowParallel represents workflow parallel execution.
	SpanTypeWorkflowParallel SpanType = "workflow_parallel"
	// SpanTypeWorkflowLoop represents workflow loop execution.
	SpanTypeWorkflowLoop SpanType = "workflow_loop"
	// SpanTypeWorkflowSleep represents workflow sleep operation.
	SpanTypeWorkflowSleep SpanType = "workflow_sleep"
	// SpanTypeWorkflowWaitEvent represents workflow wait for event operation.
	SpanTypeWorkflowWaitEvent SpanType = "workflow_wait_event"
)

// EntityType identifies the entity that created a span.
type EntityType string

const (
	EntityTypeAgent              EntityType = "agent"
	EntityTypeEval               EntityType = "eval"
	EntityTypeInputProcessor     EntityType = "input_processor"
	EntityTypeInputStepProcessor EntityType = "input_step_processor"
	EntityTypeOutputProcessor    EntityType = "output_processor"
	EntityTypeOutputStepProcessor EntityType = "output_step_processor"
	EntityTypeWorkflowStep       EntityType = "workflow_step"
	EntityTypeTool               EntityType = "tool"
	EntityTypeWorkflowRun        EntityType = "workflow_run"
)

// ============================================================================
// Type-Specific Attributes
// ============================================================================

// AIBaseAttributes is the base attributes that all spans can have.
type AIBaseAttributes struct{}

// AgentRunAttributes holds agent run span attributes.
type AgentRunAttributes struct {
	AIBaseAttributes
	// ConversationID is the conversation/thread/session identifier for multi-turn interactions.
	ConversationID string `json:"conversationId,omitempty"`
	// Instructions for the agent.
	Instructions string `json:"instructions,omitempty"`
	// Prompt for the agent.
	Prompt string `json:"prompt,omitempty"`
	// AvailableTools for this execution.
	AvailableTools []string `json:"availableTools,omitempty"`
	// MaxSteps allowed.
	MaxSteps *int `json:"maxSteps,omitempty"`
}

// InputTokenDetails provides detailed breakdown of input token usage by type.
type InputTokenDetails struct {
	// Text is regular text tokens (non-cached, non-audio, non-image).
	Text *int `json:"text,omitempty"`
	// CacheRead is tokens served from cache (cache hit/read).
	CacheRead *int `json:"cacheRead,omitempty"`
	// CacheWrite is tokens written to cache (cache creation - Anthropic only).
	CacheWrite *int `json:"cacheWrite,omitempty"`
	// Audio is audio input tokens.
	Audio *int `json:"audio,omitempty"`
	// Image is image input tokens (includes PDF pages).
	Image *int `json:"image,omitempty"`
}

// OutputTokenDetails provides detailed breakdown of output token usage by type.
type OutputTokenDetails struct {
	// Text is regular text output tokens.
	Text *int `json:"text,omitempty"`
	// Reasoning is reasoning/thinking tokens.
	Reasoning *int `json:"reasoning,omitempty"`
	// Audio is audio output tokens.
	Audio *int `json:"audio,omitempty"`
	// Image is image output tokens.
	Image *int `json:"image,omitempty"`
}

// UsageStats holds token usage statistics.
type UsageStats struct {
	// InputTokens is total input tokens.
	InputTokens *int `json:"inputTokens,omitempty"`
	// OutputTokens is total output tokens.
	OutputTokens *int `json:"outputTokens,omitempty"`
	// InputDetails is detailed breakdown of input token usage.
	InputDetails *InputTokenDetails `json:"inputDetails,omitempty"`
	// OutputDetails is detailed breakdown of output token usage.
	OutputDetails *OutputTokenDetails `json:"outputDetails,omitempty"`
}

// ModelGenerationParameters holds model generation configuration parameters.
type ModelGenerationParameters struct {
	MaxOutputTokens  *int                `json:"maxOutputTokens,omitempty"`
	Temperature      *float64            `json:"temperature,omitempty"`
	TopP             *float64            `json:"topP,omitempty"`
	TopK             *int                `json:"topK,omitempty"`
	PresencePenalty  *float64            `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float64            `json:"frequencyPenalty,omitempty"`
	StopSequences    []string            `json:"stopSequences,omitempty"`
	Seed             *int                `json:"seed,omitempty"`
	MaxRetries       *int                `json:"maxRetries,omitempty"`
	Headers          map[string]*string  `json:"headers,omitempty"`
}

// ModelGenerationAttributes holds model generation span attributes.
type ModelGenerationAttributes struct {
	AIBaseAttributes
	// Model name (e.g., 'gpt-4', 'claude-3').
	Model string `json:"model,omitempty"`
	// Provider (e.g., 'openai', 'anthropic').
	Provider string `json:"provider,omitempty"`
	// ResultType is the type of result/output this LLM call produced.
	ResultType string `json:"resultType,omitempty"`
	// Usage is token usage statistics.
	Usage *UsageStats `json:"usage,omitempty"`
	// Parameters is model parameters.
	Parameters *ModelGenerationParameters `json:"parameters,omitempty"`
	// Streaming indicates whether this was a streaming response.
	Streaming *bool `json:"streaming,omitempty"`
	// FinishReason is the reason the generation finished.
	FinishReason string `json:"finishReason,omitempty"`
	// CompletionStartTime is when the first token/chunk was received (for TTFT).
	CompletionStartTime *time.Time `json:"completionStartTime,omitempty"`
	// ResponseModel is the actual model used in the response.
	ResponseModel string `json:"responseModel,omitempty"`
	// ResponseID is the unique identifier for the response.
	ResponseID string `json:"responseId,omitempty"`
	// ServerAddress for the model endpoint.
	ServerAddress string `json:"serverAddress,omitempty"`
	// ServerPort for the model endpoint.
	ServerPort *int `json:"serverPort,omitempty"`
}

// ModelStepAttributes holds attributes for a single model execution step within a generation.
type ModelStepAttributes struct {
	AIBaseAttributes
	// StepIndex is the index of this step in the generation.
	StepIndex *int `json:"stepIndex,omitempty"`
	// Usage is token usage statistics.
	Usage *UsageStats `json:"usage,omitempty"`
	// FinishReason is the reason this step finished.
	FinishReason string `json:"finishReason,omitempty"`
	// IsContinued indicates whether execution should continue.
	IsContinued *bool `json:"isContinued,omitempty"`
	// Warnings contains result warnings.
	Warnings map[string]any `json:"warnings,omitempty"`
}

// ModelChunkAttributes holds attributes for individual streaming chunks/events.
type ModelChunkAttributes struct {
	AIBaseAttributes
	// ChunkType is the type of chunk (text-delta, reasoning-delta, tool-call, etc.).
	ChunkType string `json:"chunkType,omitempty"`
	// SequenceNumber of this chunk in the stream.
	SequenceNumber *int `json:"sequenceNumber,omitempty"`
}

// ToolCallAttributes holds tool call span attributes.
type ToolCallAttributes struct {
	AIBaseAttributes
	ToolType        string `json:"toolType,omitempty"`
	ToolDescription string `json:"toolDescription,omitempty"`
	Success         *bool  `json:"success,omitempty"`
}

// MCPToolCallAttributes holds MCP tool call span attributes.
type MCPToolCallAttributes struct {
	AIBaseAttributes
	// MCPServer identifier (required).
	MCPServer string `json:"mcpServer"`
	// ServerVersion of the MCP server.
	ServerVersion string `json:"serverVersion,omitempty"`
	// Success indicates whether tool execution was successful.
	Success *bool `json:"success,omitempty"`
}

// MessageListMutation represents a MessageList mutation performed by a processor.
type MessageListMutation struct {
	Type    string `json:"type"`
	Source  string `json:"source,omitempty"`
	Count   *int   `json:"count,omitempty"`
	IDs     []string `json:"ids,omitempty"`
	Text    string `json:"text,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Message any    `json:"message,omitempty"`
}

// ProcessorRunAttributes holds processor span attributes.
type ProcessorRunAttributes struct {
	AIBaseAttributes
	// ProcessorExecutor type (workflow or legacy).
	ProcessorExecutor string `json:"processorExecutor,omitempty"`
	// ProcessorIndex in the agent.
	ProcessorIndex *int `json:"processorIndex,omitempty"`
	// MessageListMutations performed by this processor.
	MessageListMutations []MessageListMutation `json:"messageListMutations,omitempty"`
}

// WorkflowRunStatus represents the status of a workflow run.
// Stub type - the canonical definition lives in the workflows package.
type WorkflowRunStatus string

// WorkflowStepStatus represents the status of a workflow step.
// Stub type - the canonical definition lives in the workflows package.
type WorkflowStepStatus string

// WorkflowRunAttributes holds workflow run span attributes.
type WorkflowRunAttributes struct {
	AIBaseAttributes
	Status WorkflowRunStatus `json:"status,omitempty"`
}

// WorkflowStepAttributes holds workflow step span attributes.
type WorkflowStepAttributes struct {
	AIBaseAttributes
	Status WorkflowStepStatus `json:"status,omitempty"`
}

// WorkflowConditionalAttributes holds workflow conditional span attributes.
type WorkflowConditionalAttributes struct {
	AIBaseAttributes
	// ConditionCount is the number of conditions evaluated.
	ConditionCount int `json:"conditionCount"`
	// TruthyIndexes indicates which condition indexes evaluated to true.
	TruthyIndexes []int `json:"truthyIndexes,omitempty"`
	// SelectedSteps indicates which steps will be executed.
	SelectedSteps []string `json:"selectedSteps,omitempty"`
}

// WorkflowConditionalEvalAttributes holds workflow conditional evaluation span attributes.
type WorkflowConditionalEvalAttributes struct {
	AIBaseAttributes
	// ConditionIndex is the index of this condition in the conditional.
	ConditionIndex int `json:"conditionIndex"`
	// Result of condition evaluation.
	Result *bool `json:"result,omitempty"`
}

// WorkflowParallelAttributes holds workflow parallel span attributes.
type WorkflowParallelAttributes struct {
	AIBaseAttributes
	// BranchCount is the number of parallel branches.
	BranchCount int `json:"branchCount"`
	// ParallelSteps are the step IDs being executed in parallel.
	ParallelSteps []string `json:"parallelSteps,omitempty"`
}

// WorkflowLoopAttributes holds workflow loop span attributes.
type WorkflowLoopAttributes struct {
	AIBaseAttributes
	// LoopType is the type of loop (foreach, dowhile, dountil).
	LoopType string `json:"loopType,omitempty"`
	// Iteration is the current iteration number.
	Iteration *int `json:"iteration,omitempty"`
	// TotalIterations is the total iterations (if known).
	TotalIterations *int `json:"totalIterations,omitempty"`
	// Concurrency is the number of steps to run concurrently in foreach loop.
	Concurrency *int `json:"concurrency,omitempty"`
}

// WorkflowSleepAttributes holds workflow sleep span attributes.
type WorkflowSleepAttributes struct {
	AIBaseAttributes
	// DurationMs is the sleep duration in milliseconds.
	DurationMs *int `json:"durationMs,omitempty"`
	// UntilDate is the sleep until date.
	UntilDate *time.Time `json:"untilDate,omitempty"`
	// SleepType is the sleep type (fixed or dynamic).
	SleepType string `json:"sleepType,omitempty"`
}

// WorkflowWaitEventAttributes holds workflow wait event span attributes.
type WorkflowWaitEventAttributes struct {
	AIBaseAttributes
	// EventName being waited for.
	EventName string `json:"eventName,omitempty"`
	// TimeoutMs in milliseconds.
	TimeoutMs *int `json:"timeoutMs,omitempty"`
	// EventReceived indicates whether event was received or timed out.
	EventReceived *bool `json:"eventReceived,omitempty"`
	// WaitDurationMs is the wait duration in milliseconds.
	WaitDurationMs *int `json:"waitDurationMs,omitempty"`
}

// AnySpanAttributes is a union type for cases that need to handle any span attributes.
// In Go we use any (interface{}) since we can't have discriminated union types.
type AnySpanAttributes = any

// AnySpan is a type alias for any span type, used when the specific span type is not important.
type AnySpan = any

// ============================================================================
// Span Error Info
// ============================================================================

// SpanErrorInfo holds error information for a span.
type SpanErrorInfo struct {
	Message  string         `json:"message"`
	ID       string         `json:"id,omitempty"`
	Domain   string         `json:"domain,omitempty"`
	Category string         `json:"category,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
}

// ============================================================================
// Span Interfaces
// ============================================================================

// Span is the interface for active spans used internally for tracing.
// In TypeScript this was a generic interface Span<TType extends SpanType>.
// In Go we use the interface pattern with SpanType carried as a field.
type Span interface {
	// ID returns the unique span identifier.
	ID() string
	// TraceID returns the OpenTelemetry-compatible trace ID (32 hex chars).
	TraceID() string
	// Name returns the span name.
	Name() string
	// Type returns the span type.
	Type() SpanType
	// EntityType returns the entity type that created the span.
	GetEntityType() *EntityType
	// EntityID returns the entity id that created the span.
	EntityID() string
	// EntityName returns the entity name that created the span.
	EntityName() string
	// StartTime returns when the span started.
	StartTime() time.Time
	// EndTime returns when the span ended (nil if still active).
	EndTime() *time.Time
	// Attributes returns span-type specific attributes.
	Attributes() AnySpanAttributes
	// Metadata returns user-defined metadata.
	Metadata() map[string]any
	// Tags returns labels used to categorize and filter traces (root spans only).
	Tags() []string
	// Input returns the input passed at the start of the span.
	Input() any
	// Output returns the output generated at the end of the span.
	Output() any
	// ErrorInfo returns error information if the span failed.
	ErrorInfo() *SpanErrorInfo
	// IsEvent returns true if this is an event span.
	IsEvent() bool

	// IsInternal returns true if this is an internal span.
	IsInternal() bool
	// Parent returns the parent span reference (nil for root spans).
	Parent() Span
	// ObservabilityInstance returns the pointer to the ObservabilityInstance.
	ObservabilityInstance() ObservabilityInstance
	// GetTraceState returns trace-level state shared across all spans in this trace.
	GetTraceState() *TraceState

	// End ends the span.
	End(options *EndSpanOptions)
	// Error records an error for the span.
	Error(options ErrorSpanOptions)
	// Update updates span attributes.
	Update(options UpdateSpanOptions)
	// CreateChildSpan creates a child span.
	CreateChildSpan(options ChildSpanOptions) Span
	// CreateEventSpan creates an event span.
	CreateEventSpan(options ChildEventOptions) Span

	// IsRootSpan returns true if the span is the root span of a trace.
	IsRootSpan() bool
	// IsValid returns true if the span is a valid span (not a no-op span).
	IsValid() bool
	// GetParentSpanID gets the closest parent spanId that isn't an internal span.
	GetParentSpanID(includeInternalSpans bool) string
	// FindParent finds the closest parent span of a specific type.
	FindParent(spanType SpanType) Span
	// ExportSpan returns a lightweight span ready for export.
	ExportSpan(includeInternalSpans bool) *ExportedSpan
	// ExternalTraceID returns the traceId on span, unless NoOpSpan, then empty string.
	ExternalTraceID() string
	// ExecuteInContext executes an async function within this span's tracing context.
	ExecuteInContext(fn func() (any, error)) (any, error)
	// ExecuteInContextSync executes a synchronous function within this span's tracing context.
	ExecuteInContextSync(fn func() any) any
}

// BridgeSpanContext provides context execution methods for bridge integration.
type BridgeSpanContext interface {
	ExecuteInContext(fn func() (any, error)) (any, error)
	ExecuteInContextSync(fn func() any) any
}

// AIModelGenerationSpan is a specialized span for MODEL_GENERATION spans.
type AIModelGenerationSpan interface {
	Span
	// CreateTracker creates a ModelSpanTracker for tracking model execution steps and chunks.
	CreateTracker() IModelSpanTracker
}

// IModelSpanTracker tracks model execution steps and chunks.
type IModelSpanTracker interface {
	GetTracingContext() TracingContext
	ReportGenerationError(options ErrorSpanOptions)
	EndGeneration(options *EndGenerationOptions)
	StartStep(payload any)
}

// ============================================================================
// Span Data & Exported Span
// ============================================================================

// SpanData is the data structure shared between exported and recorded spans.
type SpanData struct {
	ID           string         `json:"id"`
	TraceID      string         `json:"traceId"`
	Name         string         `json:"name"`
	Type         SpanType       `json:"type"`
	EntityType   *EntityType    `json:"entityType,omitempty"`
	EntityID     string         `json:"entityId,omitempty"`
	EntityName   string         `json:"entityName,omitempty"`
	StartTime    time.Time      `json:"startTime"`
	EndTime      *time.Time     `json:"endTime,omitempty"`
	Attributes   any            `json:"attributes,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Input        any            `json:"input,omitempty"`
	Output       any            `json:"output,omitempty"`
	ErrorInfo    *SpanErrorInfo `json:"errorInfo,omitempty"`
	IsEvent      bool           `json:"isEvent"`
	ParentSpanID string         `json:"parentSpanId,omitempty"`
	IsRootSpan   bool           `json:"isRootSpan"`
}

// ExportedSpan is the format sent to ObservabilityExporter implementations.
type ExportedSpan = SpanData

// AnyExportedSpan is a union type for cases that need to handle any exported span.
type AnyExportedSpan = ExportedSpan

// ============================================================================
// Recorded Span & Trace Interfaces
// ============================================================================

// RecordedSpan is span data that has been captured/persisted and can have
// scores and feedback attached post-hoc.
type RecordedSpan interface {
	// SpanData returns the underlying span data.
	GetSpanData() SpanData
	// Parent returns the parent recorded span (nil for root spans).
	Parent() RecordedSpan
	// Children returns child spans in execution order.
	Children() []RecordedSpan
	// AddScore adds a quality score to this recorded span.
	AddScore(score ScoreInput)
	// AddFeedback adds user feedback to this recorded span.
	AddFeedback(feedback FeedbackInput)
}

// RecordedTrace is a complete execution trace loaded from storage.
type RecordedTrace interface {
	// TraceID returns the trace identifier.
	TraceID() string
	// RootSpan returns the root span of the trace tree.
	RootSpan() RecordedSpan
	// Spans returns all spans in flat array for iteration.
	Spans() []RecordedSpan
	// GetSpan gets a specific recorded span by ID.
	GetSpan(spanID string) RecordedSpan
	// AddScore adds a score at the trace level.
	AddScore(score ScoreInput)
	// AddFeedback adds feedback at the trace level.
	AddFeedback(feedback FeedbackInput)
}

// ============================================================================
// Span Create/Update/Error Option Types
// ============================================================================

// CreateBaseOptions holds base options for creating spans.
type CreateBaseOptions struct {
	// Attributes is span-type specific attributes.
	Attributes any `json:"attributes,omitempty"`
	// Metadata is span metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Name is the span name.
	Name string `json:"name"`
	// Type is the span type.
	Type SpanType `json:"type"`
	// EntityType that created the span.
	EntityType *EntityType `json:"entityType,omitempty"`
	// EntityID that created the span.
	EntityID string `json:"entityId,omitempty"`
	// EntityName that created the span.
	EntityName string `json:"entityName,omitempty"`
	// TracingPolicy is policy-level tracing configuration.
	TracingPolicy *TracingPolicy `json:"tracingPolicy,omitempty"`
	// RequestContext for metadata extraction.
	RequestContext *requestcontext.RequestContext `json:"-"`
}

// CreateSpanOptions holds options for creating new spans.
type CreateSpanOptions struct {
	CreateBaseOptions
	// Input data.
	Input any `json:"input,omitempty"`
	// Output data (for event spans).
	Output any `json:"output,omitempty"`
	// Tags used to categorize and filter traces (root spans only).
	Tags []string `json:"tags,omitempty"`
	// Parent span.
	Parent Span `json:"-"`
	// IsEvent indicates if this is an event span.
	IsEvent bool `json:"isEvent,omitempty"`
	// TraceID to use for this span (1-32 hex chars). Root spans only.
	TraceID string `json:"traceId,omitempty"`
	// SpanID to use for this span (1-16 hex chars). Rebuild only.
	SpanID string `json:"spanId,omitempty"`
	// ParentSpanID to use for this span (1-16 hex chars). Root spans only.
	ParentSpanID string `json:"parentSpanId,omitempty"`
	// StartTime for this span. Rebuild only.
	StartTime *time.Time `json:"startTime,omitempty"`
	// TraceState is trace-level state shared across all spans in this trace.
	TraceState *TraceState `json:"traceState,omitempty"`
}

// StartSpanOptions holds options for starting new spans.
type StartSpanOptions struct {
	CreateSpanOptions
	// CustomSamplerOptions passed when using a custom sampler strategy.
	CustomSamplerOptions *CustomSamplerOptions `json:"customSamplerOptions,omitempty"`
	// TracingOptions for this execution.
	TracingOptions *TracingOptions `json:"tracingOptions,omitempty"`
}

// ChildSpanOptions holds options for new child spans.
type ChildSpanOptions struct {
	CreateBaseOptions
	// Input data.
	Input any `json:"input,omitempty"`
}

// ChildEventOptions holds options for new child event spans.
type ChildEventOptions struct {
	CreateBaseOptions
	// Output data.
	Output any `json:"output,omitempty"`
}

// EndSpanOptions holds options for ending a span.
type EndSpanOptions struct {
	// Attributes is span-type specific attributes (partial update).
	Attributes any `json:"attributes,omitempty"`
	// Metadata is span metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Output data.
	Output any `json:"output,omitempty"`
}

// EndGenerationOptions holds options for ending a model generation span.
type EndGenerationOptions struct {
	EndSpanOptions
	// Usage is raw usage data from AI SDK.
	Usage any `json:"usage,omitempty"`
	// ProviderMetadata is provider-specific metadata.
	ProviderMetadata any `json:"providerMetadata,omitempty"`
}

// UpdateSpanOptions holds options for updating a span.
type UpdateSpanOptions struct {
	// Attributes is span-type specific attributes (partial update).
	Attributes any `json:"attributes,omitempty"`
	// Metadata is span metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Input data.
	Input any `json:"input,omitempty"`
	// Output data.
	Output any `json:"output,omitempty"`
}

// ErrorSpanOptions holds options for recording an error on a span.
type ErrorSpanOptions struct {
	// Attributes is span-type specific attributes (partial update).
	Attributes any `json:"attributes,omitempty"`
	// Metadata is span metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Error is the error associated with the issue.
	Error error `json:"-"`
	// MastraError is the MastraError if applicable (checked before Error).
	MastraError *mastraerror.MastraError `json:"-"`
	// EndSpan indicates whether to end the span.
	EndSpan bool `json:"endSpan,omitempty"`
}

// GetOrCreateSpanOptions holds options for creating or getting a child span.
type GetOrCreateSpanOptions struct {
	Type           SpanType                      `json:"type"`
	Name           string                        `json:"name"`
	EntityType     *EntityType                   `json:"entityType,omitempty"`
	EntityID       string                        `json:"entityId,omitempty"`
	EntityName     string                        `json:"entityName,omitempty"`
	Input          any                           `json:"input,omitempty"`
	Attributes     any                           `json:"attributes,omitempty"`
	Metadata       map[string]any                `json:"metadata,omitempty"`
	TracingPolicy  *TracingPolicy                `json:"tracingPolicy,omitempty"`
	TracingOptions *TracingOptions               `json:"tracingOptions,omitempty"`
	TracingContext *TracingContext                `json:"-"`
	RequestContext *requestcontext.RequestContext `json:"-"`
	// Mastra is a reference to the Mastra instance for root span creation.
	// In Go this is typed as any since the Mastra type creates a circular dependency.
	Mastra any `json:"-"`
}

// ============================================================================
// InternalSpans — Bitwise options
// ============================================================================

// InternalSpans is a bitwise flag for setting different types of spans as internal.
type InternalSpans int

const (
	// InternalSpansNone means no spans are marked internal.
	InternalSpansNone InternalSpans = 0
	// InternalSpansWorkflow marks workflow spans as internal.
	InternalSpansWorkflow InternalSpans = 1 << 0
	// InternalSpansAgent marks agent spans as internal.
	InternalSpansAgent InternalSpans = 1 << 1
	// InternalSpansTool marks tool spans as internal.
	InternalSpansTool InternalSpans = 1 << 2
	// InternalSpansModel marks model spans as internal.
	InternalSpansModel InternalSpans = 1 << 3
	// InternalSpansAll marks all spans as internal.
	InternalSpansAll InternalSpans = (1 << 4) - 1
)

// TracingPolicy defines policy-level tracing configuration applied when creating
// a workflow or agent.
type TracingPolicy struct {
	// Internal is bitwise options to set different types of spans as internal.
	Internal InternalSpans `json:"internal,omitempty"`
}

// TraceState holds trace-level state computed once at the start of a trace
// and shared by all spans within that trace.
type TraceState struct {
	// RequestContextKeys to extract as metadata for all spans in this trace.
	RequestContextKeys []string `json:"requestContextKeys"`
	// HideInput indicates whether input data should be hidden.
	HideInput bool `json:"hideInput,omitempty"`
	// HideOutput indicates whether output data should be hidden.
	HideOutput bool `json:"hideOutput,omitempty"`
}

// TracingOptions holds options passed when starting a new agent or workflow execution.
type TracingOptions struct {
	// Metadata to add to the root trace span.
	Metadata map[string]any `json:"metadata,omitempty"`
	// RequestContextKeys is additional keys to extract as metadata for this trace.
	RequestContextKeys []string `json:"requestContextKeys,omitempty"`
	// TraceID to use for this execution (1-32 hex chars).
	TraceID string `json:"traceId,omitempty"`
	// ParentSpanID to use for this execution (1-16 hex chars).
	ParentSpanID string `json:"parentSpanId,omitempty"`
	// Tags to apply to this trace.
	Tags []string `json:"tags,omitempty"`
	// HideInput hides input data from all spans in this trace.
	HideInput bool `json:"hideInput,omitempty"`
	// HideOutput hides output data from all spans in this trace.
	HideOutput bool `json:"hideOutput,omitempty"`
}

// SpanIds holds identifiers returned from bridge span creation.
type SpanIds struct {
	TraceID      string `json:"traceId"`
	SpanID       string `json:"spanId"`
	ParentSpanID string `json:"parentSpanId,omitempty"`
}

// TracingContext holds the context for tracing that flows through workflow and agent execution.
type TracingContext struct {
	// CurrentSpan for creating child spans and adding metadata.
	CurrentSpan Span
}

// TracingProperties holds properties returned to the user for working with traces externally.
type TracingProperties struct {
	// TraceID used on the execution (if the execution was traced).
	TraceID string `json:"traceId,omitempty"`
}

// ============================================================================
// Exporter and Processor Interfaces
// ============================================================================

// TracingEventType enumerates tracing event types.
type TracingEventType string

const (
	TracingEventTypeSpanStarted TracingEventType = "span_started"
	TracingEventTypeSpanUpdated TracingEventType = "span_updated"
	TracingEventTypeSpanEnded   TracingEventType = "span_ended"
)

// TracingEvent represents tracing events that can be exported.
type TracingEvent struct {
	Type         TracingEventType `json:"type"`
	ExportedSpan AnyExportedSpan  `json:"exportedSpan"`
}

// SpanOutputProcessor is the interface for span processors.
type SpanOutputProcessor interface {
	// Name returns the processor name.
	Name() string
	// Process processes a span before export.
	Process(span Span) Span
	// Shutdown shuts down the processor.
	Shutdown() error
}

// CustomSpanFormatter is a function type for formatting exported spans at the exporter level.
type CustomSpanFormatter func(span AnyExportedSpan) (AnyExportedSpan, error)

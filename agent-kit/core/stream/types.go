// Ported from: packages/core/src/stream/types.ts
package stream

import (
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// LanguageModelV2FinishReason mirrors @ai-sdk/provider-v5 LanguageModelV2FinishReason.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type LanguageModelV2FinishReason string

const (
	FinishReasonStop          LanguageModelV2FinishReason = "stop"
	FinishReasonLength        LanguageModelV2FinishReason = "length"
	FinishReasonContentFilter LanguageModelV2FinishReason = "content-filter"
	FinishReasonToolCalls     LanguageModelV2FinishReason = "tool-calls"
	FinishReasonError         LanguageModelV2FinishReason = "error"
	FinishReasonOther         LanguageModelV2FinishReason = "other"
	FinishReasonUnknown       LanguageModelV2FinishReason = "unknown"
)

// LanguageModelV2CallWarning mirrors @ai-sdk/provider-v5 LanguageModelV2CallWarning.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type LanguageModelV2CallWarning struct {
	Type    string `json:"type"`
	Setting string `json:"setting,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

// LanguageModelV2ResponseMetadata mirrors @ai-sdk/provider-v5 LanguageModelV2ResponseMetadata.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type LanguageModelV2ResponseMetadata struct {
	ID        string            `json:"id,omitempty"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
	ModelID   string            `json:"modelId,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// LanguageModelV2StreamPart mirrors @ai-sdk/provider-v5 LanguageModelV2StreamPart.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type LanguageModelV2StreamPart struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

// FinishReasonV1 mirrors @internal/ai-sdk-v4 FinishReason.
// ai-kit only ported V3. V4 types remain local stubs.
type FinishReasonV1 string

// LanguageModelRequestMetadata mirrors @internal/ai-sdk-v4 LanguageModelRequestMetadata.
// ai-kit only ported V3. V4 types remain local stubs.
type LanguageModelRequestMetadata struct {
	Body map[string]any `json:"body,omitempty"`
}

// LanguageModelV1LogProbs mirrors @internal/ai-sdk-v4 LogProbs.
// ai-kit only ported V3. V4 types remain local stubs.
type LanguageModelV1LogProbs struct {
	Tokens []any `json:"tokens,omitempty"`
}

// ModelMessage mirrors @internal/ai-sdk-v5 ModelMessage.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 types remain local stubs.
type ModelMessage map[string]any

// StepResult mirrors @internal/ai-sdk-v5 StepResult.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 types remain local stubs.
type StepResult map[string]any

// ToolSet mirrors @internal/ai-sdk-v5 ToolSet.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 types remain local stubs.
type ToolSet map[string]any

// TypedToolCall mirrors @internal/ai-sdk-v5 TypedToolCall.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 types remain local stubs.
type TypedToolCall struct {
	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	Args       any    `json:"args,omitempty"`
}

// UIMessage mirrors @internal/ai-sdk-v5 UIMessage.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 types remain local stubs.
type UIMessage map[string]any

// AIV5ResponseMessage mirrors ../agent/message-list AIV5ResponseMessage.
// Stub: real agent.AIV5ResponseMessage has different struct shape; kept as lightweight alias
// to avoid coupling stream consumers to agent's internal message format.
type AIV5ResponseMessage map[string]any

// AIV5StepResultContent mirrors ../agent/message-list/types AIV5Type.StepResult content.
// Stub: parallel-stubs architecture — real type lives in agent/messagelist with different shape.
type AIV5StepResultContent []any

// MastraDBMessage mirrors ../agent/message-list/types MastraDBMessage.
// Stub: real agent.MastraDBMessage is a full struct with typed fields; this lightweight
// map alias is used to decouple stream types from agent internals.
type MastraDBMessage map[string]any

// StructuredOutputOptions mirrors ../agent/types StructuredOutputOptions.
// Stub: simplified shape — real agent type has additional fields; importing would
// pull agent package into stream's dependency graph.
type StructuredOutputOptions struct {
	Schema any `json:"schema,omitempty"`
}

// MastraLanguageModel is wired to the real llm/model.MastraLanguageModel interface.
// No circular dependency: llm/model does not import stream.
// Provides SpecificationVersion(), Provider(), ModelID() methods.
type MastraLanguageModel = model.MastraLanguageModel

// ScorerResult mirrors ../loop ScorerResult.
// Stub: real loop.ScorerResult uses float64 Score + extra fields (Duration, ScorerID);
// this simplified version avoids coupling stream to loop package internals.
type ScorerResult struct {
	Name    string `json:"name"`
	Score   any    `json:"score"`
	Passed  bool   `json:"passed"`
	Reason  string `json:"reason,omitempty"`
	Details any    `json:"details,omitempty"`
}

// ObservabilityContext mirrors ../observability ObservabilityContext.
// Stub: real observability.ObservabilityContext has Tracing/LoggerVNext/Metrics fields;
// kept empty to avoid coupling stream to observability internals.
type ObservabilityContext struct{}

// OutputProcessorOrWorkflow mirrors ../processors OutputProcessorOrWorkflow.
// Stub: parallel-stubs architecture — real type requires processor/workflow dependencies.
type OutputProcessorOrWorkflow any

// RequestContext mirrors ../request-context RequestContext.
// Stub: real requestcontext.RequestContext is a struct with sync.RWMutex + registry;
// kept as any to avoid coupling stream to requestcontext internals.
type RequestContext any

// OutputSchema mirrors ./base/schema OutputSchema.
// Stub: defined here to avoid circular import between stream/ and stream/base/.
type OutputSchema any

// WorkflowRunStatus mirrors ../workflows/types WorkflowRunStatus.
// Stub: workflows imports stream (circular dep); must remain local definition.
type WorkflowRunStatus string

const (
	WorkflowRunStatusRunning   WorkflowRunStatus = "running"
	WorkflowRunStatusSuccess   WorkflowRunStatus = "success"
	WorkflowRunStatusFailed    WorkflowRunStatus = "failed"
	WorkflowRunStatusCanceled  WorkflowRunStatus = "canceled"
	WorkflowRunStatusSuspended WorkflowRunStatus = "suspended"
	WorkflowRunStatusPaused    WorkflowRunStatus = "paused"
	WorkflowRunStatusTripwire  WorkflowRunStatus = "tripwire"
)

// WorkflowStepStatus mirrors ../workflows/types WorkflowStepStatus.
// Stub: workflows imports stream (circular dep); must remain local definition.
type WorkflowStepStatus string

const (
	WorkflowStepStatusPending   WorkflowStepStatus = "pending"
	WorkflowStepStatusRunning   WorkflowStepStatus = "running"
	WorkflowStepStatusSuccess   WorkflowStepStatus = "success"
	WorkflowStepStatusFailed    WorkflowStepStatus = "failed"
	WorkflowStepStatusSuspended WorkflowStepStatus = "suspended"
	WorkflowStepStatusWaiting   WorkflowStepStatus = "waiting"
)

// ---------------------------------------------------------------------------
// ChunkFrom enum
// ---------------------------------------------------------------------------

// ChunkFrom represents the origin of a stream chunk.
type ChunkFrom string

const (
	ChunkFromAgent    ChunkFrom = "AGENT"
	ChunkFromUser     ChunkFrom = "USER"
	ChunkFromSystem   ChunkFrom = "SYSTEM"
	ChunkFromWorkflow ChunkFrom = "WORKFLOW"
	ChunkFromNetwork  ChunkFrom = "NETWORK"
)

// ---------------------------------------------------------------------------
// MastraFinishReason — extended finish reason with Mastra-specific values
// ---------------------------------------------------------------------------

// MastraFinishReason extends LanguageModelV2FinishReason with Mastra-specific values.
// "tripwire" and "retry" are used for processor scenarios.
type MastraFinishReason string

const (
	MastraFinishReasonTripwire MastraFinishReason = "tripwire"
	MastraFinishReasonRetry    MastraFinishReason = "retry"
)

// ---------------------------------------------------------------------------
// JSON value types
// ---------------------------------------------------------------------------

// JSONValue can be a string, number, boolean, object, array, or null.
// In Go we use any for maximum flexibility — callers should assert concrete types.
type JSONValue = any

// JSONObject is a JSON object with string keys.
type JSONObject = map[string]JSONValue

// JSONArray is a JSON array.
type JSONArray = []JSONValue

// ReadonlyJSONValue mirrors the readonly variant (same as JSONValue in Go since Go
// doesn't have a built-in readonly concept).
type ReadonlyJSONValue = any

// ReadonlyJSONObject mirrors the readonly variant.
type ReadonlyJSONObject = map[string]ReadonlyJSONValue

// ReadonlyJSONArray mirrors the readonly variant.
type ReadonlyJSONArray = []ReadonlyJSONValue

// ProviderMetadata contains additional provider-specific metadata.
// The outer map is keyed by provider name, the inner by provider-specific key.
type ProviderMetadata map[string]map[string]JSONValue

// ---------------------------------------------------------------------------
// StreamTransport
// ---------------------------------------------------------------------------

// StreamTransport describes a transport mechanism for streaming.
type StreamTransport struct {
	Type          string `json:"type"` // e.g. "openai-websocket"
	CloseFunc     func() `json:"-"`
	CloseOnFinish bool   `json:"closeOnFinish"`
}

// Close invokes the transport's close function.
func (t *StreamTransport) Close() {
	if t.CloseFunc != nil {
		t.CloseFunc()
	}
}

// StreamTransportRef is a mutable reference to a StreamTransport.
type StreamTransportRef struct {
	Current *StreamTransport
}

// ---------------------------------------------------------------------------
// Payload types
// ---------------------------------------------------------------------------

// BaseChunkType is the common base for all chunk types.
type BaseChunkType struct {
	RunID    string         `json:"runId"`
	From     ChunkFrom      `json:"from"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ResponseMetadataPayload contains response metadata.
type ResponseMetadataPayload struct {
	Signature string `json:"signature,omitempty"`
	Extra     map[string]any
}

// TextStartPayload marks the start of a text content part.
type TextStartPayload struct {
	ID               string           `json:"id"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

// TextDeltaPayload carries a text delta chunk.
type TextDeltaPayload struct {
	ID               string           `json:"id"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	Text             string           `json:"text"`
}

// TextEndPayload marks the end of a text content part.
type TextEndPayload struct {
	ID               string           `json:"id"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	Extra            map[string]any
}

// ReasoningStartPayload marks the start of a reasoning content part.
type ReasoningStartPayload struct {
	ID               string           `json:"id"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	Signature        string           `json:"signature,omitempty"`
}

// ReasoningDeltaPayload carries a reasoning text delta.
type ReasoningDeltaPayload struct {
	ID               string           `json:"id"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	Text             string           `json:"text"`
}

// ReasoningEndPayload marks the end of a reasoning content part.
type ReasoningEndPayload struct {
	ID               string           `json:"id"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	Signature        string           `json:"signature,omitempty"`
}

// SourcePayload carries information about a source reference.
type SourcePayload struct {
	ID               string           `json:"id"`
	SourceType       string           `json:"sourceType"` // "url" | "document"
	Title            string           `json:"title"`
	MimeType         string           `json:"mimeType,omitempty"`
	Filename         string           `json:"filename,omitempty"`
	URL              string           `json:"url,omitempty"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

// FilePayload carries a file chunk.
type FilePayload struct {
	Data             any              `json:"data"` // string or []byte
	Base64           string           `json:"base64,omitempty"`
	MimeType         string           `json:"mimeType"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

// ---------------------------------------------------------------------------
// MastraMetadata types
// ---------------------------------------------------------------------------

// MastraMetadataMessage represents a message in MastraMetadata.
type MastraMetadataMessage struct {
	Type       string           `json:"type"` // "text" | "tool"
	Content    string           `json:"content,omitempty"`
	ToolName   string           `json:"toolName,omitempty"`
	ToolInput  ReadonlyJSONValue `json:"toolInput,omitempty"`
	ToolOutput ReadonlyJSONValue `json:"toolOutput,omitempty"`
	Args       ReadonlyJSONValue `json:"args,omitempty"`
	ToolCallID string           `json:"toolCallId,omitempty"`
	Result     ReadonlyJSONValue `json:"result,omitempty"`
}

// MastraMetadata carries Mastra-specific metadata on tool call args.
type MastraMetadata struct {
	IsStreaming       *bool                   `json:"isStreaming,omitempty"`
	From              string                  `json:"from,omitempty"` // "AGENT" | "WORKFLOW" | "USER" | "SYSTEM"
	NetworkMetadata   ReadonlyJSONObject      `json:"networkMetadata,omitempty"`
	ToolOutput        any                     `json:"toolOutput,omitempty"` // ReadonlyJSONValue or []ReadonlyJSONValue
	Messages          []MastraMetadataMessage `json:"messages,omitempty"`
	WorkflowFullState ReadonlyJSONObject      `json:"workflowFullState,omitempty"`
	SelectionReason   string                  `json:"selectionReason,omitempty"`
}

// ---------------------------------------------------------------------------
// Tool-related payloads
// ---------------------------------------------------------------------------

// ToolCallPayload describes a tool invocation.
type ToolCallPayload struct {
	ToolCallID       string           `json:"toolCallId"`
	ToolName         string           `json:"toolName"`
	Args             any              `json:"args,omitempty"` // may contain __mastraMetadata
	ProviderExecuted *bool            `json:"providerExecuted,omitempty"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	Output           any              `json:"output,omitempty"`
	Dynamic          *bool            `json:"dynamic,omitempty"`
}

// ToolResultPayload describes the result of a tool invocation.
type ToolResultPayload struct {
	ToolCallID       string           `json:"toolCallId"`
	ToolName         string           `json:"toolName"`
	Result           any              `json:"result"`
	IsError          *bool            `json:"isError,omitempty"`
	ProviderExecuted *bool            `json:"providerExecuted,omitempty"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	Args             any              `json:"args,omitempty"`
	Dynamic          *bool            `json:"dynamic,omitempty"`
}

// DynamicToolCallPayload is a ToolCallPayload with any-typed args/output.
type DynamicToolCallPayload = ToolCallPayload

// DynamicToolResultPayload is a ToolResultPayload with any-typed result/args.
type DynamicToolResultPayload = ToolResultPayload

// ToolCallInputStreamingStartPayload marks the start of streaming tool call args.
type ToolCallInputStreamingStartPayload struct {
	ToolCallID       string           `json:"toolCallId"`
	ToolName         string           `json:"toolName"`
	ProviderExecuted *bool            `json:"providerExecuted,omitempty"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	Dynamic          *bool            `json:"dynamic,omitempty"`
}

// ToolCallDeltaPayload carries a delta for streaming tool call args.
type ToolCallDeltaPayload struct {
	ArgsTextDelta    string           `json:"argsTextDelta"`
	ToolCallID       string           `json:"toolCallId"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	ToolName         string           `json:"toolName,omitempty"`
}

// ToolCallInputStreamingEndPayload marks the end of streaming tool call args.
type ToolCallInputStreamingEndPayload struct {
	ToolCallID       string           `json:"toolCallId"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

// ---------------------------------------------------------------------------
// Finish / Error / Start payloads
// ---------------------------------------------------------------------------

// FinishPayloadStepResult contains step-level finish information.
type FinishPayloadStepResult struct {
	// Reason includes 'tripwire' and 'retry' for processor scenarios.
	Reason      string                       `json:"reason"`
	Warnings    []LanguageModelV2CallWarning  `json:"warnings,omitempty"`
	IsContinued *bool                         `json:"isContinued,omitempty"`
	Logprobs    *LanguageModelV1LogProbs      `json:"logprobs,omitempty"`
}

// FinishPayloadOutput contains output-level finish information.
type FinishPayloadOutput struct {
	Usage LanguageModelUsage   `json:"usage"`
	Steps []MastraStepResult   `json:"steps,omitempty"`
}

// FinishPayloadMetadata contains metadata for the finish event.
type FinishPayloadMetadata struct {
	ProviderMetadata ProviderMetadata              `json:"providerMetadata,omitempty"`
	Request          *LanguageModelRequestMetadata `json:"request,omitempty"`
	Extra            map[string]any
}

// FinishPayloadMessages contains the messages at finish time.
type FinishPayloadMessages struct {
	All     []ModelMessage          `json:"all"`
	User    []ModelMessage          `json:"user"`
	NonUser []AIV5ResponseMessage   `json:"nonUser"`
}

// FinishPayload is emitted when the model finishes generating.
type FinishPayload struct {
	StepResult FinishPayloadStepResult  `json:"stepResult"`
	Output     FinishPayloadOutput      `json:"output"`
	Metadata   FinishPayloadMetadata    `json:"metadata"`
	Messages   FinishPayloadMessages    `json:"messages"`
	Response   *LLMStepResultResponse   `json:"response,omitempty"`
	Extra      map[string]any
}

// ErrorPayload carries error information.
type ErrorPayload struct {
	Error any            `json:"error"`
	Extra map[string]any
}

// RawPayload carries raw provider data.
type RawPayload map[string]any

// StartPayload marks the start of generation.
type StartPayload map[string]any

// ---------------------------------------------------------------------------
// Step payloads
// ---------------------------------------------------------------------------

// StepStartPayload is emitted at the start of a step.
type StepStartPayload struct {
	MessageID string         `json:"messageId,omitempty"`
	Request   map[string]any `json:"request"`
	Warnings  []LanguageModelV2CallWarning `json:"warnings,omitempty"`
	Extra     map[string]any
}

// StepFinishPayloadStepResult contains step-level result data.
type StepFinishPayloadStepResult struct {
	Logprobs    *LanguageModelV1LogProbs     `json:"logprobs,omitempty"`
	IsContinued *bool                        `json:"isContinued,omitempty"`
	Warnings    []LanguageModelV2CallWarning `json:"warnings,omitempty"`
	Reason      LanguageModelV2FinishReason  `json:"reason"`
}

// StepFinishPayloadOutput contains step output data.
type StepFinishPayloadOutput struct {
	Text      string             `json:"text,omitempty"`
	ToolCalls []TypedToolCall    `json:"toolCalls,omitempty"`
	Usage     LanguageModelUsage `json:"usage"`
	Steps     []MastraStepResult `json:"steps,omitempty"`
	Object    any                `json:"object,omitempty"` // OUTPUT
}

// StepFinishPayloadMetadata contains step metadata.
type StepFinishPayloadMetadata struct {
	Request          *LanguageModelRequestMetadata `json:"request,omitempty"`
	ProviderMetadata ProviderMetadata              `json:"providerMetadata,omitempty"`
	Extra            map[string]any
}

// StepFinishPayloadMessages contains optional messages at step finish.
type StepFinishPayloadMessages struct {
	All     []ModelMessage        `json:"all"`
	User    []ModelMessage        `json:"user"`
	NonUser []AIV5ResponseMessage `json:"nonUser"`
}

// StepFinishPayload is emitted when a step finishes.
type StepFinishPayload struct {
	ID               string                           `json:"id,omitempty"`
	ProviderMetadata ProviderMetadata                 `json:"providerMetadata,omitempty"`
	TotalUsage       *LanguageModelUsage              `json:"totalUsage,omitempty"`
	Response         *LanguageModelV2ResponseMetadata  `json:"response,omitempty"`
	MessageID        string                           `json:"messageId,omitempty"`
	StepResult       StepFinishPayloadStepResult      `json:"stepResult"`
	Output           StepFinishPayloadOutput          `json:"output"`
	Metadata         StepFinishPayloadMetadata        `json:"metadata"`
	Messages         *StepFinishPayloadMessages       `json:"messages,omitempty"`
	Extra            map[string]any
}

// ---------------------------------------------------------------------------
// Tool error, abort, reasoning payloads
// ---------------------------------------------------------------------------

// ToolErrorPayload describes a tool execution error.
type ToolErrorPayload struct {
	ID               string           `json:"id,omitempty"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	ToolCallID       string           `json:"toolCallId"`
	ToolName         string           `json:"toolName"`
	Args             map[string]any   `json:"args,omitempty"`
	Error            any              `json:"error"`
	ProviderExecuted *bool            `json:"providerExecuted,omitempty"`
}

// AbortPayload is emitted when the stream is aborted.
type AbortPayload map[string]any

// ReasoningSignaturePayload carries a reasoning signature.
type ReasoningSignaturePayload struct {
	ID               string           `json:"id"`
	Signature        string           `json:"signature"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

// RedactedReasoningPayload carries redacted reasoning data.
type RedactedReasoningPayload struct {
	ID               string           `json:"id"`
	Data             any              `json:"data"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

// ---------------------------------------------------------------------------
// Tool output / Step output payloads
// ---------------------------------------------------------------------------

// ToolOutputPayload describes a tool's output chunk.
type ToolOutputPayload struct {
	Output     any            `json:"output"`
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName,omitempty"`
	Extra      map[string]any
}

// DynamicToolOutputPayload is a ToolOutputPayload with any-typed output.
type DynamicToolOutputPayload = ToolOutputPayload

// NestedWorkflowOutput represents nested workflow output.
type NestedWorkflowOutput struct {
	From    ChunkFrom      `json:"from"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"` // may contain output, usage, etc.
	Extra   map[string]any
}

// StepOutputPayload wraps output from a step (which can be a ChunkType or NestedWorkflowOutput).
type StepOutputPayload struct {
	Output any            `json:"output"` // ChunkType or NestedWorkflowOutput
	Extra  map[string]any
}

// WatchPayload carries watch event data.
type WatchPayload map[string]any

// ---------------------------------------------------------------------------
// Tripwire and scoring payloads
// ---------------------------------------------------------------------------

// TripwirePayload is emitted when a processor triggers a tripwire.
type TripwirePayload struct {
	// Reason for the tripwire.
	Reason string `json:"reason"`
	// Retry indicates whether the agent should retry with the tripwire reason as feedback.
	Retry *bool `json:"retry,omitempty"`
	// Metadata is strongly typed metadata from the processor.
	Metadata any `json:"metadata,omitempty"`
	// ProcessorID is the ID of the processor that triggered the tripwire.
	ProcessorID string `json:"processorId,omitempty"`
}

// IsTaskCompletePayload is emitted during stream/generate scoring.
type IsTaskCompletePayload struct {
	Iteration         int            `json:"iteration"`
	Passed            bool           `json:"passed"`
	Results           []ScorerResult `json:"results"`
	Duration          float64        `json:"duration"`
	TimedOut          bool           `json:"timedOut"`
	Reason            string         `json:"reason,omitempty"`
	MaxIterationReached bool         `json:"maxIterationReached"`
	SuppressFeedback  bool           `json:"suppressFeedback"`
}

// ---------------------------------------------------------------------------
// Tool call approval / suspended payloads
// ---------------------------------------------------------------------------

// ToolCallApprovalPayload describes a tool call pending approval.
type ToolCallApprovalPayload struct {
	ToolCallID   string         `json:"toolCallId"`
	ToolName     string         `json:"toolName"`
	Args         map[string]any `json:"args"`
	ResumeSchema string         `json:"resumeSchema"`
}

// ToolCallSuspendedPayload describes a suspended tool call.
type ToolCallSuspendedPayload struct {
	ToolCallID     string         `json:"toolCallId"`
	ToolName       string         `json:"toolName"`
	SuspendPayload any            `json:"suspendPayload"`
	Args           map[string]any `json:"args"`
	ResumeSchema   string         `json:"resumeSchema"`
}

// ---------------------------------------------------------------------------
// Network-specific payload types
// ---------------------------------------------------------------------------

// RoutingAgentStartInputData contains the input data for routing agent start.
type RoutingAgentStartInputData struct {
	Task                 string `json:"task"`
	PrimitiveID          string `json:"primitiveId"`
	PrimitiveType        string `json:"primitiveType"`
	Result               string `json:"result,omitempty"`
	Iteration            int    `json:"iteration"`
	ThreadID             string `json:"threadId,omitempty"`
	ThreadResourceID     string `json:"threadResourceId,omitempty"`
	IsOneOff             bool   `json:"isOneOff"`
	VerboseIntrospection bool   `json:"verboseIntrospection"`
}

// RoutingAgentStartPayload is emitted when the routing agent starts.
type RoutingAgentStartPayload struct {
	AgentID   string                     `json:"agentId"`
	NetworkID string                     `json:"networkId"`
	RunID     string                     `json:"runId"`
	InputData RoutingAgentStartInputData `json:"inputData"`
}

// RoutingAgentEndPayload is emitted when the routing agent finishes.
type RoutingAgentEndPayload struct {
	Task            string             `json:"task"`
	PrimitiveID     string             `json:"primitiveId"`
	PrimitiveType   string             `json:"primitiveType"`
	Prompt          string             `json:"prompt"`
	Result          string             `json:"result"`
	IsComplete      *bool              `json:"isComplete,omitempty"`
	SelectionReason string             `json:"selectionReason"`
	Iteration       int                `json:"iteration"`
	RunID           string             `json:"runId"`
	Usage           LanguageModelUsage `json:"usage"`
}

// RoutingAgentTextDeltaPayload carries routing agent text delta.
type RoutingAgentTextDeltaPayload struct {
	Text string `json:"text"`
}

// RoutingAgentTextStartPayload marks the start of routing agent text.
type RoutingAgentTextStartPayload struct {
	RunID string `json:"runId"`
}

// AgentExecutionStartArgs contains args for agent execution start.
type AgentExecutionStartArgs struct {
	Task            string `json:"task"`
	PrimitiveID     string `json:"primitiveId"`
	PrimitiveType   string `json:"primitiveType"`
	Prompt          string `json:"prompt"`
	Result          string `json:"result"`
	IsComplete      *bool  `json:"isComplete,omitempty"`
	SelectionReason string `json:"selectionReason"`
	Iteration       int    `json:"iteration"`
}

// AgentExecutionStartPayload is emitted when an agent execution starts.
type AgentExecutionStartPayload struct {
	AgentID string                  `json:"agentId"`
	Args    AgentExecutionStartArgs `json:"args"`
	RunID   string                  `json:"runId"`
}

// AgentExecutionApprovalPayload is emitted when an agent execution needs approval.
type AgentExecutionApprovalPayload struct {
	ToolCallApprovalPayload
	AgentID         string             `json:"agentId"`
	Usage           LanguageModelUsage `json:"usage"`
	RunID           string             `json:"runId"`
	SelectionReason string             `json:"selectionReason"`
}

// AgentExecutionSuspendedPayload is emitted when an agent execution is suspended.
type AgentExecutionSuspendedPayload struct {
	ToolCallSuspendedPayload
	AgentID        string             `json:"agentId"`
	SuspendPayload any                `json:"suspendPayload"`
	Usage          LanguageModelUsage `json:"usage"`
	RunID          string             `json:"runId"`
	SelectionReason string            `json:"selectionReason"`
}

// AgentExecutionEndPayload is emitted when an agent execution ends.
type AgentExecutionEndPayload struct {
	Task       string             `json:"task"`
	AgentID    string             `json:"agentId"`
	Result     string             `json:"result"`
	IsComplete bool               `json:"isComplete"`
	Iteration  int                `json:"iteration"`
	Usage      LanguageModelUsage `json:"usage"`
	RunID      string             `json:"runId"`
}

// WorkflowExecutionStartPayload is emitted when a workflow execution starts within a network.
type WorkflowExecutionStartPayload struct {
	Name       string                  `json:"name"`
	WorkflowID string                  `json:"workflowId"`
	Args       AgentExecutionStartArgs `json:"args"` // same shape
	RunID      string                  `json:"runId"`
}

// WorkflowExecutionEndPayload is emitted when a workflow execution ends within a network.
type WorkflowExecutionEndPayload struct {
	Name          string             `json:"name"`
	WorkflowID    string             `json:"workflowId"`
	Task          string             `json:"task"`
	PrimitiveID   string             `json:"primitiveId"`
	PrimitiveType string             `json:"primitiveType"`
	Result        string             `json:"result"`
	IsComplete    bool               `json:"isComplete"`
	Iteration     int                `json:"iteration"`
	Usage         LanguageModelUsage `json:"usage"`
	RunID         string             `json:"runId"`
}

// WorkflowExecutionSuspendPayload is emitted when a workflow execution is suspended.
type WorkflowExecutionSuspendPayload struct {
	ToolCallSuspendedPayload
	Name            string             `json:"name"`
	WorkflowID      string             `json:"workflowId"`
	SuspendPayload  any                `json:"suspendPayload"`
	Usage           LanguageModelUsage `json:"usage"`
	RunID           string             `json:"runId"`
	SelectionReason string             `json:"selectionReason"`
}

// ToolExecutionStartPayload is emitted when a tool execution starts within a network.
type ToolExecutionStartPayload struct {
	Args  map[string]any `json:"args"`
	RunID string         `json:"runId"`
}

// ToolExecutionApprovalPayload is emitted when a tool execution needs approval.
type ToolExecutionApprovalPayload struct {
	ToolCallApprovalPayload
	SelectionReason string `json:"selectionReason"`
	RunID           string `json:"runId"`
}

// ToolExecutionSuspendedPayload is emitted when a tool execution is suspended.
type ToolExecutionSuspendedPayload struct {
	ToolCallSuspendedPayload
	SelectionReason string `json:"selectionReason"`
	RunID           string `json:"runId"`
}

// ToolExecutionEndPayload is emitted when a tool execution ends within a network.
type ToolExecutionEndPayload struct {
	Task          string `json:"task"`
	PrimitiveID   string `json:"primitiveId"`
	PrimitiveType string `json:"primitiveType"`
	Result        any    `json:"result"`
	IsComplete    bool   `json:"isComplete"`
	Iteration     int    `json:"iteration"`
	ToolCallID    string `json:"toolCallId"`
	ToolName      string `json:"toolName"`
}

// NetworkStepFinishPayload is emitted when a network step finishes.
type NetworkStepFinishPayload struct {
	Task       string `json:"task"`
	Result     string `json:"result"`
	IsComplete bool   `json:"isComplete"`
	Iteration  int    `json:"iteration"`
	RunID      string `json:"runId"`
}

// NetworkFinishPayload is emitted when the entire network finishes.
type NetworkFinishPayload struct {
	Task             string             `json:"task"`
	PrimitiveID      string             `json:"primitiveId"`
	PrimitiveType    string             `json:"primitiveType"`
	Prompt           string             `json:"prompt"`
	Result           string             `json:"result"`
	Object           any                `json:"object,omitempty"` // OUTPUT when structuredOutput is provided
	IsComplete       *bool              `json:"isComplete,omitempty"`
	CompletionReason string             `json:"completionReason"`
	Iteration        int                `json:"iteration"`
	ThreadID         string             `json:"threadId,omitempty"`
	ThreadResourceID string             `json:"threadResourceId,omitempty"`
	IsOneOff         bool               `json:"isOneOff"`
	Usage            LanguageModelUsage `json:"usage"`
}

// NetworkValidationStartPayload is emitted when network validation starts.
type NetworkValidationStartPayload struct {
	RunID       string `json:"runId"`
	Iteration   int    `json:"iteration"`
	ChecksCount int    `json:"checksCount"`
}

// NetworkValidationEndPayload is emitted when network validation ends.
type NetworkValidationEndPayload struct {
	RunID               string         `json:"runId"`
	Iteration           int            `json:"iteration"`
	Passed              bool           `json:"passed"`
	Results             []ScorerResult `json:"results"`
	Duration            float64        `json:"duration"`
	TimedOut            bool           `json:"timedOut"`
	Reason              string         `json:"reason,omitempty"`
	MaxIterationReached bool           `json:"maxIterationReached"`
	SuppressFeedback    bool           `json:"suppressFeedback"`
}

// ---------------------------------------------------------------------------
// Abort payloads (by primitive type)
// ---------------------------------------------------------------------------

// RoutingAgentAbortPayload is emitted when a routing agent is aborted.
type RoutingAgentAbortPayload struct {
	PrimitiveType string `json:"primitiveType"` // "routing"
	PrimitiveID   string `json:"primitiveId"`
}

// AgentExecutionAbortPayload is emitted when an agent execution is aborted.
type AgentExecutionAbortPayload struct {
	PrimitiveType string `json:"primitiveType"` // "agent"
	PrimitiveID   string `json:"primitiveId"`
}

// WorkflowExecutionAbortPayload is emitted when a workflow execution is aborted.
type WorkflowExecutionAbortPayload struct {
	PrimitiveType string `json:"primitiveType"` // "workflow"
	PrimitiveID   string `json:"primitiveId"`
}

// ToolExecutionAbortPayload is emitted when a tool execution is aborted.
type ToolExecutionAbortPayload struct {
	PrimitiveType string `json:"primitiveType"` // "tool"
	PrimitiveID   string `json:"primitiveId"`
}

// ---------------------------------------------------------------------------
// DataChunkType
// ---------------------------------------------------------------------------

// DataChunkType represents a custom data event.
type DataChunkType struct {
	Type string `json:"type"` // "data-<name>"
	Data any    `json:"data"`
	ID   string `json:"id,omitempty"`
}

// ---------------------------------------------------------------------------
// ChunkType — the unified stream chunk discriminated union
// ---------------------------------------------------------------------------

// ChunkType is the Go equivalent of the TS discriminated union.
// In Go we use a single struct with a Type discriminator and a Payload field
// that holds the specific payload type. Callers switch on Type and assert Payload.
//
// For "object" / "object-result" chunks, use the Object field instead of Payload.
type ChunkType struct {
	BaseChunkType

	// Type is the discriminator (e.g. "text-delta", "tool-call", "finish", etc.).
	Type string `json:"type"`

	// Payload holds the type-specific payload. Callers should type-assert
	// based on Type. For example, if Type == "text-delta", Payload is *TextDeltaPayload.
	Payload any `json:"payload,omitempty"`

	// Object is used for "object" and "object-result" chunk types.
	Object any `json:"object,omitempty"`

	// Data is used for DataChunkType ("data-*" types).
	Data any `json:"data,omitempty"`

	// ID is an optional identifier (used by some chunk types like workflow-step-start).
	ID string `json:"id,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for ChunkType.
func (c ChunkType) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"type":  c.Type,
		"runId": c.RunID,
		"from":  c.From,
	}
	if c.Metadata != nil {
		m["metadata"] = c.Metadata
	}
	if c.Payload != nil {
		m["payload"] = c.Payload
	}
	if c.Object != nil {
		m["object"] = c.Object
	}
	if c.Data != nil {
		m["data"] = c.Data
	}
	if c.ID != "" {
		m["id"] = c.ID
	}
	return json.Marshal(m)
}

// AgentChunkType is an alias for ChunkType representing agent-specific chunks.
// In TS this is a separate union, but in Go we use the same struct with different Type values.
type AgentChunkType = ChunkType

// NetworkChunkType is an alias for ChunkType representing network-specific chunks.
type NetworkChunkType = ChunkType

// TypedChunkType is an alias for ChunkType (Go doesn't need the union distinction).
type TypedChunkType = ChunkType

// ---------------------------------------------------------------------------
// WorkflowStreamEvent
// ---------------------------------------------------------------------------

// WorkflowStreamEvent represents workflow-specific stream events.
// Uses the same ChunkType struct; Type values include:
//   - "workflow-start"
//   - "workflow-finish"
//   - "workflow-canceled"
//   - "workflow-paused"
//   - "workflow-step-start"
//   - "workflow-step-finish"
//   - "workflow-step-suspended"
//   - "workflow-step-waiting"
//   - "workflow-step-output"
//   - "workflow-step-progress"
//   - "workflow-step-result"
type WorkflowStreamEvent = ChunkType

// ---------------------------------------------------------------------------
// Workflow payload types (inlined from WorkflowStreamEvent union)
// ---------------------------------------------------------------------------

// WorkflowStartPayload is the payload for "workflow-start".
type WorkflowStartPayload struct {
	WorkflowID string `json:"workflowId"`
}

// WorkflowFinishPayload is the payload for "workflow-finish".
type WorkflowFinishPayload struct {
	WorkflowStatus WorkflowRunStatus  `json:"workflowStatus"`
	Output         WorkflowFinishUsage `json:"output"`
	Metadata       map[string]any     `json:"metadata"`
	Tripwire       *StepTripwireData  `json:"tripwire,omitempty"`
}

// WorkflowFinishUsage contains token usage at workflow finish.
type WorkflowFinishUsage struct {
	Usage LanguageModelUsage `json:"usage"`
}

// WorkflowStepStartPayload is the payload for "workflow-step-start".
type WorkflowStepStartPayload struct {
	ID             string         `json:"id"`
	StepCallID     string         `json:"stepCallId"`
	Status         WorkflowStepStatus `json:"status"`
	Output         map[string]any `json:"output,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	ResumePayload  map[string]any `json:"resumePayload,omitempty"`
	SuspendPayload map[string]any `json:"suspendPayload,omitempty"`
}

// WorkflowStepFinishPayload is the payload for "workflow-step-finish".
type WorkflowStepFinishPayload struct {
	ID       string         `json:"id"`
	Metadata map[string]any `json:"metadata"`
}

// WorkflowStepSuspendedPayload is the payload for "workflow-step-suspended".
type WorkflowStepSuspendedPayload struct {
	ID             string         `json:"id"`
	Status         WorkflowStepStatus `json:"status"`
	Output         map[string]any `json:"output,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	ResumePayload  map[string]any `json:"resumePayload,omitempty"`
	SuspendPayload map[string]any `json:"suspendPayload,omitempty"`
}

// WorkflowStepWaitingPayload is the payload for "workflow-step-waiting".
type WorkflowStepWaitingPayload struct {
	ID        string         `json:"id"`
	Payload   map[string]any `json:"payload"`
	StartedAt int64          `json:"startedAt"`
	Status    WorkflowStepStatus `json:"status"`
}

// WorkflowStepProgressPayload is the payload for "workflow-step-progress".
type WorkflowStepProgressPayload struct {
	ID              string         `json:"id"`
	CompletedCount  int            `json:"completedCount"`
	TotalCount      int            `json:"totalCount"`
	CurrentIndex    int            `json:"currentIndex"`
	IterationStatus string         `json:"iterationStatus"` // "success" | "failed" | "suspended"
	IterationOutput map[string]any `json:"iterationOutput,omitempty"`
}

// WorkflowStepResultPayload is the payload for "workflow-step-result".
type WorkflowStepResultPayload struct {
	ID             string             `json:"id"`
	StepCallID     string             `json:"stepCallId"`
	Status         WorkflowStepStatus `json:"status"`
	Output         map[string]any     `json:"output,omitempty"`
	Payload        map[string]any     `json:"payload,omitempty"`
	ResumePayload  map[string]any     `json:"resumePayload,omitempty"`
	SuspendPayload map[string]any     `json:"suspendPayload,omitempty"`
	Tripwire       *StepTripwireData  `json:"tripwire,omitempty"`
}

// ---------------------------------------------------------------------------
// LanguageModelV2StreamResult
// ---------------------------------------------------------------------------

// LanguageModelV2StreamResult represents a raw language model stream result.
type LanguageModelV2StreamResult struct {
	Stream      <-chan LanguageModelV2StreamPart `json:"-"`
	Request     LLMStepResultRequest            `json:"request"`
	Response    *LLMStepResultResponse          `json:"response,omitempty"`
	RawResponse any                             `json:"rawResponse"`
	Warnings    []LanguageModelV2CallWarning    `json:"warnings,omitempty"`
}

// OnResult is a callback receiving a stream result without the stream itself.
type OnResult func(result LanguageModelV2StreamResultMeta)

// LanguageModelV2StreamResultMeta is LanguageModelV2StreamResult without the Stream field.
type LanguageModelV2StreamResultMeta struct {
	Request     LLMStepResultRequest       `json:"request"`
	Response    *LLMStepResultResponse     `json:"response,omitempty"`
	RawResponse any                        `json:"rawResponse"`
	Warnings    []LanguageModelV2CallWarning `json:"warnings,omitempty"`
}

// CreateStream is a factory that creates a new LanguageModelV2StreamResult.
type CreateStream func() (*LanguageModelV2StreamResult, error)

// ---------------------------------------------------------------------------
// Convenience chunk aliases
// ---------------------------------------------------------------------------

// SourceChunk is a source-type chunk.
type SourceChunk = ChunkType

// FileChunk is a file-type chunk.
type FileChunk = ChunkType

// ToolCallChunk is a tool-call-type chunk.
type ToolCallChunk = ChunkType

// ToolResultChunk is a tool-result-type chunk.
type ToolResultChunk = ChunkType

// ReasoningChunk is a reasoning-type chunk.
type ReasoningChunk = ChunkType

// ---------------------------------------------------------------------------
// Model manager types
// ---------------------------------------------------------------------------

// ModelManagerModelConfig configures a model within the model manager.
type ModelManagerModelConfig struct {
	Model      MastraLanguageModel `json:"model"`
	MaxRetries int                 `json:"maxRetries"`
	ID         string              `json:"id"`
	Headers    map[string]string   `json:"headers,omitempty"`
}

// ExecuteStreamModelManager runs a callback against the model manager configuration.
type ExecuteStreamModelManager func(callback func(config ModelManagerModelConfig, isLastModel bool) error) error

// ---------------------------------------------------------------------------
// LanguageModelUsage — extended usage type
// ---------------------------------------------------------------------------

// LanguageModelUsage extends base usage with raw provider data.
type LanguageModelUsage struct {
	InputTokens       int `json:"inputTokens"`
	OutputTokens      int `json:"outputTokens"`
	TotalTokens       int `json:"totalTokens"`
	ReasoningTokens   int `json:"reasoningTokens,omitempty"`
	CachedInputTokens int `json:"cachedInputTokens,omitempty"`
	// Raw is raw usage data from the provider, preserved for advanced use cases.
	Raw any `json:"raw,omitempty"`
}

// PartialModel contains optional model identification fields.
type PartialModel struct {
	ModelID  string `json:"modelId,omitempty"`
	Provider string `json:"provider,omitempty"`
	Version  string `json:"version,omitempty"`
}

// ---------------------------------------------------------------------------
// Callback types
// ---------------------------------------------------------------------------

// MastraOnStepFinishCallback is called when a step finishes.
type MastraOnStepFinishCallback func(event MastraOnStepFinishEvent) error

// MastraOnStepFinishEvent is the event passed to MastraOnStepFinishCallback.
type MastraOnStepFinishEvent struct {
	LLMStepResult
	Model *PartialModel `json:"model,omitempty"`
	RunID string        `json:"runId,omitempty"`
}

// MastraOnFinishCallback is called when the entire generation finishes.
type MastraOnFinishCallback func(event MastraOnFinishCallbackArgs) error

// MastraOnFinishCallbackArgs contains all data at generation finish.
type MastraOnFinishCallbackArgs struct {
	LLMStepResult
	Error      any                `json:"error,omitempty"` // Error | string | {message, stack}
	Object     any                `json:"object,omitempty"`
	Steps      []LLMStepResult    `json:"steps"`
	TotalUsage LanguageModelUsage `json:"totalUsage"`
	Model      *PartialModel      `json:"model,omitempty"`
	RunID      string             `json:"runId,omitempty"`
}

// MastraModelOutputOptions carries all options for model output construction.
type MastraModelOutputOptions struct {
	RunID              string                    `json:"runId"`
	ToolCallStreaming   *bool                     `json:"toolCallStreaming,omitempty"`
	OnFinish           MastraOnFinishCallback    `json:"-"`
	OnStepFinish       MastraOnStepFinishCallback `json:"-"`
	IncludeRawChunks   *bool                     `json:"includeRawChunks,omitempty"`
	StructuredOutput   *StructuredOutputOptions  `json:"structuredOutput,omitempty"`
	OutputProcessors   []OutputProcessorOrWorkflow `json:"outputProcessors,omitempty"`
	IsLLMExecutionStep *bool                     `json:"isLLMExecutionStep,omitempty"`
	ReturnScorerData   *bool                     `json:"returnScorerData,omitempty"`
	ProcessorStates    map[string]any            `json:"processorStates,omitempty"`
	RequestContext     RequestContext            `json:"requestContext,omitempty"`
	TransportRef       *StreamTransportRef       `json:"transportRef,omitempty"`
	ObservabilityContext
}

// ---------------------------------------------------------------------------
// StepTripwireData
// ---------------------------------------------------------------------------

// StepTripwireData carries tripwire information attached to a step when a
// processor triggers a tripwire. When a step has tripwire data, its text
// is excluded from the final output.
type StepTripwireData struct {
	Reason      string `json:"reason"`
	Retry       *bool  `json:"retry,omitempty"`
	Metadata    any    `json:"metadata,omitempty"`
	ProcessorID string `json:"processorId,omitempty"`
}

// MastraStepResult extends StepResult with tripwire data.
type MastraStepResult struct {
	StepResult
	// Tripwire data if this step was rejected by a processor.
	Tripwire *StepTripwireData `json:"tripwire,omitempty"`
}

// ---------------------------------------------------------------------------
// LLMStepResult
// ---------------------------------------------------------------------------

// LLMStepResultRequest contains the request body for an LLM step.
type LLMStepResultRequest struct {
	Body any `json:"body,omitempty"`
}

// LLMStepResultResponse contains the response data for an LLM step.
type LLMStepResultResponse struct {
	Headers      map[string]string `json:"headers,omitempty"`
	Messages     []StepResult      `json:"messages,omitempty"`
	DBMessages   []MastraDBMessage `json:"dbMessages,omitempty"`
	UIMessages   []UIMessage       `json:"uiMessages,omitempty"`
	ID           string            `json:"id,omitempty"`
	Timestamp    *time.Time        `json:"timestamp,omitempty"`
	ModelID      string            `json:"modelId,omitempty"`
	Extra        map[string]any
}

// LLMStepResult holds the accumulated result of a single LLM step.
type LLMStepResult struct {
	StepType           string                   `json:"stepType,omitempty"` // "initial" | "tool-result"
	ToolCalls          []ToolCallChunk           `json:"toolCalls"`
	ToolResults        []ToolResultChunk         `json:"toolResults"`
	DynamicToolCalls   []ToolCallChunk           `json:"dynamicToolCalls"`
	DynamicToolResults []ToolResultChunk         `json:"dynamicToolResults"`
	StaticToolCalls    []ToolCallChunk           `json:"staticToolCalls"`
	StaticToolResults  []ToolResultChunk         `json:"staticToolResults"`
	Files              []FileChunk               `json:"files"`
	Sources            []SourceChunk             `json:"sources"`
	Text               string                    `json:"text"`
	Reasoning          []ReasoningChunk          `json:"reasoning"`
	Content            AIV5StepResultContent     `json:"content"`
	FinishReason       string                    `json:"finishReason,omitempty"`
	Usage              LanguageModelUsage        `json:"usage"`
	Warnings           []LanguageModelV2CallWarning `json:"warnings"`
	Request            LLMStepResultRequest      `json:"request"`
	Response           LLMStepResultResponse     `json:"response"`
	ReasoningText      *string                   `json:"reasoningText"`
	ProviderMetadata   ProviderMetadata          `json:"providerMetadata"`
	Tripwire           *StepTripwireData         `json:"tripwire,omitempty"`
}

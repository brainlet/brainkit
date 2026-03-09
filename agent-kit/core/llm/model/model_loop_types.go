// Ported from: packages/core/src/llm/model/model.loop.types.ts
package model

import (
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// ModelMethodType
// ---------------------------------------------------------------------------

// ModelMethodType represents the method type for model loop calls.
// TS: export type ModelMethodType = 'generate' | 'stream';
type ModelMethodType string

const (
	ModelMethodGenerate ModelMethodType = "generate"
	ModelMethodStream   ModelMethodType = "stream"
)

// ---------------------------------------------------------------------------
// Stub types for unported packages referenced by ModelLoopStreamArgs
// ---------------------------------------------------------------------------

// RequestContext is imported from the requestcontext package.
type RequestContext = *requestcontext.RequestContext

// LoopOptions is a stub for loop/types.LoopOptions.
// STUB REASON: Cannot import loop due to circular dependency: loop imports llm/model.
// The real loop.LoopOptions has typed fields (Mastra, Logger, Models as
// []ModelManagerModelConfig). This stub uses `any` for those fields.
// In TS this is a generic type LoopOptions<TOOLS extends ToolSet, OUTPUT>.
// The fields below capture the shape used by ModelLoopStreamArgs.
type LoopOptions struct {
	// ResumeContext holds context for resuming a suspended loop.
	ResumeContext any `json:"resumeContext,omitempty"`
	// RunID is the run identifier for tracking.
	RunID string `json:"runId,omitempty"`
	// ToolCallID is the tool call identifier when resuming.
	ToolCallID string `json:"toolCallId,omitempty"`
	// Models is the list of model configs for retry/fallback.
	// Excluded from ModelLoopStreamArgs (provided by MastraLLMVNext).
	Models any `json:"models,omitempty"`
	// Logger is the logger instance.
	Logger any `json:"logger,omitempty"`
	// Tools is the set of tools available for the loop.
	Tools ToolSet `json:"tools,omitempty"`
	// StopWhen is the stop condition for the loop.
	StopWhen any `json:"stopWhen,omitempty"`
	// ToolChoice controls tool selection. Default: "auto".
	ToolChoice string `json:"toolChoice,omitempty"`
	// ModelSettings holds model-specific settings (temperature, etc.).
	ModelSettings any `json:"modelSettings,omitempty"`
	// ProviderOptions holds provider-specific options.
	ProviderOptions any `json:"providerOptions,omitempty"`
	// Internal holds internal options.
	Internal any `json:"_internal,omitempty"`
	// StructuredOutput is the structured output schema.
	StructuredOutput any `json:"structuredOutput,omitempty"`
	// InputProcessors is the list of input processors.
	InputProcessors any `json:"inputProcessors,omitempty"`
	// OutputProcessors is the list of output processors.
	OutputProcessors []OutputProcessorOrWorkflow `json:"outputProcessors,omitempty"`
	// ReturnScorerData controls whether scorer data is returned.
	ReturnScorerData bool `json:"returnScorerData,omitempty"`
	// ModelSpanTracker tracks model spans for observability.
	ModelSpanTracker any `json:"modelSpanTracker,omitempty"`
	// RequireToolApproval controls whether tool calls require approval.
	RequireToolApproval any `json:"requireToolApproval,omitempty"`
	// ToolCallConcurrency controls the concurrency of tool calls.
	ToolCallConcurrency int `json:"toolCallConcurrency,omitempty"`
	// AgentID is the agent identifier.
	AgentID string `json:"agentId,omitempty"`
	// AgentName is the agent display name.
	AgentName string `json:"agentName,omitempty"`
	// RequestContext holds the request context.
	RequestContext RequestContext `json:"requestContext,omitempty"`
	// MethodType is the method type (generate or stream).
	MethodType ModelMethodType `json:"methodType,omitempty"`
	// IncludeRawChunks controls whether raw chunks are included.
	IncludeRawChunks bool `json:"includeRawChunks,omitempty"`
	// AutoResumeSuspendedTools controls auto-resumption of suspended tools.
	AutoResumeSuspendedTools bool `json:"autoResumeSuspendedTools,omitempty"`
	// MaxProcessorRetries is the max retries for processors.
	MaxProcessorRetries int `json:"maxProcessorRetries,omitempty"`
	// ProcessorStates holds processor state data.
	ProcessorStates any `json:"processorStates,omitempty"`
	// ActiveTools is the set of currently active tools.
	ActiveTools any `json:"activeTools,omitempty"`
	// IsTaskComplete is a function to check if the task is complete.
	IsTaskComplete any `json:"isTaskComplete,omitempty"`
	// OnIterationComplete is a callback for iteration completion.
	OnIterationComplete any `json:"onIterationComplete,omitempty"`
	// Workspace holds workspace configuration.
	Workspace any `json:"workspace,omitempty"`
	// Options holds additional streaming options (onStepFinish, onFinish).
	Options *ModelLoopStreamOptions `json:"options,omitempty"`
	// MaxSteps is the maximum number of steps.
	MaxSteps int `json:"maxSteps,omitempty"`
	// MessageList holds the agent message list.
	// Excluded from ModelLoopStreamArgs (provided by MastraLLMVNext).
	MessageList MessageList `json:"messageList,omitempty"`
}

// ModelLoopStreamOptions holds the streaming callback options within the loop.
type ModelLoopStreamOptions struct {
	// OnStepFinish is called when a step finishes.
	OnStepFinish func(props any) error `json:"-"`
	// OnFinish is called when the stream finishes.
	OnFinish func(props any) error `json:"-"`
}

// ---------------------------------------------------------------------------
// OriginalStreamTextOptions
// ---------------------------------------------------------------------------

// OriginalStreamTextOptions represents the options for the AI SDK streamText call.
// TS: Parameters<typeof streamText<TOOLS, inferOutput<Output>, DeepPartial<inferOutput<Output>>>>[0]
// In Go we use a struct capturing the relevant fields.
type OriginalStreamTextOptions struct {
	// Model is the language model to use.
	Model any `json:"model,omitempty"`
	// Messages is the list of messages.
	Messages any `json:"messages,omitempty"`
	// Tools is the set of tools.
	Tools ToolSet `json:"tools,omitempty"`
	// ToolChoice controls tool selection.
	ToolChoice string `json:"toolChoice,omitempty"`
	// MaxSteps is the maximum number of tool-use steps.
	MaxSteps int `json:"maxSteps,omitempty"`
	// OnStepFinish callback.
	OnStepFinish func(props any) error `json:"-"`
	// OnFinish callback.
	OnFinish func(props any) error `json:"-"`
}

// ---------------------------------------------------------------------------
// Callback types for model loop
// ---------------------------------------------------------------------------

// OriginalStreamTextOnFinishEventArg is the event argument for the original
// AI SDK stream text on-finish callback.
// TS: Parameters<OriginalStreamTextOnFinishCallback<Tools>>[0]
type OriginalStreamTextOnFinishEventArg struct {
	Text             string       `json:"text,omitempty"`
	FinishReason     string       `json:"finishReason,omitempty"`
	Usage            *TokenUsage  `json:"usage,omitempty"`
	TotalUsage       *TokenUsage  `json:"totalUsage,omitempty"`
	ToolCalls        any          `json:"toolCalls,omitempty"`
	ToolResults      any          `json:"toolResults,omitempty"`
	Response         *ResponseMeta `json:"response,omitempty"`
	Reasoning        any          `json:"reasoning,omitempty"`
	ReasoningText    string       `json:"reasoningText,omitempty"`
	Files            any          `json:"files,omitempty"`
	Sources          any          `json:"sources,omitempty"`
	Object           any          `json:"object,omitempty"`
	Warnings         []any        `json:"warnings,omitempty"`
	ProviderMetadata any          `json:"providerMetadata,omitempty"`
	Model            any          `json:"model,omitempty"`
}

// ModelLoopStreamTextOnFinishCallback is called when a model loop text stream finishes.
// TS: (event: OriginalStreamTextOnFinishEventArg<Tools> & { runId: string }) => Promise<void> | void
type ModelLoopStreamTextOnFinishCallback func(event ModelLoopFinishEvent) error

// ModelLoopFinishEvent extends the finish event with a RunID.
type ModelLoopFinishEvent struct {
	OriginalStreamTextOnFinishEventArg
	RunID string `json:"runId"`
}

// ModelLoopStreamTextOnStepFinishCallback is called when a model loop step finishes.
// TS: (event: Parameters<OriginalStreamTextOnStepFinishCallback<Tools>>[0] & { runId: string }) => Promise<void> | void
type ModelLoopStreamTextOnStepFinishCallback func(event ModelLoopStepFinishEvent) error

// ModelLoopStepFinishEvent extends the step finish event with a RunID.
type ModelLoopStepFinishEvent struct {
	StepFinishEvent
}

// ---------------------------------------------------------------------------
// ModelLoopStreamArgs
// ---------------------------------------------------------------------------

// ModelLoopStreamArgs holds the arguments for a model loop stream call.
// TS: ModelLoopStreamArgs<TOOLS extends ToolSet, OUTPUT = undefined>
// This is a combination of several fields from LoopOptions (minus 'models' and 'messageList')
// plus additional fields specific to the model loop.
type ModelLoopStreamArgs struct {
	// MethodType is the method type (generate or stream).
	MethodType ModelMethodType `json:"methodType"`
	// Messages is the optional list of messages.
	Messages any `json:"messages,omitempty"`
	// OutputProcessors is the list of output processors.
	OutputProcessors []OutputProcessorOrWorkflow `json:"outputProcessors,omitempty"`
	// RequestContext holds the request context.
	RequestCtx RequestContext `json:"requestContext,omitempty"`
	// ResourceID for resource-scoped operations.
	ResourceID string `json:"resourceId,omitempty"`
	// ThreadID for conversation threading.
	ThreadID string `json:"threadId,omitempty"`
	// ReturnScorerData controls whether scorer data is returned.
	ReturnScorerData bool `json:"returnScorerData,omitempty"`
	// MessageList holds the agent message list.
	MessageList MessageList `json:"messageList,omitempty"`

	// --- Fields from LoopOptions (minus 'models' and 'messageList') ---

	// ResumeContext holds context for resuming a suspended loop.
	ResumeContext any `json:"resumeContext,omitempty"`
	// RunID is the run identifier for tracking.
	RunID string `json:"runId,omitempty"`
	// ToolCallID is the tool call identifier when resuming.
	ToolCallID string `json:"toolCallId,omitempty"`
	// Tools is the set of tools available for the loop.
	Tools ToolSet `json:"tools,omitempty"`
	// StopWhen is the stop condition for the loop.
	StopWhen any `json:"stopWhen,omitempty"`
	// MaxSteps is the maximum number of steps.
	MaxSteps int `json:"maxSteps,omitempty"`
	// ToolChoice controls tool selection. Default: "auto".
	ToolChoice string `json:"toolChoice,omitempty"`
	// ModelSettings holds model-specific settings.
	ModelSettings any `json:"modelSettings,omitempty"`
	// ProviderOptions holds provider-specific options.
	ProviderOptions any `json:"providerOptions,omitempty"`
	// Internal holds internal options.
	Internal any `json:"_internal,omitempty"`
	// StructuredOutput is the structured output schema.
	StructuredOutput any `json:"structuredOutput,omitempty"`
	// InputProcessors is the list of input processors.
	InputProcessors any `json:"inputProcessors,omitempty"`
	// RequireToolApproval controls whether tool calls require approval.
	RequireToolApproval any `json:"requireToolApproval,omitempty"`
	// ToolCallConcurrency controls the concurrency of tool calls.
	ToolCallConcurrency int `json:"toolCallConcurrency,omitempty"`
	// AgentID is the agent identifier.
	AgentID string `json:"agentId,omitempty"`
	// AgentName is the agent display name.
	AgentName string `json:"agentName,omitempty"`
	// IncludeRawChunks controls whether raw chunks are included.
	IncludeRawChunks bool `json:"includeRawChunks,omitempty"`
	// AutoResumeSuspendedTools controls auto-resumption of suspended tools.
	AutoResumeSuspendedTools bool `json:"autoResumeSuspendedTools,omitempty"`
	// MaxProcessorRetries is the max retries for processors.
	MaxProcessorRetries int `json:"maxProcessorRetries,omitempty"`
	// ProcessorStates holds processor state data.
	ProcessorStates any `json:"processorStates,omitempty"`
	// ActiveTools is the set of currently active tools.
	ActiveTools any `json:"activeTools,omitempty"`
	// IsTaskComplete is a function to check if the task is complete.
	IsTaskComplete any `json:"isTaskComplete,omitempty"`
	// OnIterationComplete is a callback for iteration completion.
	OnIterationComplete any `json:"onIterationComplete,omitempty"`
	// Workspace holds workspace configuration.
	Workspace any `json:"workspace,omitempty"`
	// Options holds additional streaming options (onStepFinish, onFinish).
	Options *ModelLoopStreamOptions `json:"options,omitempty"`

	// --- ObservabilityContext fields ---
	ObservabilityContext
}

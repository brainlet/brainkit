// Ported from: packages/core/src/loop/types.ts
package loop

import (
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	model "github.com/brainlet/brainkit/agent-kit/core/llm/model"
	"github.com/brainlet/brainkit/agent-kit/core/memory"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	"github.com/brainlet/brainkit/agent-kit/core/processors"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/stream"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
	"github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workspace"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// LanguageModelV2 is a stub for @ai-sdk/provider-v5.LanguageModelV2.
// ai-kit only ported V3 (lm.LanguageModel). V2 stubs remain local.
// See brainlink/experiments/ai-kit/provider/languagemodel for V3.
type LanguageModelV2 interface{}

// ToolSet is a stub for @internal/ai-sdk-v5.ToolSet.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 ToolSet remains local.
// model.ToolSet = map[string]Tool where Tool = any — same shape but different package context.
type ToolSet map[string]any

// ToolChoice is a stub for @internal/ai-sdk-v5.ToolChoice.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 ToolChoice remains local.
// processors.ToolChoice = any, but loop uses string type. Shape mismatch.
type ToolChoice string

// CallSettings is a stub for @internal/ai-sdk-v5.CallSettings.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 CallSettings remains local.
// processors.CallSettings has AbortSignal as context.Context; loop uses any. Shape mismatch.
type CallSettings struct {
	AbortSignal any `json:"abortSignal,omitempty"`
}

// IdGenerator is a stub for @internal/ai-sdk-v5.IdGenerator.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 IdGenerator remains local.
type IdGenerator func() string

// StopCondition is a stub for @internal/ai-sdk-v5.StopCondition / @internal/ai-v6.StopCondition.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5/V6 StopCondition remains local.
type StopCondition interface{}

// IsTaskCompleteConfig is a stub for ../agent/agent.types.IsTaskCompleteConfig.
// Stub: real agent.IsTaskCompleteConfig (= CompletionConfig) has Scorers, Strategy, Timeout etc.
// Importing agent would create import cycle risk if agent imports loop. Kept as empty struct.
type IsTaskCompleteConfig struct{}

// OnIterationCompleteHandler is a stub for ../agent/agent.types.OnIterationCompleteHandler.
// Stub: signature mismatch — real type is func(ctx IterationCompleteContext) (*IterationCompleteResult, error),
// loop uses func(args any) error. Wiring requires updating all call sites.
type OnIterationCompleteHandler func(args any) error

// MessageInput is a stub for ../agent/message-list.MessageInput.
// Stub: real agent.MessageInput is = any. Same shape; kept local to avoid agent dependency.
type MessageInput = any

// MessageList is a stub for ../agent/message-list.MessageList.
// Stub: real agent.MessageList is struct{} with methods. Importing agent could create cycle.
type MessageList interface{}

// SaveQueueManager is a stub for ../agent/save-queue.SaveQueueManager.
// Stub: real type lives in agent package. Importing agent could create cycle.
type SaveQueueManager interface{}

// StructuredOutputOptions is a stub for ../agent/types.StructuredOutputOptions.
// Stub: real agent.StructuredOutputOptions has Schema, Model, JSONPromptInjection fields.
// Importing agent could create cycle. Kept as empty struct.
type StructuredOutputOptions struct{}

// ModelRouterModelId is imported from llm/model.
type ModelRouterModelId = model.ModelRouterModelID

// ModelMethodType is imported from llm/model.
type ModelMethodType = model.ModelMethodType

// MastraLanguageModelV2 is imported from llm/model.
type MastraLanguageModelV2 = model.MastraLanguageModelV2

// OpenAICompatibleConfig is imported from llm/model.
type OpenAICompatibleConfig = model.OpenAICompatibleConfig

// SharedProviderOptions is imported from llm/model.
type SharedProviderOptions = model.SharedProviderOptions

// IMastraLogger is imported from logger.
type IMastraLogger = logger.IMastraLogger

// Mastra represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core imports loop (indirectly via agent), so loop cannot import core.
// core.Mastra struct satisfies this interface.
// The loop passes this through to sub-components (network, workflows) for
// logging, storage access, and workflow registration.
type Mastra interface {
	// GetLogger returns the configured logger instance.
	GetLogger() IMastraLogger
}

// MastraMemory is a stub for ../memory.MastraMemory.
// Cannot import memory directly: core.Mastra references memory types and agent references
// memory, creating potential circular dependencies through the loop package.
// The loop package only stores and passes MastraMemory through (via StreamInternal.Memory).
// The network sub-package defines its own richer Memory interface for its call patterns.
// These methods represent the minimum useful contract for pass-through identification.
type MastraMemory interface {
	// ID returns the unique identifier for this memory instance.
	ID() string
	// GetMergedThreadConfig merges the given config with the base thread config.
	GetMergedThreadConfig(config *MemoryConfig) MemoryConfig
}

// MemoryConfig is imported from memory.
type MemoryConfig = memory.MemoryConfig

// IModelSpanTracker is imported from observability/types.
type IModelSpanTracker = obstypes.IModelSpanTracker

// ObservabilityContext is imported from observability/types.
type ObservabilityContext = obstypes.ObservabilityContext

// InputProcessorOrWorkflow is a stub for ../processors.InputProcessorOrWorkflow.
// Stub: type not defined in processors package — only ProcessorOrWorkflow exists there
// with a marker method. This parallel-stubs type represents the union concept locally.
type InputProcessorOrWorkflow interface{}

// OutputProcessorOrWorkflow is a stub for ../processors.OutputProcessorOrWorkflow.
// Stub: type not defined in processors package — only ProcessorOrWorkflow exists there
// with a marker method. This parallel-stubs type represents the union concept locally.
type OutputProcessorOrWorkflow interface{}

// ProcessInputStepArgs is imported from processors.
type ProcessInputStepArgs = processors.ProcessInputStepArgs

// ProcessInputStepResult is imported from processors.
type ProcessInputStepResult = processors.ProcessInputStepResult

// ProcessorState is = any in both loop and processors. Same shape, no import needed.
type ProcessorState = any

// RequestContext is imported from requestcontext.
type RequestContext = requestcontext.RequestContext

// ChunkType is imported from stream.
type ChunkType = stream.ChunkType

// MastraOnFinishCallback is a stub for ../stream/types.MastraOnFinishCallback.
// Stub: signature mismatch — real type is func(event stream.MastraOnFinishCallbackArgs) error,
// but loop code uses func(result any) error throughout. Wiring requires updating all call sites.
type MastraOnFinishCallback func(result any) error

// MastraOnStepFinishCallback is a stub for ../stream/types.MastraOnStepFinishCallback.
// Stub: signature mismatch — real type is func(event stream.MastraOnStepFinishEvent) error,
// but loop code uses func(result any) error throughout. Wiring requires updating all call sites.
type MastraOnStepFinishCallback func(result any) error

// ModelManagerModelConfig is a stub for ../stream/types.ModelManagerModelConfig.
// Stub: shape mismatch — real type has flat Model field (MastraLanguageModel interface),
// but loop uses nested anonymous struct with ModelID/Provider/SpecificationVersion strings.
// Wiring requires updating all construction/access sites.
type ModelManagerModelConfig struct {
	Model struct {
		ModelID              string `json:"modelId"`
		Provider             string `json:"provider"`
		SpecificationVersion string `json:"specificationVersion,omitempty"`
	} `json:"model"`
}

// StreamTransportRef is imported from stream.
type StreamTransportRef = stream.StreamTransportRef

// MastraIdGenerator is imported from types.
// Note: real type takes *IdGeneratorContext (pointer), stub was value receiver.
type MastraIdGenerator = aktypes.MastraIdGenerator

// IdGeneratorContext is imported from types.
// Note: real type has richer fields (IdType, Source as typed enums, optional pointer fields).
type IdGeneratorContext = aktypes.IdGeneratorContext

// OutputWriter is imported from workflows.
type OutputWriter = workflows.OutputWriter

// Workspace is imported from workspace.
// Note: real type is a struct, not interface. Used as pointer (*Workspace) in field types.
type Workspace = workspace.Workspace

// ---------------------------------------------------------------------------
// PrimitiveType enum
// ---------------------------------------------------------------------------

// PrimitiveType enumerates the valid primitive types.
type PrimitiveType string

const (
	PrimitiveTypeAgent    PrimitiveType = "agent"
	PrimitiveTypeWorkflow PrimitiveType = "workflow"
	PrimitiveTypeNone     PrimitiveType = "none"
	PrimitiveTypeTool     PrimitiveType = "tool"
)

// PrimitiveTypes lists all valid primitive types (mirrors TS z.enum).
var PrimitiveTypes = []PrimitiveType{
	PrimitiveTypeAgent,
	PrimitiveTypeWorkflow,
	PrimitiveTypeNone,
	PrimitiveTypeTool,
}

// ---------------------------------------------------------------------------
// StreamInternal
// ---------------------------------------------------------------------------

// StreamInternal holds internal state passed through the loop for memory,
// ID generation, time, and delegation control.
type StreamInternal struct {
	Now              func() time.Time  `json:"-"`
	GenerateID       IdGenerator       `json:"-"`
	CurrentDate      func() time.Time  `json:"-"`
	SaveQueueManager SaveQueueManager  `json:"-"`
	MemoryConfig     *MemoryConfig     `json:"memoryConfig,omitempty"`
	ThreadID         string            `json:"threadId,omitempty"`
	ResourceID       string            `json:"resourceId,omitempty"`
	Memory           MastraMemory      `json:"-"`
	ThreadExists     bool              `json:"threadExists,omitempty"`
	StepTools        ToolSet           `json:"-"`
	StepWorkspace    Workspace         `json:"-"`
	DelegationBailed bool              `json:"_delegationBailed,omitempty"`
	TransportRef     StreamTransportRef `json:"-"`
}

// ---------------------------------------------------------------------------
// PrepareStepResult
// ---------------------------------------------------------------------------

// PrepareStepResult describes the output of a PrepareStepFunction.
// Any field may be nil/zero to indicate "no change".
type PrepareStepResult struct {
	// Model can be a LanguageModelV2, ModelRouterModelId string,
	// OpenAICompatibleConfig, or MastraLanguageModelV2.
	Model       any            `json:"model,omitempty"`
	ToolChoice  ToolChoice     `json:"toolChoice,omitempty"`
	ActiveTools []string       `json:"activeTools,omitempty"`
	Messages    []MessageInput `json:"messages,omitempty"`
	// Workspace to use for this step. When provided, this workspace will be
	// passed to tool execution context, allowing tools to access
	// workspace.filesystem and workspace.sandbox.
	Workspace Workspace `json:"-"`
}

// ---------------------------------------------------------------------------
// PrepareStepFunction
// ---------------------------------------------------------------------------

// PrepareStepFunction is called before each step of multi-step execution.
type PrepareStepFunction func(args ProcessInputStepArgs) (*ProcessInputStepResult, error)

// ---------------------------------------------------------------------------
// LoopConfig
// ---------------------------------------------------------------------------

// LoopConfig holds per-invocation callbacks and settings.
type LoopConfig struct {
	OnChunk      func(chunk ChunkType) error `json:"-"`
	OnError      func(err error) error       `json:"-"`
	OnFinish     MastraOnFinishCallback       `json:"-"`
	OnStepFinish MastraOnStepFinishCallback   `json:"-"`
	OnAbort      func(event any) error        `json:"-"`
	// AbortSignal is not directly portable; use context.Context cancellation in Go.
	// Kept as a placeholder for API parity.
	AbortSignal      any               `json:"-"`
	ReturnScorerData bool              `json:"returnScorerData,omitempty"`
	PrepareStep      PrepareStepFunction `json:"-"`
}

// ---------------------------------------------------------------------------
// LoopOptions
// ---------------------------------------------------------------------------

// LoopOptions captures everything needed to start a loop execution.
type LoopOptions struct {
	Mastra         Mastra          `json:"-"`
	ResumeContext  *ResumeContext   `json:"resumeContext,omitempty"`
	ToolCallID     string           `json:"toolCallId,omitempty"`
	Models         []ModelManagerModelConfig `json:"models"`
	Logger         IMastraLogger    `json:"-"`
	Mode           string           `json:"mode,omitempty"` // "generate" | "stream"
	RunID          string           `json:"runId,omitempty"`
	IDGenerator    MastraIdGenerator `json:"-"`
	ToolCallStreaming bool          `json:"toolCallStreaming,omitempty"`
	MessageList    MessageList      `json:"-"`
	IncludeRawChunks bool          `json:"includeRawChunks,omitempty"`
	ModelSettings  *CallSettings    `json:"modelSettings,omitempty"`
	ToolChoice     ToolChoice       `json:"toolChoice,omitempty"`
	ActiveTools    []string         `json:"activeTools,omitempty"`
	Options        *LoopConfig      `json:"-"`
	ProviderOptions SharedProviderOptions `json:"providerOptions,omitempty"`
	OutputProcessors []OutputProcessorOrWorkflow `json:"-"`
	InputProcessors  []InputProcessorOrWorkflow  `json:"-"`
	Tools          ToolSet          `json:"tools,omitempty"`
	ExperimentalGenerateMessageID func() string `json:"-"`
	StopWhen       []StopCondition  `json:"stopWhen,omitempty"`
	MaxSteps       int              `json:"maxSteps,omitempty"`
	Internal       *StreamInternal  `json:"-"`
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
	ReturnScorerData bool          `json:"returnScorerData,omitempty"`
	DownloadRetries  int           `json:"downloadRetries,omitempty"`
	DownloadConcurrency int        `json:"downloadConcurrency,omitempty"`
	ModelSpanTracker IModelSpanTracker `json:"-"`
	RequireToolApproval bool       `json:"requireToolApproval,omitempty"`
	AutoResumeSuspendedTools bool  `json:"autoResumeSuspendedTools,omitempty"`
	AgentID        string           `json:"agentId"`
	ToolCallConcurrency int        `json:"toolCallConcurrency,omitempty"`
	AgentName      string           `json:"agentName,omitempty"`
	RequestContext *RequestContext  `json:"-"`
	MethodType     ModelMethodType  `json:"methodType"`
	// MaxProcessorRetries is the maximum number of times processors can
	// trigger a retry per generation. When a processor calls abort({retry: true}),
	// the agent will retry with feedback. If not set, no retries are performed.
	MaxProcessorRetries int        `json:"maxProcessorRetries,omitempty"`
	// IsTaskComplete is the scoring configuration for supervisor patterns.
	// Scorers evaluate whether the task is complete after each iteration.
	IsTaskComplete *IsTaskCompleteConfig `json:"isTaskComplete,omitempty"`
	// OnIterationComplete is a callback fired after each iteration completes.
	// Allows monitoring and controlling iteration flow with feedback.
	OnIterationComplete OnIterationCompleteHandler `json:"-"`
	// Workspace is the default workspace for the agent. This workspace will
	// be passed to tool execution context unless overridden by prepareStep
	// or processInputStep.
	Workspace Workspace `json:"-"`
	// ProcessorStates is a shared processor state map that persists across
	// loop iterations. Used by all processor methods (input and output) to
	// share state. Keyed by processor ID.
	ProcessorStates map[string]ProcessorState `json:"-"`
	// ObservabilityContext is embedded observability context (tracing, etc.).
	ObservabilityContext
}

// ResumeContext holds data needed to resume a suspended loop.
type ResumeContext struct {
	ResumeData any `json:"resumeData"`
	Snapshot   any `json:"snapshot"`
}

// ---------------------------------------------------------------------------
// LoopRun
// ---------------------------------------------------------------------------

// LoopRun extends LoopOptions with runtime state for an active loop execution.
type LoopRun struct {
	LoopOptions
	MessageID      string          `json:"messageId"`
	RunID          string          `json:"runId"`
	StartTimestamp int64           `json:"startTimestamp"`
	Internal       *StreamInternal `json:"-"`
	StreamState    StreamState     `json:"-"`
	MethodType     ModelMethodType `json:"methodType"`
}

// StreamState provides serialization/deserialization for model output state.
type StreamState struct {
	Serialize   func() any       `json:"-"`
	Deserialize func(state any)  `json:"-"`
}

// ---------------------------------------------------------------------------
// OuterLLMRun
// ---------------------------------------------------------------------------

// OuterLLMRun extends LoopRun with controller and writer for the outer
// streaming layer.
type OuterLLMRun struct {
	LoopRun
	MessageID    string       `json:"messageId"`
	// Controller is the writable side of the ReadableStream.
	// In Go this maps to a channel or writer interface.
	Controller   any          `json:"-"`
	OutputWriter OutputWriter `json:"-"`
}

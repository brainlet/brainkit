// Ported from: packages/core/src/agent/types.ts
package agent

import (
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist"
	"github.com/brainlet/brainkit/agent-kit/core/llm"
	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/memory"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/tools"
	"github.com/brainlet/brainkit/agent-kit/core/voice"
	"github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workspace"
	"github.com/brainlet/brainkit/agent-kit/core/workspace/skills"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// ToolSet is a stub for @internal/ai-sdk-v4.ToolSet.
// MISMATCH: real tools.ToolSet is map[string]ToolRef (not map[string]any).
// Cannot wire because agent code constructs ToolSet with any values, not ToolRef.
type ToolSet map[string]any

// GenerateTextOnStepFinishCallback is a stub for @internal/ai-sdk-v4.GenerateTextOnStepFinishCallback.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type GenerateTextOnStepFinishCallback func(result any) error

// ProviderDefinedTool is a stub for @internal/external-types.ProviderDefinedTool.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type ProviderDefinedTool = any

// JSONSchema7 is a stub for json-schema.JSONSchema7.
// MISMATCH: real stream/base.JSONSchema7 is a struct with 20+ fields (Schema, Type,
// Properties, Required, etc.). Agent code uses JSONSchema7 = any for opaque passing.
// Cannot wire without updating all call sites to use the real struct.
type JSONSchema7 = any

// ZodSchema is a stub for zod.ZodSchema.
// NOT PORTED: Zod is a TypeScript-specific runtime validation library with no Go equivalent.
// Kept as = any. Go code should use standard JSON Schema or struct-based validation.
type ZodSchema = any

// MastraScorer is a stub for ../evals.MastraScorer.
// MISMATCH: real evals.MastraScorer is a concrete struct (not interface) with
// ID()/Name() methods. The interface stub here has the right methods but wrong
// kind (interface vs struct). Cannot wire because Go type aliases don't support
// interface-to-struct aliasing.
type MastraScorer interface {
	ID() string
	Name() string
}

// MastraScorers is a stub for ../evals.MastraScorers.
// MISMATCH: real evals.MastraScorers is map[string]MastraScorerEntry (not
// []MastraScorer). Cannot wire until agent code is updated to use map-based
// scorer lookup instead of slice iteration.
type MastraScorers = []MastraScorer

// ScoringSamplingConfig is a stub for ../evals.ScoringSamplingConfig.
// MISMATCH: real evals.ScoringSamplingConfig has Type (ScoringSamplingConfigType)
// and Rate (float64) fields, not SampleSize (int) and Percentage (float64).
// Cannot wire until agent code is updated to use the real field names.
type ScoringSamplingConfig struct {
	SampleSize int     `json:"sampleSize,omitempty"`
	Percentage float64 `json:"percentage,omitempty"`
}

// CoreMessage is re-exported from llm/model.
type CoreMessage = model.CoreMessage

// SystemMessage is re-exported from llm.
type SystemMessage = llm.SystemMessage

// MastraModelConfig is re-exported from llm/model.
type MastraModelConfig = model.MastraModelConfig

// OpenAICompatibleConfig is re-exported from llm/model.
type OpenAICompatibleConfig = model.OpenAICompatibleConfig

// OutputType is re-exported from llm.
type OutputType = llm.OutputType

// DefaultLLMTextOptions is a stub for ../llm.DefaultLLMTextOptions.
// NOT PORTED: these option types don't exist in the llm package. They come from
// the TypeScript generic defaults and have no Go equivalent yet.
type DefaultLLMTextOptions struct{}

// DefaultLLMTextObjectOptions is a stub for ../llm.DefaultLLMTextObjectOptions.
// NOT PORTED: see DefaultLLMTextOptions.
type DefaultLLMTextObjectOptions struct{}

// DefaultLLMStreamOptions is a stub for ../llm.DefaultLLMStreamOptions.
// NOT PORTED: see DefaultLLMTextOptions.
type DefaultLLMStreamOptions struct{}

// DefaultLLMStreamObjectOptions is a stub for ../llm.DefaultLLMStreamObjectOptions.
// NOT PORTED: see DefaultLLMTextOptions.
type DefaultLLMStreamObjectOptions struct{}

// ModelRouterModelID is re-exported from llm/model.
type ModelRouterModelID = model.ModelRouterModelID

// StreamTextOnFinishCallback is re-exported from llm/model.
type StreamTextOnFinishCallback = model.StreamTextOnFinishCallback

// StreamTextOnStepFinishCallback is re-exported from llm/model.
type StreamTextOnStepFinishCallback = model.StreamTextOnStepFinishCallback

// StreamObjectOnFinishCallback is re-exported from llm/model.
type StreamObjectOnFinishCallback = model.StreamObjectOnFinishCallback

// ProviderOptions is a stub for ../llm/model/provider-options.ProviderOptions.
// MISMATCH: real model.ProviderOptions is a struct with named fields (Anthropic, DeepSeek,
// Google, OpenAI, XAI, Extra), not map[string]map[string]any. Cannot wire until
// all agent code is updated to use the struct-based API.
type ProviderOptions = map[string]map[string]any

// IMastraLogger is a re-export reference for ../logger.IMastraLogger.
type IMastraLogger = logger.IMastraLogger

// Mastra represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core imports agent (registers agents, calls agent methods), so agent cannot import core.
// core.Mastra struct satisfies this interface.
// Agent stores this reference and passes it to dynamic resolver functions
// (instructions, tools, workflows, scorers) so they can access framework services.
// Agent also accesses GetLogger for workspace logger propagation and GetStorage
// for workflow persistence.
type Mastra interface {
	// GetLogger returns the configured logger instance.
	GetLogger() IMastraLogger
}

// MastraMemory is re-exported from memory.
type MastraMemory = memory.MastraMemory

// MemoryConfig is re-exported from memory.
type MemoryConfig = memory.MemoryConfig

// StorageThreadType is re-exported from memory.
type StorageThreadType = memory.StorageThreadType

// ObservabilityContext is re-exported from observability/types.
type ObservabilityContext = obstypes.ObservabilityContext

// TracingOptions is re-exported from observability/types.
type TracingOptions = obstypes.TracingOptions

// TracingPolicy is re-exported from observability/types.
type TracingPolicy = obstypes.TracingPolicy

// Span is re-exported from observability/types.
type Span = obstypes.Span

// SpanType is re-exported from observability/types.
type SpanType = obstypes.SpanType

// InputProcessorOrWorkflow is a stub for ../processors.InputProcessorOrWorkflow.
// NOT DEFINED in processors package — real ProcessorRunner uses []any for input/output
// processor slices. This stub serves as a documentation type; real values are
// processors.InputProcessor or processors.ProcessorWorkflow.
type InputProcessorOrWorkflow interface{}

// OutputProcessorOrWorkflow is a stub for ../processors.OutputProcessorOrWorkflow.
// NOT DEFINED in processors package — real ProcessorRunner uses []any for input/output
// processor slices. This stub serves as a documentation type; real values are
// processors.OutputProcessor or processors.ProcessorWorkflow.
type OutputProcessorOrWorkflow interface{}

// ProcessorWorkflow is a stub for ../processors.ProcessorWorkflow.
// MISMATCH: real processors.ProcessorWorkflow is an interface with Run(context.Context,
// ProcessorContext, any) (any, error) + Processor methods. Cannot wire as alias because
// stub is used as interface{} and real interface has required methods.
type ProcessorWorkflow interface{}

// Processor is a stub for ../processors.Processor.
// MISMATCH: real processors.Processor interface has ID() (not GetID()), plus Name(),
// Description(), ProcessorIndex(), SetProcessorIndex(int) methods. Cannot wire until
// agent code is updated to use the real interface method names.
type Processor interface {
	GetID() string
}

// ProcessorRunner is a stub for ../processors/runner.ProcessorRunner.
// MISMATCH: real processors.ProcessorRunner has private fields (logger, agentName,
// processorStates sync.Map) and uses []any for processor slices instead of typed interfaces.
// Cannot use type alias because of private fields. Use processors.ProcessorRunner directly
// where the real type is needed.
type ProcessorRunner struct {
	InputProcessors  []InputProcessorOrWorkflow
	OutputProcessors []OutputProcessorOrWorkflow
}

// ProcessorState is a stub for ../processors/runner.ProcessorState.
// MISMATCH: real processors.ProcessorState is a struct with sync.Mutex, accumulated
// text fields, StreamParts, and Span. Cannot use type alias because it's a concrete
// struct with private fields and sync primitives.
type ProcessorState = any

// DefaultVoice is re-exported from voice.
type DefaultVoice = voice.DefaultVoice

// RequestContext is a re-export reference for ../request-context.RequestContext.
type RequestContext = requestcontext.RequestContext

// OutputSchema is a stub for ../stream.OutputSchema.
// Both stream/base.OutputSchema and stream/types.OutputSchema are = any.
// No value in wiring an import just for type alias = any.
type OutputSchema = any

// ModelManagerModelConfig is a stub for ../stream/types.ModelManagerModelConfig.
// MISMATCH: real stream.ModelManagerModelConfig has Model typed as MastraLanguageModel
// (interface with SpecificationVersion/Provider/ModelID methods), while this stub
// uses Model any. Cannot wire until agent code is updated to pass real
// MastraLanguageModel values instead of any.
type ModelManagerModelConfig struct {
	ID         string            `json:"id"`
	Model      any               `json:"model"` // MastraLanguageModel
	MaxRetries int               `json:"maxRetries"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// ToolAction is re-exported from tools.
type ToolAction = tools.ToolAction

// VercelTool is re-exported from tools.
type VercelTool = tools.VercelTool

// VercelToolV5 is re-exported from tools.
type VercelToolV5 = tools.VercelToolV5

// DynamicArgument is a stub for ../types.DynamicArgument.
// MISMATCH: real types.DynamicArgument[T] is a generic struct with static/resolver/isDynamic
// fields. Cannot alias a generic struct to = any. Agent code uses DynamicArgument = any
// for opaque storage of values that may be static or dynamic (function-based).
type DynamicArgument = any

// MastraVoice is re-exported from voice.
type MastraVoice = voice.MastraVoice

// Workflow is re-exported from workflows.
type Workflow = *workflows.Workflow

// AnyWorkspace is re-exported from workspace.
type AnyWorkspace = workspace.AnyWorkspace

// SkillFormat is re-exported from workspace/skills.
type SkillFormat = skills.SkillFormat

// OutputWriter is re-exported from workflows.
type OutputWriter = workflows.OutputWriter

// ---------------------------------------------------------------------------
// Re-exported types from message-list (stubbed)
// ---------------------------------------------------------------------------

// MastraDBMessage represents a stored conversation message.
// MISMATCH: real mlstate.MastraDBMessage embeds MastraMessageShared (ID, Role,
// CreatedAt, ThreadID, ResourceID, Type) + Content field. This stub has those fields
// directly. Cannot wire because struct literal construction in test_utils.go uses
// direct field assignment (e.g., MastraDBMessage{ID: ..., Role: ...}) which won't
// compile with embedded struct. Requires refactoring all constructors to use
// embedded struct syntax (e.g., MastraMessageShared: mlstate.MastraMessageShared{...}).
type MastraDBMessage struct {
	ID         string                 `json:"id"`
	Role       string                 `json:"role"`
	Content    MastraMessageContentV2 `json:"content"`
	CreatedAt  time.Time              `json:"createdAt"`
	ThreadID   string                 `json:"threadId,omitempty"`
	ResourceID string                 `json:"resourceId,omitempty"`
	Type       string                 `json:"type,omitempty"`
}

// MastraMessageContentV2 represents v2 message content.
// MISMATCH: real mlstate.MastraMessageContentV2 has additional fields
// (ExperimentalAttachments, ToolInvocations, Reasoning, Annotations, ProviderMetadata
// as map[string]map[string]any). Stub has fewer fields and ProviderMetadata as
// map[string]any. Cannot wire until struct literal constructors are updated.
type MastraMessageContentV2 struct {
	Format           int            `json:"format"`
	Parts            []MastraMessagePart `json:"parts"`
	Content          string         `json:"content,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
}

// MastraMessagePart represents a single part within a message.
// MISMATCH: real mlstate.MastraMessagePart has 12+ fields (Reasoning, Details,
// Data, MimeType, Filename, Source, ProviderMetadata, ProviderExecuted, Metadata,
// DataPayload) beyond the 3 in this stub. Cannot wire until struct literal
// constructors in test_utils.go are updated for the richer type.
type MastraMessagePart struct {
	Type           string         `json:"type"`
	Text           string         `json:"text,omitempty"`
	ToolInvocation *ToolInvocation `json:"toolInvocation,omitempty"`
}

// ToolInvocation represents a tool invocation within a message part.
// MISMATCH: real mlstate.ToolInvocation has additional Step *int field beyond
// the 5 fields in this stub. Struct literal constructors in test_utils.go
// would compile (extra fields just zero-valued), but keeping stub for
// consistency with other message-list types until batch migration.
type ToolInvocation struct {
	State      string         `json:"state"`
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Args       map[string]any `json:"args,omitempty"`
	Result     any            `json:"result,omitempty"`
}

// UIMessageWithMetadata is a stub for message-list.UIMessageWithMetadata.
// Real mlstate.UIMessageWithMetadata is a struct with 10+ fields. Kept as any
// since this is an alias type and changing to the struct would require updating
// all callers. Wire when message-list types are batch-migrated.
type UIMessageWithMetadata = any

// MessageList is re-exported from agent/messagelist.
type MessageList = messagelist.MessageList

// MessageListInput is re-exported from agent/messagelist.
type MessageListInput = messagelist.MessageListInput

// AiMessageType is a re-export of @internal/ai-sdk-v4.Message.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type AiMessageType = any

// LLMStepResult is a stub for ../stream/base.LLMStepResult (also in stream/types).
// Real LLMStepResult is a struct with 15+ typed fields (StepType, Text, ToolCalls, etc.).
// Kept as = any because agent code uses it opaquely (never accesses fields directly).
type LLMStepResult = any

// ---------------------------------------------------------------------------
// Ported types
// ---------------------------------------------------------------------------

// ToolsInput accepts Mastra tools, Vercel AI SDK tools, and provider-defined tools
// (e.g., google.tools.googleSearch()).
type ToolsInput map[string]any

// AgentInstructions is an alias for SystemMessage.
type AgentInstructions = SystemMessage

// ToolsetsInput is a map of named tool sets.
type ToolsetsInput map[string]ToolsInput

// ---------------------------------------------------------------------------
// StructuredOutputOptions
// ---------------------------------------------------------------------------

// StructuredOutputOptionsBase holds base options for structured output.
type StructuredOutputOptionsBase struct {
	// Instructions is custom instructions for the structuring agent.
	// If not provided, will generate instructions based on the schema.
	Instructions string `json:"instructions,omitempty"`

	// JSONPromptInjection indicates whether to use system prompt injection
	// instead of native response format to coerce the LLM to respond with JSON text.
	JSONPromptInjection bool `json:"jsonPromptInjection,omitempty"`

	// Logger is optional logger instance for structured logging.
	Logger IMastraLogger `json:"-"`

	// ProviderOptions holds provider-specific options passed to the internal structuring agent.
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`

	// ErrorStrategy controls how errors are handled: "strict", "warn", or "fallback".
	// When "fallback", FallbackValue must be set.
	ErrorStrategy string `json:"errorStrategy,omitempty"`

	// FallbackValue is the value to use when ErrorStrategy is "fallback".
	FallbackValue any `json:"fallbackValue,omitempty"`
}

// StructuredOutputOptions holds options for structured output generation.
type StructuredOutputOptions struct {
	StructuredOutputOptionsBase

	// Schema is the Zod/JSON schema to validate the output against.
	Schema OutputSchema `json:"schema"`

	// Model to use for the internal structuring agent.
	// If not provided, falls back to the agent's model.
	Model MastraModelConfig `json:"model,omitempty"`
}

// SerializableStructuredOutputOptions is a serializable variant of StructuredOutputOptions.
type SerializableStructuredOutputOptions struct {
	StructuredOutputOptionsBase

	// Model can be a ModelRouterModelID string or OpenAICompatibleConfig.
	Model any `json:"model,omitempty"`

	// Schema is the Zod/JSON schema to validate the output against.
	Schema OutputSchema `json:"schema"`
}

// ---------------------------------------------------------------------------
// AgentCreateOptions
// ---------------------------------------------------------------------------

// AgentCreateOptions provides options while creating an agent.
type AgentCreateOptions struct {
	TracingPolicy *TracingPolicy `json:"tracingPolicy,omitempty"`
}

// ---------------------------------------------------------------------------
// ModelWithRetries
// ---------------------------------------------------------------------------

// ModelWithRetries wraps a model config with retry and enabled settings.
type ModelWithRetries struct {
	// ID is an optional identifier for this model config.
	ID string `json:"id,omitempty"`
	// Model is the model configuration or dynamic resolver function.
	Model any `json:"model"`
	// MaxRetries defaults to 0.
	MaxRetries *int `json:"maxRetries,omitempty"`
	// Enabled defaults to true.
	Enabled *bool `json:"enabled,omitempty"`
}

// ---------------------------------------------------------------------------
// AgentConfig
// ---------------------------------------------------------------------------

// AgentConfig holds the full configuration for creating an Agent.
// In TypeScript this was generic over TAgentId, TTools, TOutput, TRequestContext.
// In Go we collapse all generics to concrete types or any.
type AgentConfig struct {
	// ID uniquely identifies the agent.
	ID string `json:"id"`
	// Name is the display name for the agent.
	Name string `json:"name"`
	// Description of the agent's purpose and capabilities.
	Description string `json:"description,omitempty"`

	// Instructions that guide the agent's behavior.
	// Can be a static SystemMessage or a DynamicArgument resolver.
	Instructions DynamicArgument `json:"instructions"`

	// Model is the language model used by the agent.
	// Can be a MastraModelConfig, DynamicModel func, or []ModelWithRetries.
	Model any `json:"model"`

	// MaxRetries for model calls in case of failure. Default: 0.
	MaxRetries *int `json:"maxRetries,omitempty"`

	// Tools that the agent can access. Can be static or dynamic.
	Tools DynamicArgument `json:"tools,omitempty"`

	// Workflows that the agent can execute. Can be static or dynamic.
	Workflows DynamicArgument `json:"workflows,omitempty"`

	// DefaultGenerateOptionsLegacy is the default options for generate() calls.
	// Deprecated: use DefaultOptions.
	DefaultGenerateOptionsLegacy DynamicArgument `json:"defaultGenerateOptionsLegacy,omitempty"`

	// DefaultStreamOptionsLegacy is the default options for stream() calls.
	// Deprecated: use DefaultOptions.
	DefaultStreamOptionsLegacy DynamicArgument `json:"defaultStreamOptionsLegacy,omitempty"`

	// DefaultOptions is the default options for stream()/generate() in vNext mode.
	DefaultOptions DynamicArgument `json:"defaultOptions,omitempty"`

	// DefaultNetworkOptions is the default options for network() calls.
	DefaultNetworkOptions DynamicArgument `json:"defaultNetworkOptions,omitempty"`

	// Mastra is a reference to the Mastra runtime instance (injected automatically).
	Mastra Mastra `json:"-"`

	// Agents are sub-agents that the agent can access. Can be static or dynamic.
	Agents DynamicArgument `json:"agents,omitempty"`

	// Scorers for runtime evaluation and observability. Can be static or dynamic.
	Scorers DynamicArgument `json:"scorers,omitempty"`

	// Memory module used for storing and retrieving stateful context.
	Memory DynamicArgument `json:"memory,omitempty"`

	// SkillsFormat for skill information injection when workspace has skills.
	// Default: "xml".
	SkillsFormat SkillFormat `json:"skillsFormat,omitempty"`

	// Voice settings for speech input and output.
	Voice MastraVoice `json:"-"`

	// Workspace for file storage and code execution.
	Workspace DynamicArgument `json:"workspace,omitempty"`

	// InputProcessors that can modify or validate messages before agent processing.
	InputProcessors DynamicArgument `json:"inputProcessors,omitempty"`

	// OutputProcessors that can modify or validate messages from the agent.
	OutputProcessors DynamicArgument `json:"outputProcessors,omitempty"`

	// MaxProcessorRetries is the maximum number of times processors can trigger
	// a retry per generation. Default: no retries.
	MaxProcessorRetries *int `json:"maxProcessorRetries,omitempty"`

	// Options to pass to the agent upon creation.
	Options *AgentCreateOptions `json:"options,omitempty"`

	// RawConfig is the raw storage configuration this agent was created from.
	RawConfig map[string]any `json:"rawConfig,omitempty"`

	// RequestContextSchema is an optional schema for validating request context values.
	RequestContextSchema any `json:"requestContextSchema,omitempty"`
}

// ---------------------------------------------------------------------------
// AgentMemoryOption
// ---------------------------------------------------------------------------

// AgentMemoryOption configures memory for an agent execution.
type AgentMemoryOption struct {
	// Thread can be a thread ID string or a partial StorageThreadType with required ID.
	Thread any `json:"thread"`
	// Resource is the resource identifier.
	Resource string `json:"resource"`
	// Options holds optional memory configuration.
	Options *MemoryConfig `json:"options,omitempty"`
}

// ---------------------------------------------------------------------------
// AgentGenerateOptions
// ---------------------------------------------------------------------------

// AgentGenerateOptions holds options for generating responses with an agent.
type AgentGenerateOptions struct {
	// Instructions to override the agent's default instructions.
	Instructions SystemMessage `json:"instructions,omitempty"`
	// Toolsets are additional tool sets for this generation.
	Toolsets ToolsetsInput `json:"toolsets,omitempty"`
	// ClientTools are client-side tools available during execution.
	ClientTools ToolsInput `json:"clientTools,omitempty"`
	// Context holds additional context messages.
	Context []CoreMessage `json:"context,omitempty"`
	// Memory configures conversation persistence (preferred over ThreadID/ResourceID).
	Memory *AgentMemoryOption `json:"memory,omitempty"`
	// RunID is a unique ID for this generation run.
	RunID string `json:"runId,omitempty"`
	// OnStepFinish is called after each generation step completes.
	OnStepFinish GenerateTextOnStepFinishCallback `json:"-"`
	// MaxSteps is the maximum number of steps allowed for generation.
	MaxSteps *int `json:"maxSteps,omitempty"`
	// Output is the schema for structured output (does not work with tools).
	Output any `json:"output,omitempty"`
	// ExperimentalOutput is the schema for structured output generation alongside tool calls.
	ExperimentalOutput any `json:"experimental_output,omitempty"`
	// ToolChoice controls how tools are selected during generation.
	// "auto" | "none" | "required" | { type: "tool", toolName: string }
	ToolChoice any `json:"toolChoice,omitempty"`
	// RequestContext for dependency injection.
	RequestContext *RequestContext `json:"requestContext,omitempty"`
	// Scorers to use for this generation.
	Scorers any `json:"scorers,omitempty"`
	// ReturnScorerData indicates whether to return scorer input data. Default: false.
	ReturnScorerData bool `json:"returnScorerData,omitempty"`
	// SavePerStep indicates whether to save messages incrementally on step finish. Default: false.
	SavePerStep bool `json:"savePerStep,omitempty"`
	// InputProcessors to use for this generation call (overrides agent's default).
	InputProcessors []InputProcessorOrWorkflow `json:"inputProcessors,omitempty"`
	// OutputProcessors to use for this generation call (overrides agent's default).
	OutputProcessors []OutputProcessorOrWorkflow `json:"outputProcessors,omitempty"`
	// MaxProcessorRetries overrides agent's default maxProcessorRetries.
	MaxProcessorRetries *int `json:"maxProcessorRetries,omitempty"`
	// TracingOptions for starting new traces.
	TracingOptions *TracingOptions `json:"tracingOptions,omitempty"`
	// ProviderOptions holds provider-specific options.
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`

	// Deprecated: Use Memory instead.
	ResourceID *string `json:"resourceId,omitempty"`
	// Deprecated: Use Memory instead.
	ThreadID *string `json:"threadId,omitempty"`

	// Embedded observability context (partial).
	ObservabilityContext
}

// ---------------------------------------------------------------------------
// AgentStreamOptions
// ---------------------------------------------------------------------------

// AgentStreamOptions holds options for streaming responses with an agent.
type AgentStreamOptions struct {
	// Instructions to override the agent's default instructions.
	Instructions SystemMessage `json:"instructions,omitempty"`
	// Toolsets are additional tool sets for this generation.
	Toolsets ToolsetsInput `json:"toolsets,omitempty"`
	// ClientTools are client-side tools available during execution.
	ClientTools ToolsInput `json:"clientTools,omitempty"`
	// Context holds additional context messages.
	Context []CoreMessage `json:"context,omitempty"`
	// MemoryOptions is deprecated; use Memory instead.
	// Deprecated: Use Memory instead.
	MemoryOptions *MemoryConfig `json:"memoryOptions,omitempty"`
	// Memory configures conversation persistence (preferred).
	Memory *AgentMemoryOption `json:"memory,omitempty"`
	// RunID is a unique ID for this generation run.
	RunID string `json:"runId,omitempty"`
	// OnFinish is called when streaming completes.
	OnFinish any `json:"-"`
	// OnStepFinish is called after each generation step completes.
	OnStepFinish any `json:"-"`
	// MaxSteps is the maximum number of steps allowed for generation.
	MaxSteps *int `json:"maxSteps,omitempty"`
	// Output is the schema for structured output.
	Output any `json:"output,omitempty"`
	// Temperature parameter for controlling randomness.
	Temperature *float64 `json:"temperature,omitempty"`
	// ToolChoice controls how tools are selected during generation.
	ToolChoice any `json:"toolChoice,omitempty"`
	// ExperimentalOutput is the experimental schema for structured output.
	ExperimentalOutput any `json:"experimental_output,omitempty"`
	// RequestContext for dependency injection.
	RequestContext *RequestContext `json:"requestContext,omitempty"`
	// SavePerStep indicates whether to save messages incrementally on step finish. Default: false.
	SavePerStep bool `json:"savePerStep,omitempty"`
	// InputProcessors to use for this generation call (overrides agent's default).
	InputProcessors []InputProcessorOrWorkflow `json:"inputProcessors,omitempty"`
	// TracingOptions for starting new traces.
	TracingOptions *TracingOptions `json:"tracingOptions,omitempty"`
	// Scorers to use for this generation.
	Scorers any `json:"scorers,omitempty"`
	// ProviderOptions holds provider-specific options.
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`

	// Deprecated: Use Memory instead.
	ResourceID *string `json:"resourceId,omitempty"`
	// Deprecated: Use Memory instead.
	ThreadID *string `json:"threadId,omitempty"`

	// Embedded observability context (partial).
	ObservabilityContext
}

// ---------------------------------------------------------------------------
// AgentModelManagerConfig
// ---------------------------------------------------------------------------

// AgentModelManagerConfig extends ModelManagerModelConfig with an Enabled flag.
type AgentModelManagerConfig struct {
	ModelManagerModelConfig
	Enabled bool `json:"enabled"`
}

// ---------------------------------------------------------------------------
// AgentExecuteOnFinishOptions
// ---------------------------------------------------------------------------

// AgentExecuteOnFinishOptions holds options for the onFinish handler of agent execution.
type AgentExecuteOnFinishOptions struct {
	RunID            string              `json:"runId"`
	Result           any                 `json:"result"` // Parameters<StreamTextOnFinishCallback>[0] & { object?: unknown }
	Thread           *StorageThreadType  `json:"thread,omitempty"`
	ReadOnlyMemory   bool                `json:"readOnlyMemory,omitempty"`
	ThreadID         string              `json:"threadId,omitempty"`
	ResourceID       string              `json:"resourceId,omitempty"`
	RequestContext   *RequestContext     `json:"requestContext,omitempty"`
	AgentSpan        Span                `json:"-"`
	MemoryConfig     *MemoryConfig       `json:"memoryConfig,omitempty"`
	OutputText       string              `json:"outputText"`
	MessageList      *MessageList        `json:"-"`
	ThreadExists     bool                `json:"threadExists"`
	StructuredOutput bool                `json:"structuredOutput,omitempty"`
	OverrideScorers  any                 `json:"overrideScorers,omitempty"`
}

// ---------------------------------------------------------------------------
// AgentMethodType
// ---------------------------------------------------------------------------

// AgentMethodType enumerates the method types for agent execution.
type AgentMethodType string

const (
	AgentMethodTypeGenerate       AgentMethodType = "generate"
	AgentMethodTypeStream         AgentMethodType = "stream"
	AgentMethodTypeGenerateLegacy AgentMethodType = "generateLegacy"
	AgentMethodTypeStreamLegacy   AgentMethodType = "streamLegacy"
)

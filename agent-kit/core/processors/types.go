// Ported from: packages/core/src/processors/index.ts
package processors

import (
	"context"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// MastraDBMessage is a stub for ../agent/message-list.MastraDBMessage.
// STUB REASON: Importing agent would create a circular dependency since agent
// imports processors. This struct replicates the agent package's MastraDBMessage shape.
type MastraDBMessage struct {
	ID         string                `json:"id"`
	Role       string                `json:"role"`
	Content    MastraMessageContentV2 `json:"content"`
	CreatedAt  time.Time             `json:"createdAt"`
	ThreadID   string                `json:"threadId,omitempty"`
	ResourceID string                `json:"resourceId,omitempty"`
	Type       string                `json:"type,omitempty"`
}

// MastraMessageContentV2 is a stub for ../agent/message-list.MastraMessageContentV2.
// STUB REASON: Same as MastraDBMessage — agent imports processors (circular).
type MastraMessageContentV2 struct {
	Format           int                    `json:"format"`
	Parts            []MessagePart          `json:"parts"`
	Content          string                 `json:"content,omitempty"`
	Metadata         map[string]any         `json:"metadata,omitempty"`
	ProviderMetadata map[string]any         `json:"providerMetadata,omitempty"`
}

// MessageList is a stub for ../agent/message-list.MessageList.
// STUB REASON: Same as MastraDBMessage — agent imports processors (circular).
// Now stores actual state so memory processors can use it for dedup, adding
// historical messages, and persisting new input/response messages.
//
// Source tracking: each message is tagged with one of "input", "response", or
// "memory" so that GetInputDB / GetResponseDB / GetAllDB can return the right
// subsets.  IsNewMessage returns true for "input" and "response" sources (i.e.
// messages that were not loaded from storage).
type MessageList struct {
	// messages stores all non-system messages added via Add.
	messages []MastraDBMessage

	// sources tracks the source tag for each message by index (parallel to messages).
	sources []string

	// systemMessages stores system messages added via AddSystem.
	systemMessages []SystemMessageEntry

	// memoryInfo stores optional thread/resource info for fallback context.
	memoryInfo *MessageListMemoryInfo
}

// SystemMessageEntry stores a system message with its optional tag.
type SystemMessageEntry struct {
	Text string `json:"text"`
	Tag  string `json:"tag,omitempty"`
}

// MessageListMemoryInfo stores thread/resource info attached to a MessageList.
// Mirrors the memoryInfo field in the real MessageList.
type MessageListMemoryInfo struct {
	ThreadID   string `json:"threadId,omitempty"`
	ResourceID string `json:"resourceId,omitempty"`
}

// MessageListSerializedState is the serialized state returned by Serialize.
// Matches the shape the memory processors expect when falling back to
// MessageList's memoryInfo.
type MessageListSerializedState struct {
	MemoryInfo *MessageListMemoryInfo `json:"memoryInfo,omitempty"`
}

// SetMemoryInfo sets the thread/resource info on the MessageList.
// Called when the MessageList is created with thread context.
func (ml *MessageList) SetMemoryInfo(threadID, resourceID string) {
	ml.memoryInfo = &MessageListMemoryInfo{
		ThreadID:   threadID,
		ResourceID: resourceID,
	}
}

// Serialize returns the serialized state of the MessageList.
// Used by memory processors to access memoryInfo as a fallback.
func (ml *MessageList) Serialize() MessageListSerializedState {
	return MessageListSerializedState{
		MemoryInfo: ml.memoryInfo,
	}
}

// MessageListMutation records a single mutation applied to a MessageList.
// Used for observability tracking of processor-applied changes.
type MessageListMutation struct {
	Type    string `json:"type"`              // "add", "addSystem", "removeByIds", "clear", "replaceAllSystemMessages"
	Source  string `json:"source,omitempty"`   // Message source (e.g., "input", "response")
	Count   int    `json:"count,omitempty"`    // Number of items affected
	IDs     []string `json:"ids,omitempty"`    // IDs involved in the mutation
	Text    string `json:"text,omitempty"`     // Text content (for addSystem)
	Tag     string `json:"tag,omitempty"`      // Tag (for addSystem)
	Message any    `json:"message,omitempty"`  // Message data
}

// MessageSourceChecker provides source tracking for processor messages.
// Returned by MessageList.MakeMessageSourceChecker.
type MessageSourceChecker struct{}

// GetSource returns the source of a message, or empty string if unknown.
func (c *MessageSourceChecker) GetSource(msg MastraDBMessage) string {
	return ""
}

// GetResponseDB returns all response messages in MastraDBMessage format.
// Ported from: messageList.get.response.db() in TS.
func (ml *MessageList) GetResponseDB() []MastraDBMessage {
	var result []MastraDBMessage
	for i, msg := range ml.messages {
		if i < len(ml.sources) && ml.sources[i] == "response" {
			result = append(result, msg)
		}
	}
	return result
}

// GetInputDB returns all input messages in MastraDBMessage format.
// Ported from: messageList.get.input.db() in TS.
func (ml *MessageList) GetInputDB() []MastraDBMessage {
	var result []MastraDBMessage
	for i, msg := range ml.messages {
		if i < len(ml.sources) && ml.sources[i] == "input" {
			result = append(result, msg)
		}
	}
	return result
}

// GetAllDB returns all messages in MastraDBMessage format.
// Ported from: messageList.get.all.db() in TS.
func (ml *MessageList) GetAllDB() []MastraDBMessage {
	result := make([]MastraDBMessage, len(ml.messages))
	copy(result, ml.messages)
	return result
}

// StartRecording begins recording mutations applied to this MessageList.
// Used by processors for observability tracking.
// Stub: no-op until MessageList is fully ported.
func (ml *MessageList) StartRecording() {}

// StopRecording stops recording mutations and returns all recorded mutations.
// Stub: returns nil until MessageList is fully ported.
func (ml *MessageList) StopRecording() []MessageListMutation { return nil }

// MakeMessageSourceChecker creates a source checker for tracking message origins.
// Stub: returns empty checker until MessageList is fully ported.
func (ml *MessageList) MakeMessageSourceChecker() *MessageSourceChecker {
	return &MessageSourceChecker{}
}

// RemoveByIds removes messages with the given IDs.
// Ported from: messageList.removeByIds() in TS.
func (ml *MessageList) RemoveByIds(ids []string) {
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	var remaining []MastraDBMessage
	var remainingSources []string
	for i, msg := range ml.messages {
		if _, found := idSet[msg.ID]; !found {
			remaining = append(remaining, msg)
			if i < len(ml.sources) {
				remainingSources = append(remainingSources, ml.sources[i])
			}
		}
	}
	ml.messages = remaining
	ml.sources = remainingSources
}

// Add adds a message with the given source.
// source is one of "input", "response", "memory".
// Ported from: messageList.add(msg, source) in TS.
// When source is "memory", duplicate messages (by ID) are skipped.
func (ml *MessageList) Add(msg MastraDBMessage, source string) {
	// Normalize "user" source to "input" (matches TS behavior).
	if source == "user" {
		source = "input"
	}
	// For memory source, skip duplicates by ID.
	if source == "memory" && msg.ID != "" {
		for _, existing := range ml.messages {
			if existing.ID == msg.ID {
				return
			}
		}
	}
	ml.messages = append(ml.messages, msg)
	ml.sources = append(ml.sources, source)
}

// AddSystem adds a system message with an optional tag.
// Ported from: messageList.addSystem(text, tag?) in TS.
func (ml *MessageList) AddSystem(text string, tag ...string) {
	t := ""
	if len(tag) > 0 {
		t = tag[0]
	}
	ml.systemMessages = append(ml.systemMessages, SystemMessageEntry{
		Text: text,
		Tag:  t,
	})
}

// IsNewMessage returns true if the message (or message ID) is a new user or
// response message (i.e. not from memory/storage).
// Ported from: messageList.isNewMessage() in TS.
func (ml *MessageList) IsNewMessage(msg MastraDBMessage) bool {
	for i, m := range ml.messages {
		if m.ID == msg.ID && i < len(ml.sources) {
			src := ml.sources[i]
			return src == "input" || src == "response"
		}
	}
	// If not found in the list, treat as new.
	return true
}

// GetAllSystemMessages returns all system messages.
// Stub: returns nil until MessageList is fully ported.
func (ml *MessageList) GetAllSystemMessages() []CoreMessageV4 { return nil }

// ReplaceAllSystemMessages replaces all system messages with the given messages.
// Stub: no-op until MessageList is fully ported.
func (ml *MessageList) ReplaceAllSystemMessages(msgs []CoreMessageV4) {}

// CoreMessageV4 is a stub for @internal/ai-sdk-v4.CoreMessage.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type CoreMessageV4 struct {
	Role    string `json:"role"`
	Content any    `json:"content,omitempty"`
}

// StepResult is a stub for @internal/ai-sdk-v5.StepResult.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type StepResult struct {
	// Placeholder fields for step result data.
}

// ToolChoice is a stub for @internal/ai-sdk-v5.ToolChoice.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type ToolChoice = any

// CallSettings is a stub for @internal/ai-sdk-v5.CallSettings.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type CallSettings struct {
	AbortSignal context.Context `json:"-"`
}

// TripWireOptions holds options for the abort/tripwire mechanism.
// STUB REASON: Part of the agent package's tripwire system. Agent imports processors (circular).
type TripWireOptions struct {
	Retry    bool `json:"retry,omitempty"`
	Metadata any  `json:"metadata,omitempty"`
}

// ChunkType is a stub for ../stream.ChunkType.
// STUB REASON: The stream package contains stub types itself. Importing it would add
// a dependency with no real type benefit (stream.ChunkType is also a stub struct).
type ChunkType struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// DataChunkType is a stub for ../stream/types.DataChunkType.
// STUB REASON: Same as ChunkType — stream package has only stub types.
type DataChunkType struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Data any    `json:"data,omitempty"`
}

// InferSchemaOutput is a stub for ../stream.InferSchemaOutput.
// STUB REASON: Same as ChunkType — stream package has only stub types.
type InferSchemaOutput = any

// OutputSchema is a stub for ../stream.OutputSchema.
// STUB REASON: Same as ChunkType — stream package has only stub types.
type OutputSchema = any

// MastraLanguageModel is imported from the llm/model package.
type MastraLanguageModel = model.MastraLanguageModel

// OpenAICompatibleConfig is imported from the llm/model package.
type OpenAICompatibleConfig = model.OpenAICompatibleConfig

// SharedProviderOptions is imported from the llm/model package.
type SharedProviderOptions = model.SharedProviderOptions

// ModelRouterModelId is imported from the llm/model package.
type ModelRouterModelId = model.ModelRouterModelID

// LanguageModelV2 is a stub for @ai-sdk/provider-v5.LanguageModelV2.
// ai-kit only ported V3. See brainlink/experiments/ai-kit/provider/languagemodel.
type LanguageModelV2 interface {
	MastraLanguageModel
}

// Mastra represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core may import processors (via processor runner), so processors cannot import core.
// core.Mastra struct satisfies this interface.
// Processors use this to access logging and storage when transforming inputs/outputs.
type Mastra interface {
	// GetLogger returns the configured logger instance.
	GetLogger() logger.IMastraLogger
}

// MastraLogger is a type alias to logger.IMastraLogger so that core.Mastra
// satisfies the processors.Mastra interface at compile time.
//
// Ported from: packages/core/src/processors — uses mastra.getLogger()
type MastraLogger = logger.IMastraLogger

// Workflow is a stub for ../workflows.Workflow.
// STUB REASON: Importing workflows would create a dependency that doesn't exist
// in the current architecture. The real workflows.Workflow struct uses GetID().
// This stub matches the real type's method signature.
type Workflow interface {
	GetID() string
}

// StructuredOutputOptions holds options for structured output.
// Defined here as canonical type. Used by processors/processors subpackage.
type StructuredOutputOptions struct {
	Schema              any  `json:"schema,omitempty"`
	Model               any  `json:"model,omitempty"`
	JSONPromptInjection bool `json:"jsonPromptInjection,omitempty"`
}

// ---------------------------------------------------------------------------
// ProcessorStreamWriter
// ---------------------------------------------------------------------------

// ProcessorStreamWriter is the writer interface for processors to emit custom
// data chunks to the stream. This enables real-time streaming of
// processor-specific data (e.g., observation markers).
type ProcessorStreamWriter interface {
	// Custom emits a custom data chunk to the stream.
	// The chunk type must start with 'data-' prefix.
	Custom(data DataChunkType) error
}

// ---------------------------------------------------------------------------
// ProcessorContext
// ---------------------------------------------------------------------------

// ProcessorContext is the base context shared by all processor methods.
type ProcessorContext struct {
	// Abort aborts processing with an optional reason and options.
	// In Go, this returns an error rather than using "never" return type.
	Abort func(reason string, options *TripWireOptions) error

	// RequestContext holds optional runtime context with execution metadata.
	RequestContext *requestcontext.RequestContext

	// RetryCount is the number of times processors have triggered retry for this generation.
	// Use this to implement retry limits within your processor.
	RetryCount int

	// Writer is an optional stream writer for emitting custom data chunks.
	// Available when the agent is streaming and outputWriter is provided.
	Writer ProcessorStreamWriter

	// AbortSignal is an optional context for cancellation from the parent agent execution.
	// Processors should pass this to any long-running operations (e.g., LLM calls)
	// so they can be canceled when the parent agent is aborted.
	AbortSignal context.Context

	// ObservabilityContext provides tracing context for processors.
	// Passed through to processor methods for span creation and correlation.
	ObservabilityContext *obstypes.ObservabilityContext
}

// ---------------------------------------------------------------------------
// ProcessorMessageContext
// ---------------------------------------------------------------------------

// ProcessorMessageContext is the context for message-based processor methods
// (processInput, processOutputResult, processInputStep).
type ProcessorMessageContext struct {
	ProcessorContext

	// Messages is the current messages being processed.
	Messages []MastraDBMessage

	// MessageList is the MessageList instance for managing message sources.
	MessageList *MessageList
}

// ---------------------------------------------------------------------------
// ProcessInputResultWithSystemMessages
// ---------------------------------------------------------------------------

// ProcessInputResultWithSystemMessages is the return type for processInput
// that includes modified system messages.
type ProcessInputResultWithSystemMessages struct {
	Messages       []MastraDBMessage `json:"messages"`
	SystemMessages []CoreMessageV4   `json:"systemMessages"`
}

// ---------------------------------------------------------------------------
// ProcessInputArgs
// ---------------------------------------------------------------------------

// ProcessInputArgs holds arguments for the ProcessInput method.
type ProcessInputArgs struct {
	ProcessorMessageContext

	// SystemMessages contains all system messages (agent instructions, user-provided, memory)
	// for read/modify access.
	SystemMessages []CoreMessageV4

	// State is per-processor state that persists across all method calls within this request.
	State map[string]any
}

// ---------------------------------------------------------------------------
// ProcessOutputResultArgs
// ---------------------------------------------------------------------------

// ProcessOutputResultArgs holds arguments for the ProcessOutputResult method.
type ProcessOutputResultArgs struct {
	ProcessorMessageContext

	// State is per-processor state that persists across all method calls within this request.
	State map[string]any
}

// ---------------------------------------------------------------------------
// ProcessInputStepArgs
// ---------------------------------------------------------------------------

// ProcessInputStepArgs holds arguments for the ProcessInputStep method.
//
// Note: StructuredOutput.Schema is typed as OutputSchema (not the specific OUTPUT type) because
// processors run in a chain and any previous processor may have modified structuredOutput.
// The actual schema type is only known at the generate()/stream() call site.
type ProcessInputStepArgs struct {
	ProcessorMessageContext

	// StepNumber is the current step number (0-indexed).
	StepNumber int

	// Steps contains results from previous steps.
	Steps []StepResult

	// SystemMessages contains all system messages for read/modify access.
	SystemMessages []CoreMessageV4

	// State is per-processor state that persists across all method calls within this request.
	State map[string]any

	// Model is the current model for this step.
	// Can be a resolved LanguageModelV2 or an unresolved config (string, OpenAI-compatible config).
	Model MastraLanguageModel

	// Tools contains the current tools available for this step.
	Tools map[string]any

	// ToolChoice specifies how tools should be selected.
	ToolChoice ToolChoice

	// ActiveTools lists the currently active tools.
	ActiveTools []string

	// ProviderOptions contains provider-specific options.
	ProviderOptions SharedProviderOptions

	// ModelSettings contains model settings (temperature, etc.), excluding AbortSignal.
	ModelSettings map[string]any

	// StructuredOutput contains structured output configuration.
	StructuredOutput *StructuredOutputOptions
}

// ---------------------------------------------------------------------------
// RunProcessInputStepArgs
// ---------------------------------------------------------------------------

// RunProcessInputStepArgs is ProcessInputStepArgs without Messages, SystemMessages,
// Abort, and State (which are injected by the runner).
type RunProcessInputStepArgs struct {
	MessageList      *MessageList
	StepNumber       int
	Steps            []StepResult
	Model            MastraLanguageModel
	Tools            map[string]any
	ToolChoice       ToolChoice
	ActiveTools      []string
	ProviderOptions  SharedProviderOptions
	ModelSettings    map[string]any
	StructuredOutput *StructuredOutputOptions
	RetryCount       int
	RequestContext   *requestcontext.RequestContext
	Writer           ProcessorStreamWriter
	AbortSignal      context.Context

	// ObservabilityContext provides tracing context for processor span creation.
	ObservabilityContext *obstypes.ObservabilityContext
}

// ---------------------------------------------------------------------------
// ProcessInputStepResult
// ---------------------------------------------------------------------------

// ProcessInputStepResult is the result from ProcessInputStep method.
//
// Note: StructuredOutput.Schema is typed as OutputSchema (not the specific OUTPUT type) because
// processors can modify it dynamically, and the actual type is only known at runtime.
type ProcessInputStepResult struct {
	Model            any                  `json:"model,omitempty"`
	Tools            map[string]any       `json:"tools,omitempty"`
	ToolChoice       ToolChoice           `json:"toolChoice,omitempty"`
	ActiveTools      []string             `json:"activeTools,omitempty"`
	Messages         []MastraDBMessage    `json:"messages,omitempty"`
	MessageList      *MessageList         `json:"messageList,omitempty"`
	SystemMessages   []CoreMessageV4      `json:"systemMessages,omitempty"`
	ProviderOptions  SharedProviderOptions `json:"providerOptions,omitempty"`
	ModelSettings    map[string]any       `json:"modelSettings,omitempty"`
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
	RetryCount       *int                 `json:"retryCount,omitempty"`
}

// RunProcessInputStepResult is ProcessInputStepResult with Model resolved to MastraLanguageModel.
type RunProcessInputStepResult struct {
	Model            MastraLanguageModel   `json:"model,omitempty"`
	Tools            map[string]any        `json:"tools,omitempty"`
	ToolChoice       ToolChoice            `json:"toolChoice,omitempty"`
	ActiveTools      []string              `json:"activeTools,omitempty"`
	Messages         []MastraDBMessage     `json:"messages,omitempty"`
	MessageList      *MessageList          `json:"messageList,omitempty"`
	SystemMessages   []CoreMessageV4       `json:"systemMessages,omitempty"`
	ProviderOptions  SharedProviderOptions  `json:"providerOptions,omitempty"`
	ModelSettings    map[string]any        `json:"modelSettings,omitempty"`
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
	RetryCount       int                   `json:"retryCount,omitempty"`
}

// ---------------------------------------------------------------------------
// ProcessOutputStreamArgs
// ---------------------------------------------------------------------------

// ProcessOutputStreamArgs holds arguments for the ProcessOutputStream method.
type ProcessOutputStreamArgs struct {
	ProcessorContext

	// Part is the current chunk being processed.
	Part ChunkType

	// StreamParts contains all chunks seen so far.
	StreamParts []ChunkType

	// State is a mutable state object that persists across chunks.
	State map[string]any

	// MessageList is an optional MessageList instance for accessing conversation history.
	MessageList *MessageList
}

// ---------------------------------------------------------------------------
// ToolCallInfo
// ---------------------------------------------------------------------------

// ToolCallInfo contains tool call information for ProcessOutputStep.
type ToolCallInfo struct {
	ToolName   string `json:"toolName"`
	ToolCallID string `json:"toolCallId"`
	Args       any    `json:"args,omitempty"`
}

// ---------------------------------------------------------------------------
// ProcessOutputStepArgs
// ---------------------------------------------------------------------------

// ProcessOutputStepArgs holds arguments for the ProcessOutputStep method.
// Called after each LLM response in the agentic loop, before tool execution.
type ProcessOutputStepArgs struct {
	ProcessorMessageContext

	// StepNumber is the current step number (0-indexed).
	StepNumber int

	// FinishReason is the finish reason from the LLM (stop, tool-use, length, etc.).
	FinishReason string

	// ToolCalls contains tool calls made in this step (if any).
	ToolCalls []ToolCallInfo

	// Text is the generated text from this step.
	Text string

	// SystemMessages contains all system messages.
	SystemMessages []CoreMessageV4

	// Steps contains all completed steps so far (including the current step).
	Steps []StepResult

	// State is a mutable state object that persists across steps.
	State map[string]any
}

// ---------------------------------------------------------------------------
// Processor interface
// ---------------------------------------------------------------------------

// Processor is the interface for transforming messages and stream chunks.
type Processor interface {
	// ID returns the processor's unique identifier.
	ID() string

	// Name returns the processor's optional human-readable name.
	Name() string

	// Description returns the processor's optional description.
	Description() string

	// ProcessorIndex returns the index of this processor in the workflow (set at runtime).
	ProcessorIndex() int

	// SetProcessorIndex sets the index of this processor in the workflow.
	SetProcessorIndex(index int)
}

// InputProcessor extends Processor with input processing capability.
// Must implement either ProcessInput or ProcessInputStep (or both).
type InputProcessor interface {
	Processor
	InputProcessorMethods
}

// InputProcessorMethods defines the optional input processing methods.
type InputProcessorMethods interface {
	// ProcessInput processes input messages before they are sent to the LLM.
	//
	// Returns one of:
	//   - ([]MastraDBMessage, nil, nil): Transformed messages array
	//   - (nil, nil, &ProcessInputResultWithSystemMessages{...}): Messages + modified system messages
	//   - (nil, messageList, nil): Same messageList instance (indicates mutation)
	ProcessInput(args ProcessInputArgs) ([]MastraDBMessage, *MessageList, *ProcessInputResultWithSystemMessages, error)

	// ProcessInputStep processes input messages at each step of the agentic loop.
	// Unlike ProcessInput which runs once at the start, this runs at every step.
	//
	// Returns one of:
	//   - (*ProcessInputStepResult, nil): Step result with model, toolChoice, messages, etc.
	//   - (nil, []MastraDBMessage): Transformed messages array
	//   - (nil, nil): No changes
	ProcessInputStep(args ProcessInputStepArgs) (*ProcessInputStepResult, []MastraDBMessage, error)
}

// OutputProcessor extends Processor with output processing capability.
// Must implement either ProcessOutputStream, ProcessOutputResult, or ProcessOutputStep.
type OutputProcessor interface {
	Processor
	OutputProcessorMethods
}

// OutputProcessorMethods defines the optional output processing methods.
type OutputProcessorMethods interface {
	// ProcessOutputStream processes output stream chunks with built-in state management.
	// Return nil to skip emitting the part.
	ProcessOutputStream(args ProcessOutputStreamArgs) (*ChunkType, error)

	// ProcessOutputResult processes the complete output result after streaming/generate is finished.
	//
	// Returns one of:
	//   - ([]MastraDBMessage, nil): Transformed messages array
	//   - (nil, messageList): Same messageList instance (indicates mutation)
	ProcessOutputResult(args ProcessOutputResultArgs) ([]MastraDBMessage, *MessageList, error)

	// ProcessOutputStep processes output after each LLM response in the agentic loop,
	// before tool execution.
	//
	// Returns one of:
	//   - ([]MastraDBMessage, nil): Transformed messages array
	//   - (nil, messageList): Same messageList instance (indicates mutation)
	ProcessOutputStep(args ProcessOutputStepArgs) ([]MastraDBMessage, *MessageList, error)
}

// MastraRegistrable is an interface for processors that need access to Mastra services.
type MastraRegistrable interface {
	// RegisterMastra is called when the processor is registered with a Mastra instance.
	RegisterMastra(mastra Mastra)
}

// ---------------------------------------------------------------------------
// BaseProcessor
// ---------------------------------------------------------------------------

// BaseProcessor provides a base implementation for processors that need access
// to Mastra services. Embed this struct to get access to the Mastra instance
// when the processor is registered with an agent.
type BaseProcessor struct {
	id             string
	name           string
	description    string
	processorIndex int
	mastra         Mastra
}

// NewBaseProcessor creates a new BaseProcessor with the given ID and name.
func NewBaseProcessor(id, name string) BaseProcessor {
	return BaseProcessor{
		id:   id,
		name: name,
	}
}

// ID returns the processor's unique identifier.
func (bp *BaseProcessor) ID() string { return bp.id }

// Name returns the processor's optional human-readable name.
func (bp *BaseProcessor) Name() string { return bp.name }

// Description returns the processor's optional description.
func (bp *BaseProcessor) Description() string { return bp.description }

// ProcessorIndex returns the index of this processor in the workflow.
func (bp *BaseProcessor) ProcessorIndex() int { return bp.processorIndex }

// SetProcessorIndex sets the index of this processor in the workflow.
func (bp *BaseProcessor) SetProcessorIndex(index int) { bp.processorIndex = index }

// RegisterMastra is called when the processor is registered with a Mastra instance.
func (bp *BaseProcessor) RegisterMastra(mastra Mastra) { bp.mastra = mastra }

// GetMastra returns the Mastra instance this processor is registered with.
func (bp *BaseProcessor) GetMastra() Mastra { return bp.mastra }

// ---------------------------------------------------------------------------
// ProcessorWorkflow
// ---------------------------------------------------------------------------

// ProcessorWorkflow is a Workflow that can be used as a processor.
// The workflow must accept ProcessorStepInput and return ProcessorStepOutput.
type ProcessorWorkflow interface {
	Workflow
	Execute(input ProcessorStepOutput) (*ProcessorStepOutput, error)
	CreateRun() (WorkflowRun, error)
}

// WorkflowRun is a stub for workflow run instances.
// STUB REASON: Importing workflows would add a dependency that doesn't exist
// in the current architecture. This interface matches the subset of methods
// used by the processor runner.
type WorkflowRun interface {
	Start(opts WorkflowRunStartOpts) (*WorkflowRunResult, error)
}

// WorkflowRunStartOpts holds options for starting a workflow run.
// STUB REASON: Same as WorkflowRun — workflow dependency not yet wired.
type WorkflowRunStartOpts struct {
	InputData            ProcessorStepOutput
	RequestContext       *requestcontext.RequestContext
	OutputWriter         func(data any) error
	ObservabilityContext *obstypes.ObservabilityContext
}

// WorkflowRunResult holds the result of a workflow run.
// STUB REASON: Same as WorkflowRun — workflow dependency not yet wired.
type WorkflowRunResult struct {
	Status  string                  `json:"status"`
	Result  *ProcessorStepOutput    `json:"result,omitempty"`
	Error   *WorkflowRunError       `json:"error,omitempty"`
	Steps   map[string]WorkflowStep `json:"steps,omitempty"`
	Tripwire *WorkflowTripwireData  `json:"tripwire,omitempty"`
}

// WorkflowRunError holds error data from a workflow run.
type WorkflowRunError struct {
	Message string `json:"message"`
}

// WorkflowStep holds a step result from a workflow run.
type WorkflowStep struct {
	Status string           `json:"status"`
	Error  *WorkflowRunError `json:"error,omitempty"`
}

// WorkflowTripwireData holds tripwire data from a workflow run.
type WorkflowTripwireData struct {
	Reason      string `json:"reason,omitempty"`
	Retry       bool   `json:"retry,omitempty"`
	Metadata    any    `json:"metadata,omitempty"`
	ProcessorID string `json:"processorId,omitempty"`
}

// ---------------------------------------------------------------------------
// IsProcessorWorkflow
// ---------------------------------------------------------------------------

// IsProcessorWorkflow checks if an object implements the ProcessorWorkflow interface.
// A ProcessorWorkflow must have ID, Execute, and CreateRun methods, and must NOT
// implement InputProcessor or OutputProcessor interfaces (to distinguish from Processor).
func IsProcessorWorkflow(obj any) bool {
	if obj == nil {
		return false
	}
	_, isProcessorWorkflow := obj.(ProcessorWorkflow)
	if !isProcessorWorkflow {
		return false
	}
	// Must NOT have processor-specific interfaces
	_, isInput := obj.(InputProcessor)
	_, isOutput := obj.(OutputProcessor)
	return !isInput && !isOutput
}

// ---------------------------------------------------------------------------
// ProcessorOrWorkflow
// ---------------------------------------------------------------------------

// ProcessorOrWorkflow is a union type for processor or workflow that can be
// used as a processor. In Go, we use an interface with a marker method.
type ProcessorOrWorkflow interface {
	// processorOrWorkflowMarker is a marker method to identify types that can
	// act as processors in the pipeline.
	processorOrWorkflowMarker()
}

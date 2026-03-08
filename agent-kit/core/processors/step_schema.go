// Ported from: packages/core/src/processors/step-schema.ts
package processors

// This file defines the step schema types used for processor workflows.
// In TypeScript, these are Zod schemas; in Go, they are plain struct types
// since Go does not have runtime schema validation built in.

// ---------------------------------------------------------------------------
// Message Part Types
// ---------------------------------------------------------------------------

// TextPart represents a text part in a message.
type TextPart struct {
	Type string `json:"type"` // always "text"
	Text string `json:"text"`
}

// ImagePart represents an image part in a message.
type ImagePart struct {
	Type     string `json:"type"` // always "image"
	Image    any    `json:"image"` // string | URL | []byte
	MimeType string `json:"mimeType,omitempty"`
}

// FilePart represents a file part in a message.
type FilePart struct {
	Type     string `json:"type"` // always "file"
	Data     any    `json:"data"` // string | URL | []byte
	MimeType string `json:"mimeType"`
}

// ToolInvocation holds the details of a tool invocation.
type ToolInvocation struct {
	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	Args       any    `json:"args,omitempty"`
	State      string `json:"state"` // "partial-call" | "call" | "result"
	Result     any    `json:"result,omitempty"`
}

// ToolInvocationPart represents a tool invocation part in a message.
type ToolInvocationPart struct {
	Type           string         `json:"type"` // always "tool-invocation"
	ToolInvocation ToolInvocation `json:"toolInvocation"`
}

// ReasoningDetail holds a single detail in a reasoning part.
type ReasoningDetail struct {
	Type string `json:"type"` // "text" | "redacted"
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
}

// ReasoningPart represents a reasoning part in a message.
type ReasoningPart struct {
	Type      string            `json:"type"` // always "reasoning"
	Reasoning string            `json:"reasoning"`
	Details   []ReasoningDetail `json:"details"`
}

// Source holds source metadata for citations/references.
type Source struct {
	SourceType string `json:"sourceType"`
	ID         string `json:"id"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
}

// SourcePart represents a source part in a message.
type SourcePart struct {
	Type   string `json:"type"` // always "source"
	Source Source `json:"source"`
}

// StepStartPart represents a step start marker in a message.
type StepStartPart struct {
	Type string `json:"type"` // always "step-start"
}

// DataPart represents a custom data part (for data-* custom parts).
type DataPart struct {
	Type string `json:"type"` // must start with "data-"
	ID   string `json:"id,omitempty"`
	Data any    `json:"data,omitempty"`
}

// MessagePart is a union of all message part types.
// In Go, we represent this as an interface; concrete types above implement it.
// For JSON serialization, use the Type field to discriminate.
type MessagePart struct {
	Type string `json:"type"`

	// Text fields (for "text" type)
	Text string `json:"text,omitempty"`

	// Image fields (for "image" type)
	Image any `json:"image,omitempty"`

	// File/Data fields (for "file" type)
	FileData any `json:"data,omitempty"`

	// MimeType (for "image" and "file" types)
	MimeType string `json:"mimeType,omitempty"`

	// ToolInvocation (for "tool-invocation" type)
	ToolInvocationData *ToolInvocation `json:"toolInvocation,omitempty"`

	// Reasoning fields (for "reasoning" type)
	Reasoning string            `json:"reasoning,omitempty"`
	Details   []ReasoningDetail `json:"details,omitempty"`

	// Source fields (for "source" type)
	Source *Source `json:"source,omitempty"`

	// Data part fields (for "data-*" types)
	ID       string `json:"id,omitempty"`
	DataBody any    `json:"dataBody,omitempty"`
}

// ---------------------------------------------------------------------------
// Message Content
// ---------------------------------------------------------------------------

// MessageContent represents the content structure of a processor message.
type MessageContent struct {
	Format           int                `json:"format"` // always 2
	Parts            []MessagePart      `json:"parts"`
	Content          string             `json:"content,omitempty"`
	Metadata         map[string]any     `json:"metadata,omitempty"`
	ProviderMetadata map[string]any     `json:"providerMetadata,omitempty"`
}

// ---------------------------------------------------------------------------
// SystemMessage
// ---------------------------------------------------------------------------

// SystemMessageTextPart is a text part in a system message.
type SystemMessageTextPart struct {
	Type string `json:"type"` // always "text"
	Text string `json:"text"`
}

// SystemMessage represents a system message (CoreSystemMessage from AI SDK).
type SystemMessage struct {
	Role                          string `json:"role"` // always "system"
	Content                       any    `json:"content"` // string | []SystemMessageTextPart
	ExperimentalProviderMetadata  map[string]any `json:"experimental_providerMetadata,omitempty"`
}

// ---------------------------------------------------------------------------
// CoreMessage
// ---------------------------------------------------------------------------

// CoreMessage represents any message type from AI SDK.
type CoreMessage struct {
	Role    string `json:"role"` // "system" | "user" | "assistant" | "tool"
	Content any    `json:"content,omitempty"`
}

// ---------------------------------------------------------------------------
// ProcessorMessage
// ---------------------------------------------------------------------------

// ProcessorMessage represents a message in the processor workflow.
type ProcessorMessage struct {
	ID         string         `json:"id"`
	Role       string         `json:"role"` // "user" | "assistant" | "system" | "tool"
	CreatedAt  string         `json:"createdAt"` // ISO8601 timestamp
	ThreadID   string         `json:"threadId,omitempty"`
	ResourceID string         `json:"resourceId,omitempty"`
	Type       string         `json:"type,omitempty"`
	Content    MessageContent `json:"content"`
}

// ---------------------------------------------------------------------------
// Model and Tools config types
// ---------------------------------------------------------------------------

// ProcessorStepModelConfig is a union type for model configurations.
// In workflows, model configs may not yet be resolved, so we accept both
// resolved and unresolved types.
type ProcessorStepModelConfig = any

// ProcessorStepToolsConfig is a union type for tool configurations.
type ProcessorStepToolsConfig = map[string]any

// ---------------------------------------------------------------------------
// Phase-specific types (discriminated union by Phase field)
// ---------------------------------------------------------------------------

// ProcessorPhase is a string type for the processor phase name.
type ProcessorPhase string

const (
	ProcessorPhaseInput        ProcessorPhase = "input"
	ProcessorPhaseInputStep    ProcessorPhase = "inputStep"
	ProcessorPhaseOutputStream ProcessorPhase = "outputStream"
	ProcessorPhaseOutputResult ProcessorPhase = "outputResult"
	ProcessorPhaseOutputStep   ProcessorPhase = "outputStep"
)

// ProcessorInputPhase is the data for the 'input' phase.
type ProcessorInputPhase struct {
	Phase          ProcessorPhase    `json:"phase"` // "input"
	Messages       []ProcessorMessage `json:"messages"`
	MessageList    *MessageList       `json:"-"`
	SystemMessages []CoreMessage      `json:"systemMessages,omitempty"`
	RetryCount     *int               `json:"retryCount,omitempty"`
}

// ProcessorInputStepPhase is the data for the 'inputStep' phase.
type ProcessorInputStepPhase struct {
	Phase            ProcessorPhase       `json:"phase"` // "inputStep"
	Messages         []ProcessorMessage   `json:"messages"`
	MessageList      *MessageList         `json:"-"`
	StepNumber       int                  `json:"stepNumber"`
	SystemMessages   []CoreMessage        `json:"systemMessages,omitempty"`
	RetryCount       *int                 `json:"retryCount,omitempty"`
	Model            ProcessorStepModelConfig `json:"model,omitempty"`
	Tools            ProcessorStepToolsConfig `json:"tools,omitempty"`
	ToolChoice       ToolChoice           `json:"toolChoice,omitempty"`
	ActiveTools      []string             `json:"activeTools,omitempty"`
	ProviderOptions  SharedProviderOptions `json:"providerOptions,omitempty"`
	ModelSettings    map[string]any       `json:"modelSettings,omitempty"`
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
	Steps            []StepResult         `json:"steps,omitempty"`
}

// ProcessorOutputStreamPhase is the data for the 'outputStream' phase.
type ProcessorOutputStreamPhase struct {
	Phase       ProcessorPhase     `json:"phase"` // "outputStream"
	Part        any                `json:"part"`
	StreamParts []any              `json:"streamParts"`
	State       map[string]any     `json:"state"`
	MessageList *MessageList       `json:"-"`
	RetryCount  *int               `json:"retryCount,omitempty"`
}

// ProcessorOutputResultPhase is the data for the 'outputResult' phase.
type ProcessorOutputResultPhase struct {
	Phase       ProcessorPhase     `json:"phase"` // "outputResult"
	Messages    []ProcessorMessage `json:"messages"`
	MessageList *MessageList       `json:"-"`
	RetryCount  *int               `json:"retryCount,omitempty"`
}

// ProcessorOutputStepPhase is the data for the 'outputStep' phase.
type ProcessorOutputStepPhase struct {
	Phase          ProcessorPhase     `json:"phase"` // "outputStep"
	Messages       []ProcessorMessage `json:"messages"`
	MessageList    *MessageList       `json:"-"`
	StepNumber     int                `json:"stepNumber"`
	FinishReason   string             `json:"finishReason,omitempty"`
	ToolCalls      []ToolCallInfo     `json:"toolCalls,omitempty"`
	Text           string             `json:"text,omitempty"`
	SystemMessages []CoreMessage      `json:"systemMessages,omitempty"`
	RetryCount     *int               `json:"retryCount,omitempty"`
}

// ---------------------------------------------------------------------------
// ProcessorStepInput
// ---------------------------------------------------------------------------

// ProcessorStepInput is a discriminated union of all phase-specific input types.
// Use the Phase field to determine which fields are valid.
// In Go, this is represented as a single struct with all possible fields.
type ProcessorStepInput struct {
	Phase ProcessorPhase `json:"phase"`

	// Common message fields (used by most phases)
	Messages       []ProcessorMessage `json:"messages,omitempty"`
	MessageList    *MessageList       `json:"-"`
	SystemMessages []CoreMessage      `json:"systemMessages,omitempty"`
	RetryCount     *int               `json:"retryCount,omitempty"`

	// Step fields
	StepNumber int `json:"stepNumber,omitempty"`

	// Stream fields
	Part        any            `json:"part,omitempty"`
	StreamParts []any          `json:"streamParts,omitempty"`
	State       map[string]any `json:"state,omitempty"`

	// Output step fields
	FinishReason string         `json:"finishReason,omitempty"`
	ToolCalls    []ToolCallInfo `json:"toolCalls,omitempty"`
	Text         string         `json:"text,omitempty"`

	// InputStep model/tools fields
	Model            ProcessorStepModelConfig `json:"model,omitempty"`
	Tools            ProcessorStepToolsConfig `json:"tools,omitempty"`
	ToolChoice       ToolChoice               `json:"toolChoice,omitempty"`
	ActiveTools      []string                 `json:"activeTools,omitempty"`
	ProviderOptions  SharedProviderOptions    `json:"providerOptions,omitempty"`
	ModelSettings    map[string]any           `json:"modelSettings,omitempty"`
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
	Steps            []StepResult             `json:"steps,omitempty"`
}

// ---------------------------------------------------------------------------
// ProcessorStepOutput
// ---------------------------------------------------------------------------

// ProcessorStepOutput is the output type for processor steps in workflows.
// Uses the flexible schema since outputs may be passed between phases.
type ProcessorStepOutput struct {
	Phase ProcessorPhase `json:"phase"`

	// Message-based fields (used by most phases)
	Messages       []ProcessorMessage `json:"messages,omitempty"`
	MessageList    *MessageList       `json:"-"`
	SystemMessages []CoreMessage      `json:"systemMessages,omitempty"`

	// Step-based fields
	StepNumber *int `json:"stepNumber,omitempty"`

	// Stream-based fields
	Part        any            `json:"part,omitempty"`
	StreamParts []any          `json:"streamParts,omitempty"`
	State       map[string]any `json:"state,omitempty"`

	// Output step fields
	FinishReason string         `json:"finishReason,omitempty"`
	ToolCalls    []ToolCallInfo `json:"toolCalls,omitempty"`
	Text         string         `json:"text,omitempty"`

	// Retry count
	RetryCount *int `json:"retryCount,omitempty"`

	// Model and tools configuration (for inputStep phase)
	Model            MastraLanguageModel      `json:"model,omitempty"`
	Tools            ProcessorStepToolsConfig `json:"tools,omitempty"`
	ToolChoice       ToolChoice               `json:"toolChoice,omitempty"`
	ActiveTools      []string                 `json:"activeTools,omitempty"`
	ProviderOptions  SharedProviderOptions    `json:"providerOptions,omitempty"`
	ModelSettings    map[string]any           `json:"modelSettings,omitempty"`
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
	Steps            []StepResult             `json:"steps,omitempty"`
}

// ProcessorStepData is the discriminated union type for processor step data (input).
type ProcessorStepData = ProcessorStepInput

// ProcessorStepDataFlexible is the flexible type for internal processor code
// that needs to access all fields.
type ProcessorStepDataFlexible = ProcessorStepOutput

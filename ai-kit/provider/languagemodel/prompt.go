// Ported from: packages/provider/src/language-model/v3/language-model-v3-prompt.ts
package languagemodel

import (
	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// Prompt is a list of messages.
//
// Note: Not all models and prompt formats support multi-modal inputs and
// tool calls. The validation happens at runtime.
//
// Note: This is not a user-facing prompt. The AI SDK methods will map the
// user-facing prompt types such as chat or instruction prompts to this format.
type Prompt = []Message

// Message is a sealed interface representing a prompt message.
// Implementations: SystemMessage, UserMessage, AssistantMessage, ToolMessage.
type Message interface {
	messageRole() string
}

// SystemMessage is a system message with text content.
type SystemMessage struct {
	Content         string
	ProviderOptions shared.ProviderOptions
}

func (SystemMessage) messageRole() string { return "system" }

// UserMessage is a user message with text and file parts.
type UserMessage struct {
	Content         []UserMessagePart
	ProviderOptions shared.ProviderOptions
}

func (UserMessage) messageRole() string { return "user" }

// AssistantMessage is an assistant message with various content parts.
type AssistantMessage struct {
	Content         []AssistantMessagePart
	ProviderOptions shared.ProviderOptions
}

func (AssistantMessage) messageRole() string { return "assistant" }

// ToolMessage is a tool message with tool result and approval response parts.
type ToolMessage struct {
	Content         []ToolMessagePart
	ProviderOptions shared.ProviderOptions
}

func (ToolMessage) messageRole() string { return "tool" }

// UserMessagePart is a sealed interface for parts of a user message.
// Implementations: TextPart, FilePart.
type UserMessagePart interface {
	userMessagePart()
}

// AssistantMessagePart is a sealed interface for parts of an assistant message.
// Implementations: TextPart, FilePart, ReasoningPart, ToolCallPart, ToolResultPart.
type AssistantMessagePart interface {
	assistantMessagePart()
}

// ToolMessagePart is a sealed interface for parts of a tool message.
// Implementations: ToolResultPart, ToolApprovalResponsePart.
type ToolMessagePart interface {
	toolMessagePart()
}

// TextPart is a text content part of a prompt.
type TextPart struct {
	Text            string
	ProviderOptions shared.ProviderOptions
}

func (TextPart) userMessagePart()      {}
func (TextPart) assistantMessagePart() {}

// ReasoningPart is a reasoning content part of a prompt.
type ReasoningPart struct {
	Text            string
	ProviderOptions shared.ProviderOptions
}

func (ReasoningPart) assistantMessagePart() {}

// FilePart is a file content part of a prompt.
type FilePart struct {
	// Filename is an optional filename of the file.
	Filename *string

	// Data is the file data. Can be bytes, base64 string, or URL string.
	Data DataContent

	// MediaType is the IANA media type of the file.
	MediaType string

	ProviderOptions shared.ProviderOptions
}

func (FilePart) userMessagePart()      {}
func (FilePart) assistantMessagePart() {}

// ToolCallPart is a tool call content part of a prompt.
type ToolCallPart struct {
	// ToolCallID is the ID used to match the tool call with the tool result.
	ToolCallID string

	// ToolName is the name of the tool that is being called.
	ToolName string

	// Input is the arguments of the tool call. This is a JSON-serializable value.
	Input any

	// ProviderExecuted indicates whether the tool call will be executed by the provider.
	ProviderExecuted *bool

	ProviderOptions shared.ProviderOptions
}

func (ToolCallPart) assistantMessagePart() {}

// ToolResultPart is a tool result content part of a prompt.
type ToolResultPart struct {
	// ToolCallID is the ID of the tool call that this result is associated with.
	ToolCallID string

	// ToolName is the name of the tool that generated this result.
	ToolName string

	// Output is the result of the tool call.
	Output ToolResultOutput

	ProviderOptions shared.ProviderOptions
}

func (ToolResultPart) assistantMessagePart() {}
func (ToolResultPart) toolMessagePart()      {}

// ToolApprovalResponsePart is the user's decision to approve or deny a
// provider-executed tool call.
type ToolApprovalResponsePart struct {
	// ApprovalID is the ID of the approval request this response refers to.
	ApprovalID string

	// Approved indicates whether the approval was granted (true) or denied (false).
	Approved bool

	// Reason is an optional reason for approval or denial.
	Reason *string

	ProviderOptions shared.ProviderOptions
}

func (ToolApprovalResponsePart) toolMessagePart() {}

// ToolResultOutput is a sealed interface representing the output of a tool call.
// Implementations: ToolResultOutputText, ToolResultOutputJSON, ToolResultOutputExecutionDenied,
// ToolResultOutputErrorText, ToolResultOutputErrorJSON, ToolResultOutputContent.
type ToolResultOutput interface {
	toolResultOutputType() string
}

// ToolResultOutputText is text tool output that should be directly sent to the API.
type ToolResultOutputText struct {
	Value           string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultOutputText) toolResultOutputType() string { return "text" }

// ToolResultOutputJSON is JSON tool output.
type ToolResultOutputJSON struct {
	Value           jsonvalue.JSONValue
	ProviderOptions shared.ProviderOptions
}

func (ToolResultOutputJSON) toolResultOutputType() string { return "json" }

// ToolResultOutputExecutionDenied indicates the user has denied execution of the tool call.
type ToolResultOutputExecutionDenied struct {
	// Reason is an optional reason for the execution denial.
	Reason          *string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultOutputExecutionDenied) toolResultOutputType() string { return "execution-denied" }

// ToolResultOutputErrorText is an error text output.
type ToolResultOutputErrorText struct {
	Value           string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultOutputErrorText) toolResultOutputType() string { return "error-text" }

// ToolResultOutputErrorJSON is an error JSON output.
type ToolResultOutputErrorJSON struct {
	Value           jsonvalue.JSONValue
	ProviderOptions shared.ProviderOptions
}

func (ToolResultOutputErrorJSON) toolResultOutputType() string { return "error-json" }

// ToolResultOutputContent is a content output with multiple parts.
type ToolResultOutputContent struct {
	Value []ToolResultContentPart
}

func (ToolResultOutputContent) toolResultOutputType() string { return "content" }

// ToolResultContentPart is a sealed interface for content parts of a tool result.
type ToolResultContentPart interface {
	toolResultContentPartType() string
}

// ToolResultContentText is a text content part.
type ToolResultContentText struct {
	Text            string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultContentText) toolResultContentPartType() string { return "text" }

// ToolResultContentFileData is a file-data content part (base64 encoded media data).
type ToolResultContentFileData struct {
	Data            string
	MediaType       string
	Filename        *string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultContentFileData) toolResultContentPartType() string { return "file-data" }

// ToolResultContentFileURL is a file-url content part.
type ToolResultContentFileURL struct {
	URL             string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultContentFileURL) toolResultContentPartType() string { return "file-url" }

// ToolResultContentFileID is a file-id content part.
// FileID can be a single string or a map from provider name to ID.
type ToolResultContentFileID struct {
	FileID          any // string or map[string]string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultContentFileID) toolResultContentPartType() string { return "file-id" }

// ToolResultContentImageData is an image-data content part.
type ToolResultContentImageData struct {
	Data            string
	MediaType       string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultContentImageData) toolResultContentPartType() string { return "image-data" }

// ToolResultContentImageURL is an image-url content part.
type ToolResultContentImageURL struct {
	URL             string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultContentImageURL) toolResultContentPartType() string { return "image-url" }

// ToolResultContentImageFileID is an image-file-id content part.
// FileID can be a single string or a map from provider name to ID.
type ToolResultContentImageFileID struct {
	FileID          any // string or map[string]string
	ProviderOptions shared.ProviderOptions
}

func (ToolResultContentImageFileID) toolResultContentPartType() string { return "image-file-id" }

// ToolResultContentCustom is a custom content part for provider-specific content.
type ToolResultContentCustom struct {
	ProviderOptions shared.ProviderOptions
}

func (ToolResultContentCustom) toolResultContentPartType() string { return "custom" }

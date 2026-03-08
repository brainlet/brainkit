// Ported from: packages/core/src/agent/message-list/types.ts
package messagelist

import (
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
)

// Re-export types from state for convenience.
type (
	MastraDBMessage        = state.MastraDBMessage
	MastraMessageV1        = state.MastraMessageV1
	MastraMessageContentV2 = state.MastraMessageContentV2
	MastraMessagePart      = state.MastraMessagePart
	UIMessageV4Part        = state.UIMessageV4Part
	MessageSource          = state.MessageSource
	MemoryInfo             = state.MemoryInfo
	UIMessageWithMetadata  = state.UIMessageWithMetadata
	ProviderMetadata       = state.ProviderMetadata
	ToolInvocation         = state.ToolInvocation
)

// CoreMessageV4 is a stub for the AI SDK V4 CoreMessage type.
// TODO: In TS this comes from @internal/ai-sdk-v4 CoreMessage.
// It's a discriminated union with role "system"|"user"|"assistant"|"tool" and various content shapes.
type CoreMessageV4 struct {
	Role                         string           `json:"role"` // "system" | "user" | "assistant" | "tool"
	Content                      any              `json:"content"` // string or []CoreMessageContentPart
	ExperimentalProviderMetadata ProviderMetadata `json:"experimental_providerMetadata,omitempty"`
	ProviderOptions              ProviderMetadata `json:"providerOptions,omitempty"`
	ID                           string           `json:"id,omitempty"`
	Metadata                     map[string]any   `json:"metadata,omitempty"`
}

// CoreMessageContentPart represents a part inside a CoreMessage's content array.
// TODO: Stub — in TS this is a discriminated union from @internal/ai-sdk-v4.
type CoreMessageContentPart struct {
	Type            string         `json:"type"` // "text" | "image" | "file" | "tool-call" | "tool-result" | "reasoning" | "redacted-reasoning"
	Text            string         `json:"text,omitempty"`
	Image           any            `json:"image,omitempty"`
	Data            any            `json:"data,omitempty"`
	MimeType        string         `json:"mimeType,omitempty"`
	MediaType       string         `json:"mediaType,omitempty"`
	Filename        string         `json:"filename,omitempty"`
	ToolCallID      string         `json:"toolCallId,omitempty"`
	ToolName        string         `json:"toolName,omitempty"`
	Args            map[string]any `json:"args,omitempty"`
	Input           any            `json:"input,omitempty"`
	Result          any            `json:"result,omitempty"`
	Output          any            `json:"output,omitempty"`
	Signature       string         `json:"signature,omitempty"`
	ProviderOptions ProviderMetadata `json:"providerOptions,omitempty"`
}

// AIV5UIMessage is a stub for AI SDK V5 UIMessage.
// TODO: In TS this comes from @internal/ai-sdk-v5 UIMessage.
type AIV5UIMessage struct {
	ID       string             `json:"id"`
	Role     string             `json:"role"`
	Parts    []AIV5UIPart       `json:"parts"`
	Metadata map[string]any     `json:"metadata,omitempty"`
}

// AIV5UIPart is a stub for AI SDK V5 UIMessage parts.
// TODO: In TS this is a discriminated union from @internal/ai-sdk-v5.
type AIV5UIPart struct {
	Type             string           `json:"type"`
	Text             string           `json:"text,omitempty"`
	ToolCallID       string           `json:"toolCallId,omitempty"`
	State            string           `json:"state,omitempty"` // "input-streaming" | "input-available" | "output-available" | "output-error"
	Input            any              `json:"input,omitempty"`
	Output           any              `json:"output,omitempty"`
	URL              string           `json:"url,omitempty"`
	MediaType        string           `json:"mediaType,omitempty"`
	Filename         string           `json:"filename,omitempty"`
	SourceID         string           `json:"sourceId,omitempty"`
	Title            string           `json:"title,omitempty"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
	ProviderExecuted *bool            `json:"providerExecuted,omitempty"`
	CallProviderMetadata ProviderMetadata `json:"callProviderMetadata,omitempty"`
	DataPayload      any              `json:"data,omitempty"`
}

// AIV5ModelMessage is a stub for AI SDK V5 ModelMessage.
// TODO: In TS this comes from @internal/ai-sdk-v5 ModelMessage.
type AIV5ModelMessage struct {
	Role            string                   `json:"role"` // "system" | "user" | "assistant" | "tool"
	Content         any                      `json:"content"` // string or []AIV5ModelMessagePart
	ProviderOptions ProviderMetadata         `json:"providerOptions,omitempty"`
	ID              string                   `json:"id,omitempty"`
	Metadata        map[string]any           `json:"metadata,omitempty"`
}

// AIV5ModelMessagePart represents a content part in an AIV5 ModelMessage.
// TODO: Stub — in TS this is a discriminated union from @internal/ai-sdk-v5.
type AIV5ModelMessagePart struct {
	Type            string           `json:"type"` // "text" | "image" | "file" | "tool-call" | "tool-result" | "reasoning"
	Text            string           `json:"text,omitempty"`
	Image           any              `json:"image,omitempty"`
	Data            any              `json:"data,omitempty"`
	MediaType       string           `json:"mediaType,omitempty"`
	Filename        string           `json:"filename,omitempty"`
	ToolCallID      string           `json:"toolCallId,omitempty"`
	ToolName        string           `json:"toolName,omitempty"`
	Input           any              `json:"input,omitempty"`
	Output          any              `json:"output,omitempty"`
	ProviderOptions ProviderMetadata `json:"providerOptions,omitempty"`
}

// AIV5ResponseMessage is a stub for AIV5 AssistantModelMessage | ToolModelMessage.
type AIV5ResponseMessage = AIV5ModelMessage

// MessageInput represents any supported message input format.
type MessageInput interface{}

// MessageListInput can be a string, []string, MessageInput, or []MessageInput.
type MessageListInput interface{}

// IdGeneratorContext is re-exported from the types package.
// Ported from: packages/core/src/types/dynamic-argument.ts — IdGeneratorContext
type IdGeneratorContext = aktypes.IdGeneratorContext

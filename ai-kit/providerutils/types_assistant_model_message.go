// Ported from: packages/provider-utils/src/types/assistant-model-message.ts
package providerutils

// AssistantModelMessage represents an assistant message. It can contain text,
// tool calls, or a combination of text and tool calls.
type AssistantModelMessage struct {
	Role            string           `json:"role"` // "assistant"
	Content         AssistantContent `json:"content"`
	ProviderOptions ProviderOptions  `json:"providerOptions,omitempty"`
}

// AssistantContent is the content of an assistant message.
// It can be a string or an array of content parts (TextPart, FilePart, ReasoningPart,
// ToolCallPart, ToolResultPart, ToolApprovalRequest).
// In Go we represent this as interface{} since it's a union type.
type AssistantContent = interface{}

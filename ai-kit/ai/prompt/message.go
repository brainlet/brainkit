// Ported from: packages/ai/src/prompt/message.ts
package prompt

// SystemModelMessage represents a system message.
// TODO: import from brainlink/experiments/ai-kit/providerutils once it exists
type SystemModelMessage struct {
	Role            string          `json:"role"` // always "system"
	Content         string          `json:"content"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// UserModelMessage represents a user message.
// Content can be a string or a slice of content parts (TextPart, ImagePart, FilePart).
type UserModelMessage struct {
	Role            string          `json:"role"` // always "user"
	Content         interface{}     `json:"content"` // string or []interface{} (TextPart|ImagePart|FilePart)
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// AssistantModelMessage represents an assistant message.
// Content can be a string or a slice of content parts.
type AssistantModelMessage struct {
	Role            string          `json:"role"` // always "assistant"
	Content         interface{}     `json:"content"` // string or []interface{}
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// ToolModelMessage represents a tool message.
type ToolModelMessage struct {
	Role            string          `json:"role"` // always "tool"
	Content         []interface{}   `json:"content"` // ToolResultPart or ToolApprovalResponse
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// ModelMessage represents any message type in the conversation.
// In TypeScript this is a union type; in Go we use an interface{} that can be
// any of: SystemModelMessage, UserModelMessage, AssistantModelMessage, ToolModelMessage.
// Use GetMessageRole() to determine the type.
type ModelMessage = interface{}

// GetMessageRole extracts the role from a ModelMessage.
func GetMessageRole(msg interface{}) string {
	switch m := msg.(type) {
	case SystemModelMessage:
		return "system"
	case *SystemModelMessage:
		return "system"
	case UserModelMessage:
		return "user"
	case *UserModelMessage:
		return "user"
	case AssistantModelMessage:
		return "assistant"
	case *AssistantModelMessage:
		return "assistant"
	case ToolModelMessage:
		return "tool"
	case *ToolModelMessage:
		return "tool"
	case map[string]interface{}:
		if role, ok := m["role"].(string); ok {
			return role
		}
	}
	return ""
}

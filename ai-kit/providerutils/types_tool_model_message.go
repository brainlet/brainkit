// Ported from: packages/provider-utils/src/types/tool-model-message.ts
package providerutils

// ToolModelMessage represents a tool message. It contains the result of one
// or more tool calls.
type ToolModelMessage struct {
	Role            string          `json:"role"` // "tool"
	Content         ToolContent     `json:"content"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// ToolContent is the content of a tool message.
// It is an array of tool result parts and/or tool approval responses.
// In Go we represent this as a slice of interface{} since it's a union type.
type ToolContent = []interface{}

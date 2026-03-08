// Ported from: packages/provider-utils/src/types/tool-call.ts
package providerutils

// ToolCall represents a typed tool call returned by generateText and streamText.
// It contains the tool call ID, the tool name, and the tool arguments.
type ToolCall[INPUT any] struct {
	// ToolCallID is the ID of the tool call, used to match with tool results.
	ToolCallID string `json:"toolCallId"`
	// ToolName is the name of the tool being called.
	ToolName string `json:"toolName"`
	// Input contains the arguments of the tool call.
	Input INPUT `json:"input"`
	// ProviderExecuted indicates whether the tool call will be executed by the provider.
	ProviderExecuted bool `json:"providerExecuted,omitempty"`
	// Dynamic indicates whether the tool is dynamic.
	Dynamic bool `json:"dynamic,omitempty"`
}

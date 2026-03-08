// Ported from: packages/provider-utils/src/types/tool-result.ts
package providerutils

// ToolResult represents a typed tool result returned by generateText and streamText.
// It contains the tool call ID, the tool name, the tool arguments, and the tool result.
type ToolResult[INPUT any, OUTPUT any] struct {
	// ToolCallID is the ID of the tool call, used to match with tool calls.
	ToolCallID string `json:"toolCallId"`
	// ToolName is the name of the tool that was called.
	ToolName string `json:"toolName"`
	// Input contains the arguments of the tool call.
	Input INPUT `json:"input"`
	// Output is the result of the tool call execution.
	Output OUTPUT `json:"output"`
	// ProviderExecuted indicates whether the tool result was executed by the provider.
	ProviderExecuted bool `json:"providerExecuted,omitempty"`
	// Dynamic indicates whether the tool is dynamic.
	Dynamic bool `json:"dynamic,omitempty"`
}

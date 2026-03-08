// Ported from: packages/ai/src/generate-text/tool-call-repair-function.ts
package generatetext

// ToolCallRepairFunction attempts to repair a tool call that failed to parse.
//
// It receives the error and context as arguments and returns the repaired
// tool call or nil if repair is not possible.
type ToolCallRepairFunction func(options ToolCallRepairOptions) (*LanguageModelV4ToolCall, error)

// ToolCallRepairOptions contains the context for a tool call repair attempt.
type ToolCallRepairOptions struct {
	// System is the system prompt.
	System interface{} // string | SystemModelMessage | []SystemModelMessage | nil

	// Messages are the messages in the current generation step.
	Messages []ModelMessage

	// ToolCall is the tool call that failed to parse.
	ToolCall LanguageModelV4ToolCall

	// Tools are the tools that are available.
	Tools ToolSet

	// InputSchema returns the JSON Schema for a tool.
	InputSchema func(toolName string) (JSONSchema7, error)

	// Error is the error that occurred while parsing the tool call.
	Error error
}

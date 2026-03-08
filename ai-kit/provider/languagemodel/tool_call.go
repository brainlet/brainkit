// Ported from: packages/provider/src/language-model/v3/language-model-v3-tool-call.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// ToolCall represents tool calls that the model has generated.
type ToolCall struct {
	// ToolCallID is the identifier of the tool call. Must be unique across all tool calls.
	ToolCallID string

	// ToolName is the name of the tool that should be called.
	ToolName string

	// Input is a stringified JSON object with the tool call arguments.
	// Must match the parameters schema of the tool.
	Input string

	// ProviderExecuted indicates whether the tool call will be executed by the provider.
	// If nil or false, the tool call will be executed by the client.
	ProviderExecuted *bool

	// Dynamic indicates whether the tool is dynamic, i.e. defined at runtime.
	// For example, MCP tools that are executed by the provider.
	Dynamic *bool

	// ProviderMetadata is additional provider-specific metadata for the tool call.
	ProviderMetadata shared.ProviderMetadata
}

func (ToolCall) isContent()    {}
func (ToolCall) isStreamPart() {}

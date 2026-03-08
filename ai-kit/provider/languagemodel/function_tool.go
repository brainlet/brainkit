// Ported from: packages/provider/src/language-model/v3/language-model-v3-function-tool.ts
package languagemodel

import (
	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// FunctionTool defines a tool with a name, description, and set of parameters.
//
// Note: this is not the user-facing tool definition. The AI SDK methods will
// map the user-facing tool definitions to this format.
type FunctionTool struct {
	// Name of the tool. Unique within this model call.
	Name string

	// Description of the tool. The language model uses this to understand
	// the tool's purpose and to provide better completion suggestions.
	Description *string

	// InputSchema is the JSON Schema that the tool expects.
	// The language model uses this to understand the tool's input requirements.
	InputSchema map[string]any

	// InputExamples is an optional list of input examples.
	InputExamples []FunctionToolInputExample

	// Strict mode setting for the tool. Providers that support strict mode
	// will use this to determine how the input should be generated.
	Strict *bool

	// ProviderOptions is the provider-specific options for the tool.
	ProviderOptions shared.ProviderOptions
}

// FunctionToolInputExample is an example input for a function tool.
type FunctionToolInputExample struct {
	Input jsonvalue.JSONObject
}

// Tool is a sealed interface for tools that can be provided to the language model.
// Implementations: FunctionTool, ProviderTool.
type Tool interface {
	toolType() string
}

func (FunctionTool) toolType() string { return "function" }

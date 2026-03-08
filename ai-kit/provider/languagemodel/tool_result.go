// Ported from: packages/provider/src/language-model/v3/language-model-v3-tool-result.ts
package languagemodel

import (
	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// ToolResult represents the result of a tool call that has been executed by the provider.
type ToolResult struct {
	// ToolCallID is the ID of the tool call that this result is associated with.
	ToolCallID string

	// ToolName is the name of the tool that generated this result.
	ToolName string

	// Result is the result of the tool call. This is a JSON-serializable value (non-nil).
	Result jsonvalue.JSONValue

	// IsError is an optional flag if the result is an error or an error message.
	IsError *bool

	// Preliminary indicates whether the tool result is preliminary.
	// Preliminary tool results replace each other, e.g. image previews.
	// There always has to be a final, non-preliminary tool result.
	Preliminary *bool

	// Dynamic indicates whether the tool is dynamic, i.e. defined at runtime.
	Dynamic *bool

	// ProviderMetadata is additional provider-specific metadata for the tool result.
	ProviderMetadata shared.ProviderMetadata
}

func (ToolResult) isContent()    {}
func (ToolResult) isStreamPart() {}

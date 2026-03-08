// Ported from: packages/anthropic/src/tool/tool-search-regex_20251119.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// ToolSearchRegex20251119Input is the input schema for the tool_search_regex_20251119 tool.
type ToolSearchRegex20251119Input struct {
	Pattern string `json:"pattern"`
	Limit   *int   `json:"limit,omitempty"`
}

// ToolSearchRegex20251119OutputItem represents a tool reference result.
type ToolSearchRegex20251119OutputItem struct {
	Type     string `json:"type"` // "tool_reference"
	ToolName string `json:"toolName"`
}

// ToolSearchRegex20251119Output is the output type for the tool_search_regex_20251119 tool.
type ToolSearchRegex20251119Output = []ToolSearchRegex20251119OutputItem

// ToolSearchRegex20251119 is the provider tool factory for the tool_search_regex_20251119 tool.
var ToolSearchRegex20251119 = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[ToolSearchRegex20251119Input, ToolSearchRegex20251119Output]{
		ID:                      "anthropic.tool_search_regex_20251119",
		SupportsDeferredResults: true,
	},
)

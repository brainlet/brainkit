// Ported from: packages/anthropic/src/tool/tool-search-bm25_20251119.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// ToolSearchBm2520251119Input is the input schema for the tool_search_bm25_20251119 tool.
type ToolSearchBm2520251119Input struct {
	Query string `json:"query"`
	Limit *int   `json:"limit,omitempty"`
}

// ToolSearchBm2520251119OutputItem represents a tool reference result.
type ToolSearchBm2520251119OutputItem struct {
	Type     string `json:"type"` // "tool_reference"
	ToolName string `json:"toolName"`
}

// ToolSearchBm2520251119Output is the output type for the tool_search_bm25_20251119 tool.
type ToolSearchBm2520251119Output = []ToolSearchBm2520251119OutputItem

// ToolSearchBm2520251119 is the provider tool factory for the tool_search_bm25_20251119 tool.
var ToolSearchBm2520251119 = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[ToolSearchBm2520251119Input, ToolSearchBm2520251119Output]{
		ID:                      "anthropic.tool_search_bm25_20251119",
		SupportsDeferredResults: true,
	},
)

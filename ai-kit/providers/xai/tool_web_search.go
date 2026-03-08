// Ported from: packages/xai/src/tool/web-search.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// WebSearchInput is the input for the web search tool (empty, args are passed via ProviderTool.Args).
type WebSearchInput struct{}

// WebSearchOutput is the output of the web search tool.
type WebSearchOutput struct {
	Query   string              `json:"query"`
	Sources []WebSearchSource   `json:"sources"`
}

// WebSearchSource is a single web search source.
type WebSearchSource struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// WebSearchArgs are the arguments for the web search tool.
type WebSearchArgs struct {
	// AllowedDomains is a list of allowed domains (max 5).
	AllowedDomains []string `json:"allowedDomains,omitempty"`
	// ExcludedDomains is a list of excluded domains (max 5).
	ExcludedDomains []string `json:"excludedDomains,omitempty"`
	// EnableImageUnderstanding enables image understanding.
	EnableImageUnderstanding *bool `json:"enableImageUnderstanding,omitempty"`
}

// webSearchToolFactory is the factory for the web search tool.
var webSearchToolFactory = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[WebSearchInput, WebSearchOutput]{
		ID:           "xai.web_search",
		InputSchema:  &providerutils.Schema[WebSearchInput]{},
		OutputSchema: &providerutils.Schema[WebSearchOutput]{},
	},
)

// WebSearch creates a web search provider tool.
func WebSearch(opts ...providerutils.ProviderToolOptions[WebSearchInput, WebSearchOutput]) providerutils.ProviderTool[WebSearchInput, WebSearchOutput] {
	var o providerutils.ProviderToolOptions[WebSearchInput, WebSearchOutput]
	if len(opts) > 0 {
		o = opts[0]
	}
	return webSearchToolFactory(o)
}

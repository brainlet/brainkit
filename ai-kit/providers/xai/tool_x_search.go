// Ported from: packages/xai/src/tool/x-search.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// XSearchInput is the input for the X search tool (empty, args are passed via ProviderTool.Args).
type XSearchInput struct{}

// XSearchOutput is the output of the X search tool.
type XSearchOutput struct {
	Query string       `json:"query"`
	Posts []XSearchPost `json:"posts"`
}

// XSearchPost is a single X search post.
type XSearchPost struct {
	Author string `json:"author"`
	Text   string `json:"text"`
	URL    string `json:"url"`
	Likes  int    `json:"likes"`
}

// XSearchArgs are the arguments for the X search tool.
type XSearchArgs struct {
	// AllowedXHandles is a list of allowed X handles (max 10).
	AllowedXHandles []string `json:"allowedXHandles,omitempty"`
	// ExcludedXHandles is a list of excluded X handles (max 10).
	ExcludedXHandles []string `json:"excludedXHandles,omitempty"`
	// FromDate is the start date filter.
	FromDate *string `json:"fromDate,omitempty"`
	// ToDate is the end date filter.
	ToDate *string `json:"toDate,omitempty"`
	// EnableImageUnderstanding enables image understanding.
	EnableImageUnderstanding *bool `json:"enableImageUnderstanding,omitempty"`
	// EnableVideoUnderstanding enables video understanding.
	EnableVideoUnderstanding *bool `json:"enableVideoUnderstanding,omitempty"`
}

// xSearchToolFactory is the factory for the X search tool.
var xSearchToolFactory = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[XSearchInput, XSearchOutput]{
		ID:           "xai.x_search",
		InputSchema:  &providerutils.Schema[XSearchInput]{},
		OutputSchema: &providerutils.Schema[XSearchOutput]{},
	},
)

// XSearch creates an X search provider tool.
func XSearch(opts ...providerutils.ProviderToolOptions[XSearchInput, XSearchOutput]) providerutils.ProviderTool[XSearchInput, XSearchOutput] {
	var o providerutils.ProviderToolOptions[XSearchInput, XSearchOutput]
	if len(opts) > 0 {
		o = opts[0]
	}
	return xSearchToolFactory(o)
}

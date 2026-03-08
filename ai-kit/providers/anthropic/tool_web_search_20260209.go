// Ported from: packages/anthropic/src/tool/web-search_20260209.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// WebSearch20260209Args is the args schema for the web_search_20260209 tool.
type WebSearch20260209Args struct {
	MaxUses        *int            `json:"maxUses,omitempty"`
	AllowedDomains []string        `json:"allowedDomains,omitempty"`
	BlockedDomains []string        `json:"blockedDomains,omitempty"`
	UserLocation   *map[string]any `json:"userLocation,omitempty"`
}

// WebSearch20260209Input is the input schema for the web_search_20260209 tool.
type WebSearch20260209Input struct {
	Query string `json:"query"`
}

// WebSearch20260209OutputItem represents a single web search result.
type WebSearch20260209OutputItem struct {
	Type             string  `json:"type"` // "web_search_result"
	URL              string  `json:"url"`
	Title            *string `json:"title"`
	PageAge          *string `json:"pageAge"`
	EncryptedContent string  `json:"encryptedContent"`
}

// WebSearch20260209Output is the output type for the web_search_20260209 tool.
type WebSearch20260209Output = []WebSearch20260209OutputItem

// WebSearch20260209 is the provider tool factory for the web_search_20260209 tool.
var WebSearch20260209 = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[WebSearch20260209Input, WebSearch20260209Output]{
		ID:                      "anthropic.web_search_20260209",
		SupportsDeferredResults: true,
	},
)

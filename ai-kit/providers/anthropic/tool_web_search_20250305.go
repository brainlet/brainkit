// Ported from: packages/anthropic/src/tool/web-search_20250305.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// WebSearch20250305Args is the args schema for the web_search_20250305 tool.
type WebSearch20250305Args struct {
	MaxUses        *int            `json:"maxUses,omitempty"`
	AllowedDomains []string        `json:"allowedDomains,omitempty"`
	BlockedDomains []string        `json:"blockedDomains,omitempty"`
	UserLocation   *map[string]any `json:"userLocation,omitempty"`
}

// WebSearch20250305Input is the input schema for the web_search_20250305 tool.
type WebSearch20250305Input struct {
	Query string `json:"query"`
}

// WebSearch20250305OutputItem represents a single web search result.
type WebSearch20250305OutputItem struct {
	Type             string  `json:"type"` // "web_search_result"
	URL              string  `json:"url"`
	Title            *string `json:"title"`
	PageAge          *string `json:"pageAge"`
	EncryptedContent string  `json:"encryptedContent"`
}

// WebSearch20250305Output is the output type for the web_search_20250305 tool.
type WebSearch20250305Output = []WebSearch20250305OutputItem

// WebSearch20250305 is the provider tool factory for the web_search_20250305 tool.
var WebSearch20250305 = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[WebSearch20250305Input, WebSearch20250305Output]{
		ID:                      "anthropic.web_search_20250305",
		SupportsDeferredResults: true,
	},
)

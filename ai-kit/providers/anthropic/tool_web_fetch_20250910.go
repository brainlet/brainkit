// Ported from: packages/anthropic/src/tool/web-fetch-20250910.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// WebFetch20250910Args is the args schema for the web_fetch_20250910 tool.
type WebFetch20250910Args struct {
	MaxUses          *int            `json:"maxUses,omitempty"`
	AllowedDomains   []string        `json:"allowedDomains,omitempty"`
	BlockedDomains   []string        `json:"blockedDomains,omitempty"`
	Citations        *map[string]any `json:"citations,omitempty"`
	MaxContentTokens *int            `json:"maxContentTokens,omitempty"`
}

// WebFetch20250910Input is the input schema for the web_fetch_20250910 tool.
type WebFetch20250910Input struct {
	URL string `json:"url"`
}

// WebFetch20250910Output is the output schema for the web_fetch_20250910 tool.
type WebFetch20250910Output struct {
	Type        string         `json:"type"` // "web_fetch_result"
	URL         string         `json:"url"`
	Content     map[string]any `json:"content"`
	RetrievedAt *string        `json:"retrievedAt"`
}

// WebFetch20250910 is the provider tool factory for the web_fetch_20250910 tool.
var WebFetch20250910 = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[WebFetch20250910Input, WebFetch20250910Output]{
		ID:                      "anthropic.web_fetch_20250910",
		SupportsDeferredResults: true,
	},
)

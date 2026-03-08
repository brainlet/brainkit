// Ported from: packages/anthropic/src/tool/web-fetch-20260209.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// WebFetch20260209Args is the args schema for the web_fetch_20260209 tool.
type WebFetch20260209Args struct {
	MaxUses          *int            `json:"maxUses,omitempty"`
	AllowedDomains   []string        `json:"allowedDomains,omitempty"`
	BlockedDomains   []string        `json:"blockedDomains,omitempty"`
	Citations        *map[string]any `json:"citations,omitempty"`
	MaxContentTokens *int            `json:"maxContentTokens,omitempty"`
}

// WebFetch20260209Input is the input schema for the web_fetch_20260209 tool.
type WebFetch20260209Input struct {
	URL string `json:"url"`
}

// WebFetch20260209Output is the output schema for the web_fetch_20260209 tool.
type WebFetch20260209Output struct {
	Type        string         `json:"type"` // "web_fetch_result"
	URL         string         `json:"url"`
	Content     map[string]any `json:"content"`
	RetrievedAt *string        `json:"retrievedAt"`
}

// WebFetch20260209 is the provider tool factory for the web_fetch_20260209 tool.
var WebFetch20260209 = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[WebFetch20260209Input, WebFetch20260209Output]{
		ID:                      "anthropic.web_fetch_20260209",
		SupportsDeferredResults: true,
	},
)

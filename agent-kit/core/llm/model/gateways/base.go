// Ported from: packages/core/src/llm/model/gateways/base.ts
package gateways

// ---------------------------------------------------------------------------
// ProviderConfig
// ---------------------------------------------------------------------------

// ProviderConfig holds configuration for a model provider.
type ProviderConfig struct {
	// URL is the base URL for the provider's API.
	URL string `json:"url,omitempty"`
	// APIKeyHeader is the HTTP header name for the API key.
	APIKeyHeader string `json:"apiKeyHeader,omitempty"`
	// APIKeyEnvVar is the environment variable(s) for the API key.
	// Can be a single string or a list of alternatives.
	APIKeyEnvVar any `json:"apiKeyEnvVar"`
	// Name is the display name of the provider.
	Name string `json:"name"`
	// Models is the list of supported model identifiers.
	Models []string `json:"models"`
	// DocURL is an optional documentation URL.
	DocURL string `json:"docUrl,omitempty"`
	// Gateway is the gateway that sourced this provider.
	Gateway string `json:"gateway"`
	// NPM is the NPM package name from models.dev (e.g., "@ai-sdk/anthropic").
	NPM string `json:"npm,omitempty"`
}

// APIKeyEnvVarStrings returns the API key environment variable names as a string slice.
func (c ProviderConfig) APIKeyEnvVarStrings() []string {
	switch v := c.APIKeyEnvVar.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// GatewayLanguageModel
// ---------------------------------------------------------------------------

// GatewayLanguageModel is a union type for language models returned by gateways.
// Supports both AI SDK v5 (LanguageModelV2) and v6 (LanguageModelV3).
// In Go, this is represented as an interface.
type GatewayLanguageModel interface {
	SpecificationVersion() string
	Provider() string
	ModelID() string
}

// ---------------------------------------------------------------------------
// ResolveLanguageModelArgs
// ---------------------------------------------------------------------------

// ResolveLanguageModelArgs contains the arguments for resolving a language model.
type ResolveLanguageModelArgs struct {
	ModelID    string            `json:"modelId"`
	ProviderID string            `json:"providerId"`
	APIKey     string            `json:"apiKey"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// ---------------------------------------------------------------------------
// MastraModelGateway
// ---------------------------------------------------------------------------

// MastraModelGateway is the interface for model gateway providers.
// Gateways fetch provider configurations and build URLs for model access.
type MastraModelGateway interface {
	// ID returns the unique identifier for the gateway.
	// This ID is used as the prefix for all providers from this gateway
	// (e.g., "netlify" for netlify gateway).
	// Exception: models.dev is a provider registry and doesn't use a prefix.
	ID() string

	// Name returns the display name of the gateway provider.
	Name() string

	// FetchProviders fetches provider configurations from the gateway.
	FetchProviders() (map[string]ProviderConfig, error)

	// BuildURL builds the URL for a specific model/provider combination.
	// Returns the URL string if this gateway can handle the model,
	// or empty string otherwise.
	BuildURL(modelID string, envVars map[string]string) (string, error)

	// GetAPIKey retrieves the API key for a model.
	GetAPIKey(modelID string) (string, error)

	// ResolveLanguageModel resolves a language model from the gateway.
	// Supports returning either LanguageModelV2 (AI SDK v5) or LanguageModelV3 (AI SDK v6).
	ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error)
}

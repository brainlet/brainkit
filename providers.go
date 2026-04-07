package brainkit

import "github.com/brainlet/brainkit/internal/types"

// ProviderConfig configures an AI provider. Create with OpenAI(), Anthropic(), etc.
type ProviderConfig struct {
	name    string
	typ     string
	apiKey  string
	baseURL string
	headers map[string]string
}

// ProviderOption configures a provider constructor.
type ProviderOption func(*ProviderConfig)

// WithBaseURL overrides the default API endpoint.
func WithBaseURL(url string) ProviderOption {
	return func(c *ProviderConfig) { c.baseURL = url }
}

// WithHeaders adds custom HTTP headers to provider requests.
func WithHeaders(headers map[string]string) ProviderOption {
	return func(c *ProviderConfig) { c.headers = headers }
}

func newProvider(name, typ, apiKey string, opts []ProviderOption) ProviderConfig {
	c := ProviderConfig{name: name, typ: typ, apiKey: apiKey}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// toConfig converts ProviderConfig to the concrete provider config struct
// expected by the engine's provider registry.
func (p ProviderConfig) toConfig() any {
	switch p.typ {
	case "openai":
		return types.OpenAIProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "anthropic":
		return types.AnthropicProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "google":
		return types.GoogleProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "mistral":
		return types.MistralProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "cohere":
		return types.CohereProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "groq":
		return types.GroqProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "perplexity":
		return types.PerplexityProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "deepseek":
		return types.DeepSeekProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "fireworks":
		return types.FireworksProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "togetherai":
		return types.TogetherAIProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "xai":
		return types.XAIProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	case "cerebras":
		return types.CerebrasProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	default:
		return types.OpenAIProviderConfig{APIKey: p.apiKey, BaseURL: p.baseURL, Headers: p.headers}
	}
}

// Provider constructors — one per supported AI provider.

func OpenAI(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("openai", "openai", apiKey, opts)
}

func Anthropic(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("anthropic", "anthropic", apiKey, opts)
}

func Google(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("google", "google", apiKey, opts)
}

func Mistral(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("mistral", "mistral", apiKey, opts)
}

func Groq(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("groq", "groq", apiKey, opts)
}

func DeepSeek(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("deepseek", "deepseek", apiKey, opts)
}

func XAI(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("xai", "xai", apiKey, opts)
}

func Cohere(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("cohere", "cohere", apiKey, opts)
}

func Perplexity(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("perplexity", "perplexity", apiKey, opts)
}

func TogetherAI(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("togetherai", "togetherai", apiKey, opts)
}

func Fireworks(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("fireworks", "fireworks", apiKey, opts)
}

func Cerebras(apiKey string, opts ...ProviderOption) ProviderConfig {
	return newProvider("cerebras", "cerebras", apiKey, opts)
}

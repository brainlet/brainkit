package registry

// AI provider config structs — one per provider, matching AI SDK constructor signatures.

// OpenAIProviderConfig configures the OpenAI provider.
type OpenAIProviderConfig struct {
	APIKey       string
	BaseURL      string // default: https://api.openai.com/v1
	Organization string
	Project      string
	Headers      map[string]string
}

// AnthropicProviderConfig configures the Anthropic provider.
type AnthropicProviderConfig struct {
	APIKey    string
	AuthToken string // alternative: Bearer auth instead of x-api-key
	BaseURL   string // default: https://api.anthropic.com/v1
	Headers   map[string]string
}

// GoogleProviderConfig configures the Google Generative AI provider.
type GoogleProviderConfig struct {
	APIKey  string
	BaseURL string // default: https://generativelanguage.googleapis.com/v1beta
	Headers map[string]string
}

// MistralProviderConfig configures the Mistral provider.
type MistralProviderConfig struct {
	APIKey  string
	BaseURL string // default: https://api.mistral.ai/v1
	Headers map[string]string
}

// CohereProviderConfig configures the Cohere provider.
type CohereProviderConfig struct {
	APIKey  string
	BaseURL string // default: https://api.cohere.com/v2
	Headers map[string]string
}

// GroqProviderConfig configures the Groq provider.
type GroqProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

// PerplexityProviderConfig configures the Perplexity provider.
type PerplexityProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

// DeepSeekProviderConfig configures the DeepSeek provider.
type DeepSeekProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

// FireworksProviderConfig configures the Fireworks provider.
type FireworksProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

// TogetherAIProviderConfig configures the Together AI provider.
type TogetherAIProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

// XAIProviderConfig configures the xAI provider.
type XAIProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

// AzureProviderConfig configures the Azure OpenAI provider.
type AzureProviderConfig struct {
	APIKey       string
	ResourceName string // builds URL: https://{name}.openai.azure.com/openai/v1
	BaseURL      string // alternative to ResourceName
	Headers      map[string]string
}

// BedrockProviderConfig configures the Amazon Bedrock provider.
type BedrockProviderConfig struct {
	Region    string
	AccessKey string
	SecretKey string
	Headers   map[string]string
}

// VertexProviderConfig configures the Google Vertex AI provider.
type VertexProviderConfig struct {
	Project  string
	Location string
	APIKey   string
	Headers  map[string]string
}

// HuggingFaceProviderConfig configures the Hugging Face provider.
type HuggingFaceProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

// CerebrasProviderConfig configures the Cerebras provider.
type CerebrasProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

package engine

import (
	"os"

	provreg "github.com/brainlet/brainkit/internal/providers"
	"github.com/brainlet/brainkit/internal/types"
)

// autoDetectProviders scans os.Getenv and cfg.EnvVars for known API key patterns
// and registers AI providers that aren't already explicitly configured.
// Priority: explicit AIProviders > EnvVars > os.Getenv.
func autoDetectProviders(cfg *types.KernelConfig) {
	if cfg.AIProviders == nil {
		cfg.AIProviders = make(map[string]provreg.AIProviderRegistration)
	}

	type providerMapping struct {
		name string
		typ  provreg.AIProviderType
		make func(apiKey string) any
	}

	mappings := map[string]providerMapping{
		"OPENAI_API_KEY":     {"openai", provreg.AIProviderOpenAI, func(k string) any { return provreg.OpenAIProviderConfig{APIKey: k} }},
		"ANTHROPIC_API_KEY":  {"anthropic", provreg.AIProviderAnthropic, func(k string) any { return provreg.AnthropicProviderConfig{APIKey: k} }},
		"GOOGLE_API_KEY":     {"google", provreg.AIProviderGoogle, func(k string) any { return provreg.GoogleProviderConfig{APIKey: k} }},
		"MISTRAL_API_KEY":    {"mistral", provreg.AIProviderMistral, func(k string) any { return provreg.MistralProviderConfig{APIKey: k} }},
		"GROQ_API_KEY":       {"groq", provreg.AIProviderGroq, func(k string) any { return provreg.GroqProviderConfig{APIKey: k} }},
		"DEEPSEEK_API_KEY":   {"deepseek", provreg.AIProviderDeepSeek, func(k string) any { return provreg.DeepSeekProviderConfig{APIKey: k} }},
		"XAI_API_KEY":        {"xai", provreg.AIProviderXAI, func(k string) any { return provreg.XAIProviderConfig{APIKey: k} }},
		"COHERE_API_KEY":     {"cohere", provreg.AIProviderCohere, func(k string) any { return provreg.CohereProviderConfig{APIKey: k} }},
		"PERPLEXITY_API_KEY": {"perplexity", provreg.AIProviderPerplexity, func(k string) any { return provreg.PerplexityProviderConfig{APIKey: k} }},
		"TOGETHER_API_KEY":   {"togetherai", provreg.AIProviderTogetherAI, func(k string) any { return provreg.TogetherAIProviderConfig{APIKey: k} }},
		"FIREWORKS_API_KEY":  {"fireworks", provreg.AIProviderFireworks, func(k string) any { return provreg.FireworksProviderConfig{APIKey: k} }},
		"CEREBRAS_API_KEY":   {"cerebras", provreg.AIProviderCerebras, func(k string) any { return provreg.CerebrasProviderConfig{APIKey: k} }},
	}

	for envKey, mapping := range mappings {
		if _, explicit := cfg.AIProviders[mapping.name]; explicit {
			continue
		}
		apiKey := ""
		if v, ok := cfg.EnvVars[envKey]; ok && v != "" {
			apiKey = v
		} else {
			apiKey = os.Getenv(envKey)
		}
		if apiKey == "" {
			continue
		}
		cfg.AIProviders[mapping.name] = provreg.AIProviderRegistration{
			Type:   mapping.typ,
			Config: mapping.make(apiKey),
		}
	}
}

// extractProviderCredentials extracts APIKey and BaseURL from a typed provider registration.
func extractProviderCredentials(reg provreg.AIProviderRegistration) struct{ APIKey, BaseURL string } {
	switch cfg := reg.Config.(type) {
	case provreg.OpenAIProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.AnthropicProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.GoogleProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.MistralProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.CohereProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.GroqProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.PerplexityProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.DeepSeekProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.FireworksProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.TogetherAIProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.XAIProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.AzureProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.HuggingFaceProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.CerebrasProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.VertexProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, ""}
	case provreg.BedrockProviderConfig:
		return struct{ APIKey, BaseURL string }{"", ""}
	default:
		return struct{ APIKey, BaseURL string }{}
	}
}

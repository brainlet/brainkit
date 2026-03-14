package brainkit

// Config configures a Kit.
type Config struct {
	// Providers maps provider names to AI model configs.
	Providers map[string]ProviderConfig

	// EnvVars injected into sandboxes.
	EnvVars map[string]string
}

// ProviderConfig configures an AI provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string
}

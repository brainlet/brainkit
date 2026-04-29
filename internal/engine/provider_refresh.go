package engine

// defaultProviderKeyMapping is the built-in mapping from secret names to AI
// provider names. Used by RefreshProviderIfSecret when
// KernelConfig.ProviderKeyMapping is nil.
var defaultProviderKeyMapping = map[string]string{
	"OPENAI_API_KEY":    "openai",
	"ANTHROPIC_API_KEY": "anthropic",
	"GOOGLE_API_KEY":    "google",
	"MISTRAL_API_KEY":   "mistral",
	"GROQ_API_KEY":      "groq",
	"DEEPSEEK_API_KEY":  "deepseek",
	"XAI_API_KEY":       "xai",
	"COHERE_API_KEY":    "cohere",
}

// RefreshProviderIfSecret checks if a secret name matches a provider key
// pattern and refreshes the JS-side provider cache if so. Uses
// KernelConfig.ProviderKeyMapping if set, otherwise defaultProviderKeyMapping.
func (k *Kernel) RefreshProviderIfSecret(name, newValue string) {
	mapping := k.config.ProviderKeyMapping
	if mapping == nil {
		mapping = defaultProviderKeyMapping
	}

	provName, ok := mapping[name]
	if !ok {
		return
	}

	k.callJSSync("__brainkit.secrets.refreshProvider", map[string]string{
		"provider": provName,
		"apiKey":   newValue,
	})
}

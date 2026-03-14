package aiembed

// MiddlewareConfig defines a middleware to wrap around a language model.
type MiddlewareConfig struct {
	// Type is the middleware type: "defaultSettings" or "extractReasoning".
	Type string `json:"type"`

	// Settings is used when Type is "defaultSettings".
	// Specifies default call settings that can be overridden per-call.
	Settings *MiddlewareSettings `json:"settings,omitempty"`

	// TagName is used when Type is "extractReasoning".
	// The XML tag name to extract reasoning from (default: "thinking").
	TagName string `json:"tagName,omitempty"`

	// Separator is used when Type is "extractReasoning".
	// Separator between reasoning and response text.
	Separator string `json:"separator,omitempty"`
}

// MiddlewareSettings defines default settings for the defaultSettings middleware.
type MiddlewareSettings struct {
	MaxTokens        int                               `json:"maxTokens,omitempty"`
	Temperature      *float64                          `json:"temperature,omitempty"`
	TopP             *float64                          `json:"topP,omitempty"`
	TopK             *int                              `json:"topK,omitempty"`
	PresencePenalty  *float64                          `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float64                          `json:"frequencyPenalty,omitempty"`
	StopSequences    []string                          `json:"stopSequences,omitempty"`
	Seed             *int                              `json:"seed,omitempty"`
	ProviderOptions  map[string]map[string]interface{} `json:"providerMetadata,omitempty"`
}

// DefaultSettingsMiddleware creates a middleware config that applies default settings.
func DefaultSettingsMiddleware(settings MiddlewareSettings) MiddlewareConfig {
	return MiddlewareConfig{
		Type:     "defaultSettings",
		Settings: &settings,
	}
}

// ExtractReasoningMiddleware creates a middleware config that extracts reasoning from XML tags.
func ExtractReasoningMiddleware(tagName, separator string) MiddlewareConfig {
	return MiddlewareConfig{
		Type:      "extractReasoning",
		TagName:   tagName,
		Separator: separator,
	}
}

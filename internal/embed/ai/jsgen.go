package aiembed

import (
	"encoding/json"
	"fmt"
	"strings"
)

// providerJSConfig maps provider names to their JS constructor and options.
type providerJSConfig struct {
	CreateFunc string // JS function name, e.g. "createOpenAI"
	OptsFunc   func(apiKey, baseURL string) map[string]interface{}
	EmbedFunc  string // method name for embedding, e.g. "embedding" or "textEmbeddingModel"
}

var providerConfigs = map[string]providerJSConfig{
	"openai": {
		CreateFunc: "createOpenAI",
		OptsFunc: func(apiKey, baseURL string) map[string]interface{} {
			opts := map[string]interface{}{"apiKey": apiKey, "compatibility": "strict"}
			if baseURL != "" {
				opts["baseURL"] = baseURL
			}
			return opts
		},
		EmbedFunc: "textEmbeddingModel",
	},
	"anthropic": {
		CreateFunc: "createAnthropic",
		OptsFunc: func(apiKey, baseURL string) map[string]interface{} {
			opts := map[string]interface{}{"apiKey": apiKey}
			if baseURL != "" {
				opts["baseURL"] = baseURL
			}
			return opts
		},
		EmbedFunc: "textEmbeddingModel",
	},
	"google": {
		CreateFunc: "createGoogleGenerativeAI",
		OptsFunc: func(apiKey, baseURL string) map[string]interface{} {
			opts := map[string]interface{}{"apiKey": apiKey}
			if baseURL != "" {
				opts["baseURL"] = baseURL
			}
			return opts
		},
		EmbedFunc: "textEmbeddingModel",
	},
}

// buildProviderJS generates JS code to create a provider and model instance.
// Returns a JS expression evaluating to a LanguageModel (e.g., `openai("gpt-4o")`).
func buildProviderJS(m Model, defaultProvider *ProviderConfig, envVars map[string]string) (string, error) {
	provider, modelID, apiKey, baseURL := resolveModel(m, defaultProvider, envVars)

	if apiKey == "" {
		return "", fmt.Errorf("no API key for provider %q (set via ProviderConfig or EnvVars)", provider)
	}

	cfg, ok := providerConfigs[provider]
	if !ok {
		return "", fmt.Errorf("unsupported provider %q (supported: openai, anthropic, google)", provider)
	}

	opts := cfg.OptsFunc(apiKey, baseURL)
	optsJSON, _ := json.Marshal(opts)
	return fmt.Sprintf("__ai_sdk.%s(%s)(%q)", cfg.CreateFunc, string(optsJSON), modelID), nil
}

// buildEmbeddingProviderJS generates JS code for an embedding model instance.
// Returns a JS expression evaluating to an EmbeddingModel.
func buildEmbeddingProviderJS(m Model, defaultProvider *ProviderConfig, envVars map[string]string) (string, error) {
	provider, modelID, apiKey, baseURL := resolveModel(m, defaultProvider, envVars)

	if apiKey == "" {
		return "", fmt.Errorf("no API key for provider %q (set via ProviderConfig or EnvVars)", provider)
	}

	cfg, ok := providerConfigs[provider]
	if !ok {
		return "", fmt.Errorf("unsupported embedding provider %q (supported: openai, anthropic, google)", provider)
	}

	opts := cfg.OptsFunc(apiKey, baseURL)
	optsJSON, _ := json.Marshal(opts)
	return fmt.Sprintf("__ai_sdk.%s(%s).%s(%q)", cfg.CreateFunc, string(optsJSON), cfg.EmbedFunc, modelID), nil
}

// buildCallSettingsJS generates JS object properties for call settings.
func buildCallSettingsJS(cs CallSettings) string {
	var parts []string
	if cs.MaxTokens > 0 {
		parts = append(parts, fmt.Sprintf("maxTokens: %d", cs.MaxTokens))
	}
	if cs.Temperature != nil {
		parts = append(parts, fmt.Sprintf("temperature: %v", *cs.Temperature))
	}
	if cs.TopP != nil {
		parts = append(parts, fmt.Sprintf("topP: %v", *cs.TopP))
	}
	if cs.TopK != nil {
		parts = append(parts, fmt.Sprintf("topK: %d", *cs.TopK))
	}
	if cs.PresencePenalty != nil {
		parts = append(parts, fmt.Sprintf("presencePenalty: %v", *cs.PresencePenalty))
	}
	if cs.FrequencyPenalty != nil {
		parts = append(parts, fmt.Sprintf("frequencyPenalty: %v", *cs.FrequencyPenalty))
	}
	if len(cs.StopSequences) > 0 {
		seqJSON, _ := json.Marshal(cs.StopSequences)
		parts = append(parts, fmt.Sprintf("stopSequences: %s", string(seqJSON)))
	}
	if cs.Seed != nil {
		parts = append(parts, fmt.Sprintf("seed: %d", *cs.Seed))
	}
	if cs.MaxRetries > 0 {
		parts = append(parts, fmt.Sprintf("maxRetries: %d", cs.MaxRetries))
	}
	return strings.Join(parts, ", ")
}

// buildMiddlewareJS generates JS code to wrap a model expression with middleware.
// Returns the wrapped model expression.
func buildMiddlewareJS(modelExpr string, mw []MiddlewareConfig) (string, error) {
	if len(mw) == 0 {
		return modelExpr, nil
	}

	expr := modelExpr
	for _, m := range mw {
		middlewareJS, err := middlewareToJS(m)
		if err != nil {
			return "", err
		}
		expr = fmt.Sprintf("__ai_sdk.wrapLanguageModel({model: %s, middleware: %s})", expr, middlewareJS)
	}
	return expr, nil
}

func middlewareToJS(m MiddlewareConfig) (string, error) {
	switch m.Type {
	case "defaultSettings":
		if m.Settings == nil {
			return "", fmt.Errorf("defaultSettings middleware requires Settings")
		}
		settingsJSON, err := json.Marshal(m.Settings)
		if err != nil {
			return "", fmt.Errorf("marshal middleware settings: %w", err)
		}
		return fmt.Sprintf("__ai_sdk.defaultSettingsMiddleware({settings: %s})", string(settingsJSON)), nil
	case "extractReasoning":
		opts := map[string]interface{}{}
		if m.TagName != "" {
			opts["tagName"] = m.TagName
		}
		if m.Separator != "" {
			opts["separator"] = m.Separator
		}
		optsJSON, _ := json.Marshal(opts)
		return fmt.Sprintf("__ai_sdk.extractReasoningMiddleware(%s)", string(optsJSON)), nil
	default:
		return "", fmt.Errorf("unsupported middleware type %q", m.Type)
	}
}

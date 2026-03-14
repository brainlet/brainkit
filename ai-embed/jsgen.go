package aiembed

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildProviderJS generates JS code to create a provider and model instance.
// Returns a JS expression evaluating to a LanguageModel (e.g., `openai("gpt-4o")`).
func buildProviderJS(m Model, defaultProvider *ProviderConfig, envVars map[string]string) (string, error) {
	provider, modelID, apiKey, baseURL := resolveModel(m, defaultProvider, envVars)

	if apiKey == "" {
		return "", fmt.Errorf("no API key for provider %q (set via ProviderConfig or EnvVars)", provider)
	}

	switch provider {
	case "openai":
		opts := map[string]string{"apiKey": apiKey}
		if baseURL != "" {
			opts["baseURL"] = baseURL
		}
		optsJSON, _ := json.Marshal(opts)
		return fmt.Sprintf("__ai_sdk.createOpenAI(%s)(%q)", string(optsJSON), modelID), nil
	default:
		return "", fmt.Errorf("unsupported provider %q (only 'openai' supported in v1)", provider)
	}
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

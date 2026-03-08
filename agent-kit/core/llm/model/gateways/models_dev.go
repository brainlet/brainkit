// Ported from: packages/core/src/llm/model/gateways/models-dev.ts
package gateways

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Models.dev API types
// ---------------------------------------------------------------------------

// ModelsDevProviderInfo describes a provider entry from the models.dev API.
type ModelsDevProviderInfo struct {
	ID     string                    `json:"id"`
	Name   string                    `json:"name"`
	Models map[string]ModelsDevModel `json:"models"`
	Env    []string                  `json:"env,omitempty"`
	API    string                    `json:"api,omitempty"`
	NPM    string                    `json:"npm,omitempty"`
	Doc    string                    `json:"doc,omitempty"`
}

// ModelsDevModel describes a single model entry from models.dev.
type ModelsDevModel struct {
	Status string `json:"status,omitempty"`
	// Other fields exist but are not needed for the port.
}

// ModelsDevResponse is the top-level response from models.dev/api.json.
type ModelsDevResponse = map[string]ModelsDevProviderInfo

// ---------------------------------------------------------------------------
// Provider overrides
// ---------------------------------------------------------------------------

// providerOverrides holds provider-specific overrides for URL, npm package, etc.
// These take priority over what models.dev returns.
var providerOverrides = map[string]ProviderConfig{
	"mistral": {
		URL: "https://api.mistral.ai/v1",
	},
	"groq": {
		URL: "https://api.groq.com/openai/v1",
	},
	"moonshotai": {
		URL: "https://api.moonshot.ai/anthropic/v1",
		NPM: "@ai-sdk/anthropic",
	},
	"moonshotai-cn": {
		URL: "https://api.moonshot.cn/anthropic/v1",
		NPM: "@ai-sdk/anthropic",
	},
}

// ---------------------------------------------------------------------------
// ModelsDevGateway
// ---------------------------------------------------------------------------

// ModelsDevGateway implements MastraModelGateway using the models.dev registry.
type ModelsDevGateway struct {
	providerConfigs map[string]ProviderConfig
}

// Compile-time check that ModelsDevGateway implements MastraModelGateway.
var _ MastraModelGateway = (*ModelsDevGateway)(nil)

// NewModelsDevGateway creates a new ModelsDevGateway.
// If providerConfigs is nil, they will be fetched from models.dev on FetchProviders.
func NewModelsDevGateway(providerConfigs map[string]ProviderConfig) *ModelsDevGateway {
	if providerConfigs == nil {
		providerConfigs = make(map[string]ProviderConfig)
	}
	return &ModelsDevGateway{
		providerConfigs: providerConfigs,
	}
}

// ID implements MastraModelGateway.
func (g *ModelsDevGateway) ID() string { return "models.dev" }

// Name implements MastraModelGateway.
func (g *ModelsDevGateway) Name() string { return "models.dev" }

// FetchProviders implements MastraModelGateway.
func (g *ModelsDevGateway) FetchProviders() (map[string]ProviderConfig, error) {
	resp, err := http.Get("https://models.dev/api.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from models.dev: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch from models.dev: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read models.dev response: %w", err)
	}

	var data ModelsDevResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse models.dev response: %w", err)
	}

	providerConfigs := make(map[string]ProviderConfig)

	for providerID, providerInfo := range data {
		// Skip excluded providers
		if isExcludedProvider(providerID) {
			continue
		}
		// Skip non-provider entries
		if len(providerInfo.Models) == 0 {
			continue
		}

		normalizedID := providerID

		// Check if this is OpenAI-compatible
		isOpenAICompat := providerInfo.NPM == "@ai-sdk/openai-compatible" ||
			providerInfo.NPM == "@ai-sdk/gateway"
		if _, hasOverride := providerOverrides[normalizedID]; hasOverride {
			isOpenAICompat = true
		}

		hasInstalledPackage := isProviderWithInstalledPackage(providerID)
		hasAPIAndEnv := providerInfo.API != "" && len(providerInfo.Env) > 0

		if !isOpenAICompat && !hasInstalledPackage && !hasAPIAndEnv {
			continue
		}

		// Get model IDs, filtering out deprecated models
		var modelIDs []string
		for modelID, modelInfo := range providerInfo.Models {
			if modelInfo.Status != "deprecated" {
				modelIDs = append(modelIDs, modelID)
			}
		}
		sort.Strings(modelIDs)

		// Get the API URL - overrides take priority
		apiURL := providerInfo.API
		if override, ok := providerOverrides[normalizedID]; ok && override.URL != "" {
			apiURL = override.URL
		}

		// Skip if we don't have a URL and no installed package
		if !hasInstalledPackage && apiURL == "" {
			continue
		}

		// Get the API key env var
		apiKeyEnvVar := fmt.Sprintf("%s_API_KEY", strings.ToUpper(strings.ReplaceAll(normalizedID, "-", "_")))
		if len(providerInfo.Env) > 0 {
			apiKeyEnvVar = providerInfo.Env[0]
		}

		// Determine the API key header
		var apiKeyHeader string
		if !hasInstalledPackage {
			apiKeyHeader = "Authorization"
			if override, ok := providerOverrides[normalizedID]; ok && override.APIKeyHeader != "" {
				apiKeyHeader = override.APIKeyHeader
			}
		}

		// Determine npm value
		npm := ""
		if override, ok := providerOverrides[normalizedID]; ok && override.NPM != "" {
			npm = override.NPM
		} else if providerInfo.NPM != "" &&
			providerInfo.NPM != "@ai-sdk/openai-compatible" &&
			providerInfo.NPM != "@ai-sdk/gateway" {
			npm = providerInfo.NPM
		}

		displayName := providerInfo.Name
		if displayName == "" {
			displayName = strings.ToUpper(providerID[:1]) + providerID[1:]
		}

		providerConfigs[normalizedID] = ProviderConfig{
			URL:          apiURL,
			APIKeyEnvVar: apiKeyEnvVar,
			APIKeyHeader: apiKeyHeader,
			Name:         displayName,
			Models:       modelIDs,
			DocURL:       providerInfo.Doc,
			Gateway:      "models.dev",
			NPM:          npm,
		}
	}

	// Store for later use
	g.providerConfigs = providerConfigs

	return providerConfigs, nil
}

// BuildURL implements MastraModelGateway.
func (g *ModelsDevGateway) BuildURL(routerID string, envVars map[string]string) (string, error) {
	parsed, err := parseModelRouterIDForModels(routerID)
	if err != nil {
		return "", nil // Return empty, not error, matching TS behavior
	}

	config, ok := g.providerConfigs[parsed.ProviderID]
	if !ok || config.URL == "" {
		return "", nil
	}

	// Check for custom base URL from env vars
	baseURLEnvVar := fmt.Sprintf("%s_BASE_URL", strings.ToUpper(strings.ReplaceAll(parsed.ProviderID, "-", "_")))
	if customURL, ok := envVars[baseURLEnvVar]; ok && customURL != "" {
		return customURL, nil
	}
	if customURL := os.Getenv(baseURLEnvVar); customURL != "" {
		return customURL, nil
	}

	return config.URL, nil
}

// GetAPIKey implements MastraModelGateway.
func (g *ModelsDevGateway) GetAPIKey(modelID string) (string, error) {
	parts := strings.SplitN(modelID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("could not identify provider from model id %s", modelID)
	}

	provider := parts[0]
	config, ok := g.providerConfigs[provider]
	if !ok {
		return "", fmt.Errorf("could not find config for provider %s with model id %s", provider, modelID)
	}

	envVarStr, ok := config.APIKeyEnvVar.(string)
	if !ok {
		return "", fmt.Errorf("could not find API key env var for model id %s", modelID)
	}

	apiKey := os.Getenv(envVarStr)
	if apiKey == "" {
		return "", fmt.Errorf("could not find API key %s for model id %s", envVarStr, modelID)
	}

	return apiKey, nil
}

// ResolveLanguageModel implements MastraModelGateway.
// TODO: integrate with actual AI SDK providers when available in Go.
func (g *ModelsDevGateway) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	// In TypeScript, this uses a switch on providerId to dispatch to the
	// correct AI SDK provider factory (createOpenAI, createAnthropic, etc.).
	// In Go, these AI SDK provider packages are not yet available.
	return nil, fmt.Errorf(
		"ModelsDevGateway.ResolveLanguageModel not yet implemented in Go; "+
			"provider=%s, model=%s",
		args.ProviderID, args.ModelID,
	)
}

// ---------------------------------------------------------------------------
// Local parser (doesn't import from parent model package to avoid cycles)
// ---------------------------------------------------------------------------

type parsedModelID struct {
	ProviderID string
	ModelID    string
}

// parseModelRouterIDForModels parses a model router ID into provider and model.
// This is a local copy to avoid circular imports with the parent model package.
func parseModelRouterIDForModels(routerID string) (*parsedModelID, error) {
	parts := strings.SplitN(routerID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid model router ID: %s", routerID)
	}
	return &parsedModelID{
		ProviderID: parts[0],
		ModelID:    parts[1],
	}, nil
}

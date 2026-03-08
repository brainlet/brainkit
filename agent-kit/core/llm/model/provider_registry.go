// Ported from: packages/core/src/llm/model/provider-registry.ts
package model

import (
	"fmt"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Provider / ModelForProvider / ProviderModels type aliases
// ---------------------------------------------------------------------------

// Provider is a string type for provider identifiers (e.g., "openai", "anthropic").
// In the TS source this is a generated union of all known provider strings.
type Provider = string

// ModelForProvider maps a provider to its list of model IDs.
// In TS this is a generated type; in Go we use a string alias.
type ModelForProvider = string

// ProviderModels is a map of provider IDs to their model lists.
type ProviderModels = map[string][]string

// ---------------------------------------------------------------------------
// RegistryData
// ---------------------------------------------------------------------------

// RegistryData holds the provider registry data loaded from JSON.
type RegistryData struct {
	Providers map[string]ProviderConfig `json:"providers"`
	Models    map[string][]string       `json:"models"`
	Version   string                    `json:"version"`
}

// ---------------------------------------------------------------------------
// GatewayRegistryOptions
// ---------------------------------------------------------------------------

// GatewayRegistryOptions holds options for the GatewayRegistry.
type GatewayRegistryOptions struct {
	// UseDynamicLoading enables dynamic loading from file system instead of
	// using static bundled registry.
	UseDynamicLoading bool
}

// ---------------------------------------------------------------------------
// GatewayRegistry
// ---------------------------------------------------------------------------

// GatewayRegistry manages dynamic loading and refreshing of provider data from gateways.
// It is a singleton class that handles runtime updates to the provider registry.
type GatewayRegistry struct {
	mu                sync.RWMutex
	useDynamicLoading bool
	customGateways    []MastraModelGateway
	registryData      *RegistryData
}

var (
	gatewayRegistryOnce     sync.Once
	gatewayRegistryInstance *GatewayRegistry
)

// GetGatewayRegistry returns the singleton GatewayRegistry instance.
func GetGatewayRegistry(opts ...GatewayRegistryOptions) *GatewayRegistry {
	gatewayRegistryOnce.Do(func() {
		useDynamic := false
		if len(opts) > 0 {
			useDynamic = opts[0].UseDynamicLoading
		}
		gatewayRegistryInstance = &GatewayRegistry{
			useDynamicLoading: useDynamic,
		}
	})
	return gatewayRegistryInstance
}

// RegisterCustomGateways registers custom gateways for type generation.
func (r *GatewayRegistry) RegisterCustomGateways(gateways []MastraModelGateway) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.customGateways = gateways
}

// GetCustomGateways returns all registered custom gateways.
func (r *GatewayRegistry) GetCustomGateways() []MastraModelGateway {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.customGateways
}

// GetProviderConfig returns the provider configuration for the given provider ID.
func (r *GatewayRegistry) GetProviderConfig(providerID string) (*ProviderConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.registryData == nil {
		return nil, false
	}
	cfg, ok := r.registryData.Providers[providerID]
	if !ok {
		return nil, false
	}
	return &cfg, true
}

// IsProviderRegistered checks if a provider is registered.
func (r *GatewayRegistry) IsProviderRegistered(providerID string) bool {
	_, ok := r.GetProviderConfig(providerID)
	return ok
}

// GetProviders returns all registered providers.
func (r *GatewayRegistry) GetProviders() map[string]ProviderConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.registryData == nil {
		return nil
	}
	return r.registryData.Providers
}

// GetModels returns all models grouped by provider.
func (r *GatewayRegistry) GetModels() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.registryData == nil {
		return nil
	}
	return r.registryData.Models
}

// SetRegistryData sets the registry data (used for initialization or testing).
func (r *GatewayRegistry) SetRegistryData(data *RegistryData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registryData = data
}

// ---------------------------------------------------------------------------
// ParseModelString
// ---------------------------------------------------------------------------

// ParseModelStringResult holds the result of ParseModelString.
type ParseModelStringResult struct {
	// Provider is the provider portion of the model string, or empty if not present.
	Provider string
	// ModelID is the model portion of the model string.
	ModelID string
}

// ParseModelString parses a model string to extract provider and model ID.
//
// Examples:
//
//	"openai/gpt-4o" -> { Provider: "openai", ModelID: "gpt-4o" }
//	"fireworks/accounts/etc/model" -> { Provider: "fireworks", ModelID: "accounts/etc/model" }
//	"gpt-4o" -> { Provider: "", ModelID: "gpt-4o" }
func ParseModelString(modelString string) ParseModelStringResult {
	firstSlash := strings.Index(modelString, "/")

	if firstSlash != -1 {
		provider := modelString[:firstSlash]
		modelID := modelString[firstSlash+1:]

		if provider != "" && modelID != "" {
			return ParseModelStringResult{
				Provider: provider,
				ModelID:  modelID,
			}
		}
	}

	return ParseModelStringResult{
		Provider: "",
		ModelID:  modelString,
	}
}

// GetProviderConfig is a package-level helper that retrieves a provider config
// from the singleton registry.
func GetProviderConfigByID(providerID string) (*ProviderConfig, bool) {
	return GetGatewayRegistry().GetProviderConfig(providerID)
}

// IsProviderRegistered is a package-level helper that checks if a provider
// is registered in the singleton registry.
func IsProviderRegisteredByID(providerID string) bool {
	return GetGatewayRegistry().IsProviderRegistered(providerID)
}

// GetRegisteredProviders returns all registered provider IDs from the singleton registry.
func GetRegisteredProviders() []string {
	providers := GetGatewayRegistry().GetProviders()
	if providers == nil {
		return nil
	}
	result := make([]string, 0, len(providers))
	for k := range providers {
		result = append(result, k)
	}
	return result
}

// IsValidModelID checks if a string is a valid OpenAI-compatible model ID
// (i.e., "provider/model" where provider is registered).
func IsValidModelID(modelID string) bool {
	parsed := ParseModelString(modelID)
	return parsed.Provider != "" && IsProviderRegisteredByID(parsed.Provider)
}

// ---------------------------------------------------------------------------
// FindGatewayForModel
// ---------------------------------------------------------------------------

// FindGatewayForModel finds the gateway that handles a specific model ID.
// Gateway ID is used as the prefix (e.g., "netlify" for netlify gateway).
// Exception: models.dev is a provider registry and doesn't use a prefix.
//
// Ported from: packages/core/src/llm/model/gateways/index.ts
func FindGatewayForModel(gatewayID string, gateways []MastraModelGateway) (MastraModelGateway, error) {
	// Check for gateways whose ID matches the prefix (true gateways like netlify, openrouter, vercel)
	for _, g := range gateways {
		if g.ID() != "models.dev" && (g.ID() == gatewayID || strings.HasPrefix(gatewayID, g.ID()+"/")) {
			return g, nil
		}
	}

	// Then check models.dev (provider registry without prefix)
	for _, g := range gateways {
		if g.ID() == "models.dev" {
			return g, nil
		}
	}

	return nil, fmt.Errorf("no Mastra model router gateway found for model id %s", gatewayID)
}

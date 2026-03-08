// Ported from: packages/core/src/llm/model/gateway-resolver.ts
package model

import (
	"fmt"
	"strings"
)

// ResolvedModelConfig holds the result of resolving a model router ID.
type ResolvedModelConfig struct {
	URL             string            `json:"url"`
	Headers         map[string]string `json:"headers"`
	ResolvedModelID string            `json:"resolvedModelId"`
	FullModelID     string            `json:"fullModelId"`
}

// ParsedModelRouterID holds the parsed provider and model IDs.
type ParsedModelRouterID struct {
	ProviderID string
	ModelID    string
}

// ParseModelRouterID parses a model router ID string into provider and model IDs.
// The optional gatewayPrefix strips the gateway prefix if present.
//
// Examples:
//
//	ParseModelRouterID("openai/gpt-4o", "") => { "openai", "gpt-4o" }
//	ParseModelRouterID("netlify/openai/gpt-4o", "netlify") => { "openai", "gpt-4o" }
//	ParseModelRouterID("azure-openai/my-deployment", "azure-openai") => { "azure-openai", "my-deployment" }
func ParseModelRouterID(routerID string, gatewayPrefix string) (ParsedModelRouterID, error) {
	if gatewayPrefix != "" && !strings.HasPrefix(routerID, gatewayPrefix+"/") {
		return ParsedModelRouterID{}, fmt.Errorf("expected %s/ in model router ID %s", gatewayPrefix, routerID)
	}

	parts := strings.Split(routerID, "/")

	// Azure OpenAI uses 2-part format (azure-openai/deployment), others use 3-part
	if gatewayPrefix == "azure-openai" {
		if len(parts) < 2 {
			return ParsedModelRouterID{}, fmt.Errorf("expected format azure-openai/deployment-name, but got %s", routerID)
		}
		return ParsedModelRouterID{
			ProviderID: "azure-openai",
			ModelID:    strings.Join(parts[1:], "/"),
		}, nil
	}

	// Standard 3-part format for other prefixed gateways (Netlify, etc.)
	if gatewayPrefix != "" && len(parts) < 3 {
		return ParsedModelRouterID{}, fmt.Errorf(
			"expected at least 3 id parts %s/provider/model, but only saw %d in %s",
			gatewayPrefix, len(parts), routerID,
		)
	}

	providerIdx := 0
	if gatewayPrefix != "" {
		providerIdx = 1
	}

	providerID := ""
	if providerIdx < len(parts) {
		providerID = parts[providerIdx]
	}

	modelSliceStart := providerIdx + 1
	modelID := ""
	if modelSliceStart < len(parts) {
		modelID = strings.Join(parts[modelSliceStart:], "/")
	}

	if !strings.Contains(routerID, "/") || providerID == "" || modelID == "" {
		return ParsedModelRouterID{}, fmt.Errorf(
			"attempted to parse provider/model from %s but this ID doesn't appear to contain a provider",
			routerID,
		)
	}

	return ParsedModelRouterID{
		ProviderID: providerID,
		ModelID:    modelID,
	}, nil
}

// Ported from: packages/core/src/llm/model/gateways/index.ts
package gateways

import (
	"fmt"
	"strings"
)

// FindGatewayForModel finds the gateway that handles a specific model ID
// based on gateway ID.
//
// Gateway ID is used as the prefix (e.g., "netlify" for netlify gateway).
// Exception: models.dev is a provider registry and doesn't use a prefix.
func FindGatewayForModel(gatewayID string, gateways []MastraModelGateway) (MastraModelGateway, error) {
	// First, check for gateways whose ID matches the prefix
	// (true gateways like netlify, openrouter, vercel).
	for _, g := range gateways {
		id := g.ID()
		if id != "models.dev" && (id == gatewayID || strings.HasPrefix(gatewayID, id+"/")) {
			return g, nil
		}
	}

	// Then check models.dev (provider registry without prefix).
	for _, g := range gateways {
		if g.ID() == "models.dev" {
			return g, nil
		}
	}

	return nil, fmt.Errorf(
		"no Mastra model router gateway found for model id %s",
		gatewayID,
	)
}

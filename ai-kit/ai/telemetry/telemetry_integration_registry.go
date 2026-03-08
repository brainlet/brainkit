// Ported from: packages/ai/src/telemetry/telemetry-integration-registry.ts
package telemetry

import "sync"

var (
	registryMu           sync.Mutex
	globalIntegrations   []TelemetryIntegration
	registryInitialized  bool
)

// RegisterTelemetryIntegration registers a telemetry integration globally.
func RegisterTelemetryIntegration(integration TelemetryIntegration) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if !registryInitialized {
		globalIntegrations = make([]TelemetryIntegration, 0)
		registryInitialized = true
	}
	globalIntegrations = append(globalIntegrations, integration)
}

// GetGlobalTelemetryIntegrations returns all globally registered telemetry integrations.
func GetGlobalTelemetryIntegrations() []TelemetryIntegration {
	registryMu.Lock()
	defer registryMu.Unlock()
	if globalIntegrations == nil {
		return []TelemetryIntegration{}
	}
	// Return a copy to prevent external mutation
	result := make([]TelemetryIntegration, len(globalIntegrations))
	copy(result, globalIntegrations)
	return result
}

// ResetGlobalTelemetryIntegrations resets the global registry. Used for testing.
func ResetGlobalTelemetryIntegrations() {
	registryMu.Lock()
	defer registryMu.Unlock()
	globalIntegrations = nil
	registryInitialized = false
}

// Ported from: packages/ai/src/util/log-v2-compatibility-warning.ts
package util

import "log"

// LogV2CompatibilityWarning logs a warning about using v2 specification compatibility mode.
func LogV2CompatibilityWarning(provider, modelID string) {
	log.Printf(
		"[%s/%s] WARNING: compatibility: specificationVersion: Using v2 specification compatibility mode. Some features may not be available.",
		provider, modelID,
	)
}

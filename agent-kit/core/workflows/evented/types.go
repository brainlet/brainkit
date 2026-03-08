// Ported from: packages/core/src/workflows/evented/types.ts
package evented

// PendingMarkerKey is a string key used to mark pending forEach iterations.
// Using a string key (not Symbol) ensures the marker survives JSON serialization
// which is critical for distributed execution where state is persisted to storage
// and loaded by different engine instances.
const PendingMarkerKey = "__mastra_pending__"

// PendingMarker is the type for the pending marker object used in forEach iteration tracking.
// In Go, this is represented as a map with a single key.
type PendingMarker map[string]bool

// CreatePendingMarker creates a new pending marker object.
// Used to mark forEach iterations that are about to be resumed.
func CreatePendingMarker() PendingMarker {
	return PendingMarker{PendingMarkerKey: true}
}

// IsPendingMarker checks if a value is a pending marker.
// Works correctly after JSON serialization/deserialization.
func IsPendingMarker(val any) bool {
	m, ok := val.(map[string]any)
	if !ok {
		// Also check the typed PendingMarker
		pm, ok2 := val.(PendingMarker)
		if !ok2 {
			return false
		}
		return pm[PendingMarkerKey] == true
	}
	v, exists := m[PendingMarkerKey]
	if !exists {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

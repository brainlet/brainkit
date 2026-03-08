// Ported from: packages/google/src/get-model-path.ts
package google

import "strings"

// GetModelPath returns the model path for a given model ID.
// If the model ID already contains a slash (e.g. "publishers/google/models/gemini-2.0-flash"),
// it is returned as-is. Otherwise, it is prefixed with "models/".
func GetModelPath(modelID string) string {
	if strings.Contains(modelID, "/") {
		return modelID
	}
	return "models/" + modelID
}

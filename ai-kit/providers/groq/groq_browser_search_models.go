// Ported from: packages/groq/src/groq-browser-search-models.ts
package groq

import "strings"

// BrowserSearchSupportedModels lists models that support browser search functionality.
// Based on: https://console.groq.com/docs/browser-search
var BrowserSearchSupportedModels = []GroqChatModelId{
	"openai/gpt-oss-20b",
	"openai/gpt-oss-120b",
}

// IsBrowserSearchSupportedModel checks if a model supports browser search functionality.
func IsBrowserSearchSupportedModel(modelId GroqChatModelId) bool {
	for _, m := range BrowserSearchSupportedModels {
		if m == modelId {
			return true
		}
	}
	return false
}

// GetSupportedModelsString returns a formatted list of supported models for error messages.
func GetSupportedModelsString() string {
	return strings.Join(BrowserSearchSupportedModels, ", ")
}

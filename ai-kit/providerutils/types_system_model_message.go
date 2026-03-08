// Ported from: packages/provider-utils/src/types/system-model-message.ts
package providerutils

// SystemModelMessage represents a system message.
// Using the "system" part of the prompt is strongly preferred to increase
// resilience against prompt injection attacks.
type SystemModelMessage struct {
	Role            string          `json:"role"` // "system"
	Content         string          `json:"content"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

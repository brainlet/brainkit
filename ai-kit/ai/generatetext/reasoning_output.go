// Ported from: packages/ai/src/generate-text/reasoning-output.ts
package generatetext

// ReasoningOutput represents reasoning output from text generation.
type ReasoningOutput struct {
	// Type is always "reasoning".
	Type string

	// Text is the reasoning text.
	Text string

	// ProviderMetadata contains additional provider-specific metadata.
	ProviderMetadata ProviderMetadata
}

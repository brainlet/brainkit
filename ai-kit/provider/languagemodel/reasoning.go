// Ported from: packages/provider/src/language-model/v3/language-model-v3-reasoning.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// Reasoning represents reasoning that the model has generated.
type Reasoning struct {
	// Text is the reasoning text content.
	Text string

	// ProviderMetadata is optional provider-specific metadata for the reasoning part.
	ProviderMetadata shared.ProviderMetadata
}

func (Reasoning) isContent() {}

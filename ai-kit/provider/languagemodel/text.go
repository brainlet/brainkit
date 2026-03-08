// Ported from: packages/provider/src/language-model/v3/language-model-v3-text.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// Text represents text that the model has generated.
type Text struct {
	// Text is the text content.
	Text string

	ProviderMetadata shared.ProviderMetadata
}

func (Text) isContent() {}

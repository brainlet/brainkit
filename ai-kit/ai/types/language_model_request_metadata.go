// Ported from: packages/ai/src/types/language-model-request-metadata.ts
package aitypes

// LanguageModelRequestMetadata contains metadata about a language model request.
type LanguageModelRequestMetadata struct {
	// Body is the request HTTP body that was sent to the provider API.
	Body any `json:"body,omitempty"`
}

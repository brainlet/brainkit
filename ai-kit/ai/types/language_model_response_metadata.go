// Ported from: packages/ai/src/types/language-model-response-metadata.ts
package aitypes

import "time"

// LanguageModelResponseMetadata contains metadata about a language model response.
type LanguageModelResponseMetadata struct {
	// ID is the identifier for the generated response.
	ID string `json:"id"`

	// Timestamp is the time when the response generation started.
	Timestamp time.Time `json:"timestamp"`

	// ModelID is the ID of the response model that was used to generate the response.
	ModelID string `json:"modelId"`

	// Headers contains response headers (available only for providers that use HTTP requests).
	Headers map[string]string `json:"headers,omitempty"`
}

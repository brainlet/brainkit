// Ported from: packages/ai/src/types/speech-model-response-metadata.ts
package aitypes

import "time"

// SpeechModelResponseMetadata contains metadata about a speech model response.
type SpeechModelResponseMetadata struct {
	// Timestamp is the time when the response generation started.
	Timestamp time.Time `json:"timestamp"`

	// ModelID is the ID of the response model that was used to generate the response.
	ModelID string `json:"modelId"`

	// Headers contains response headers.
	Headers map[string]string `json:"headers,omitempty"`

	// Body is the response body.
	Body any `json:"body,omitempty"`
}

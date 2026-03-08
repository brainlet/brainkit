// Ported from: packages/ai/src/types/transcription-model-response-metadata.ts
package aitypes

import "time"

// TranscriptionModelResponseMetadata contains metadata about a transcription model response.
type TranscriptionModelResponseMetadata struct {
	// Timestamp is the time when the response generation started.
	Timestamp time.Time `json:"timestamp"`

	// ModelID is the ID of the response model that was used to generate the response.
	ModelID string `json:"modelId"`

	// Headers contains response headers.
	Headers map[string]string `json:"headers,omitempty"`
}

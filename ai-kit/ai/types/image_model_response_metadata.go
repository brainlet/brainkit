// Ported from: packages/ai/src/types/image-model-response-metadata.ts
package aitypes

import "time"

// ImageModelResponseMetadata contains metadata about an image model response.
type ImageModelResponseMetadata struct {
	// Timestamp is the time when the response generation started.
	Timestamp time.Time `json:"timestamp"`

	// ModelID is the ID of the response model that was used to generate the response.
	ModelID string `json:"modelId"`

	// Headers contains response headers.
	Headers map[string]string `json:"headers,omitempty"`
}

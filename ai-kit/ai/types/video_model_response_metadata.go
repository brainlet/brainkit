// Ported from: packages/ai/src/types/video-model-response-metadata.ts
package aitypes

import "time"

// VideoModelResponseMetadata contains response metadata for a video model call.
type VideoModelResponseMetadata struct {
	// Timestamp is the time when the response generation started.
	Timestamp time.Time `json:"timestamp"`

	// ModelID is the ID of the response model that was used to generate the response.
	ModelID string `json:"modelId"`

	// Headers contains response headers.
	Headers map[string]string `json:"headers,omitempty"`

	// ProviderMetadata is provider-specific metadata for this call.
	// When multiple calls are made (n > maxVideosPerCall), each response
	// contains its own providerMetadata, allowing lossless per-call access.
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

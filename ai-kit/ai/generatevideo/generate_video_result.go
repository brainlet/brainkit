// Ported from: packages/ai/src/generate-video/generate-video-result.ts
package generatevideo

// GeneratedFile represents a generated file with data and media type.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type GeneratedFile struct {
	// Data is the raw binary data of the generated file.
	Data []byte
	// MediaType is the MIME type of the generated file (e.g., "video/mp4").
	MediaType string
}

// Warning from the model provider for this call.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type Warning struct {
	Type    string `json:"type"`
	Feature string `json:"feature,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

// VideoModelResponseMetadata holds response metadata from the video model provider.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type VideoModelResponseMetadata struct {
	Timestamp        any               `json:"timestamp,omitempty"`
	ModelID          string            `json:"modelId,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	ProviderMetadata map[string]any    `json:"providerMetadata,omitempty"`
}

// VideoModelProviderMetadata is additional provider-specific metadata for video models.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type VideoModelProviderMetadata = map[string]map[string]any

// GenerateVideoResult is the result of an experimental_generateVideo call.
// It contains the generated video and additional information.
type GenerateVideoResult struct {
	// Video is the first video that was generated.
	Video GeneratedFile

	// Videos are all videos that were generated.
	Videos []GeneratedFile

	// Warnings for the call, e.g. unsupported settings.
	Warnings []Warning

	// Responses are response metadata from the provider.
	Responses []VideoModelResponseMetadata

	// ProviderMetadata is provider-specific metadata passed through from the provider.
	ProviderMetadata VideoModelProviderMetadata
}

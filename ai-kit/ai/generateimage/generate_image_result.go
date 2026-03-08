// Ported from: packages/ai/src/generate-image/generate-image-result.ts
package generateimage

// GeneratedFile represents a generated file with data and media type.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type GeneratedFile struct {
	// Data is the raw binary data of the generated file.
	Data []byte
	// MediaType is the MIME type of the generated file (e.g., "image/png").
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

// ImageModelResponseMetadata holds response metadata from the image model provider.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type ImageModelResponseMetadata struct {
	Timestamp        any               `json:"timestamp,omitempty"`
	ModelID          string            `json:"modelId,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	ProviderMetadata map[string]any    `json:"providerMetadata,omitempty"`
}

// ImageModelUsage represents token usage for image generation operations.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type ImageModelUsage struct {
	InputTokens  *int `json:"inputTokens,omitempty"`
	OutputTokens *int `json:"outputTokens,omitempty"`
	TotalTokens  *int `json:"totalTokens,omitempty"`
}

// ImageModelProviderMetadata is additional provider-specific metadata for image models.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type ImageModelProviderMetadata = map[string]map[string]any

// GenerateImageResult is the result of a generateImage call.
// It contains the images and additional information.
type GenerateImageResult struct {
	// Image is the first image that was generated.
	Image GeneratedFile

	// Images are all the images that were generated.
	Images []GeneratedFile

	// Warnings for the call, e.g. unsupported settings.
	Warnings []Warning

	// Responses are response metadata from the provider.
	// There may be multiple responses if multiple calls were made.
	Responses []ImageModelResponseMetadata

	// ProviderMetadata is provider-specific metadata passed through from the provider.
	ProviderMetadata ImageModelProviderMetadata

	// Usage is the combined token usage across all underlying provider calls.
	Usage ImageModelUsage
}

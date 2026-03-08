// Ported from: packages/provider/src/image-model/v3/image-model-v3.ts
package imagemodel

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// ProviderMetadata is additional provider-specific metadata for image generation.
// The outer record is keyed by the provider name, and the inner
// record is provider-specific metadata. It always includes an
// "images" key with image-specific metadata.
type ProviderMetadata = map[string]ImageProviderMetadataEntry

// ImageProviderMetadataEntry contains provider-specific metadata including images.
type ImageProviderMetadataEntry struct {
	Images jsonvalue.JSONArray
	Extra  map[string]jsonvalue.JSONValue
}

// ImageData represents generated image data that can be a string (base64) or bytes.
type ImageData interface {
	imageData()
}

// ImageDataStrings represents images as base64 encoded strings.
type ImageDataStrings struct {
	Values []string
}

func (ImageDataStrings) imageData() {}

// ImageDataBytes represents images as binary data.
type ImageDataBytes struct {
	Values [][]byte
}

func (ImageDataBytes) imageData() {}

// GenerateResult is the result of an image model doGenerate call.
type GenerateResult struct {
	// Images are generated images as base64 encoded strings or binary data.
	Images ImageData

	// Warnings for the call, e.g. unsupported features.
	Warnings []shared.Warning

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata ProviderMetadata

	// Response contains response information for telemetry and debugging.
	Response GenerateResultResponse

	// Usage is optional token usage for the image generation call.
	Usage *Usage
}

// GenerateResultResponse contains response information.
type GenerateResultResponse struct {
	// Timestamp is the start timestamp of the generated response.
	Timestamp time.Time

	// ModelID is the ID of the response model.
	ModelID string

	// Headers are the response headers.
	Headers map[string]string
}

// MaxImagesPerCallFunc is a function type matching the TS GetMaxImagesPerCallFunction.
type MaxImagesPerCallFunc func(modelID string) (*int, error)

// ImageModel is the specification for an image generation model (version 3).
type ImageModel interface {
	// SpecificationVersion returns the image model interface version.
	// Must return "v3".
	SpecificationVersion() string

	// Provider returns the name of the provider for logging purposes.
	Provider() string

	// ModelID returns the provider-specific model ID for logging purposes.
	ModelID() string

	// MaxImagesPerCall returns the limit of how many images can be generated
	// in a single API call. Returns nil for no limit / use global default.
	MaxImagesPerCall() (*int, error)

	// DoGenerate generates an array of images.
	DoGenerate(options CallOptions) (GenerateResult, error)
}

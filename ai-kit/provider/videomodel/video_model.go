// Ported from: packages/provider/src/video-model/v3/video-model-v3.ts
package videomodel

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// VideoData is a sealed interface representing generated video data.
// Implementations: VideoDataURL, VideoDataBase64, VideoDataBinary.
type VideoData interface {
	videoDataType() string
}

// VideoDataURL represents a video available as a URL.
type VideoDataURL struct {
	URL       string
	MediaType string
}

func (VideoDataURL) videoDataType() string { return "url" }

// VideoDataBase64 represents a video as a base64-encoded string.
type VideoDataBase64 struct {
	Data      string
	MediaType string
}

func (VideoDataBase64) videoDataType() string { return "base64" }

// VideoDataBinary represents a video as binary data.
type VideoDataBinary struct {
	Data      []byte
	MediaType string
}

func (VideoDataBinary) videoDataType() string { return "binary" }

// GenerateResult is the result of a video model doGenerate call.
type GenerateResult struct {
	// Videos are the generated videos.
	Videos []VideoData

	// Warnings for the call, e.g. unsupported features.
	Warnings []shared.Warning

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata shared.ProviderMetadata

	// Response contains response information for telemetry and debugging.
	Response GenerateResultResponse
}

// GenerateResultResponse contains response information.
type GenerateResultResponse struct {
	// Timestamp for the start of the generated response.
	Timestamp time.Time

	// ModelID is the response model ID.
	ModelID string

	// Headers are the response headers.
	Headers map[string]string
}

// MaxVideosPerCallFunc is a function type matching the TS GetMaxVideosPerCallFunction.
type MaxVideosPerCallFunc func(modelID string) (*int, error)

// VideoModel is the specification for a video generation model (version 3).
type VideoModel interface {
	// SpecificationVersion returns the video model interface version.
	// Must return "v3".
	SpecificationVersion() string

	// Provider returns the name of the provider for logging purposes.
	Provider() string

	// ModelID returns the provider-specific model ID for logging purposes.
	ModelID() string

	// MaxVideosPerCall returns the limit of how many videos can be generated
	// in a single API call. Returns nil for no limit / use global default.
	MaxVideosPerCall() (*int, error)

	// DoGenerate generates an array of videos.
	DoGenerate(options CallOptions) (GenerateResult, error)
}

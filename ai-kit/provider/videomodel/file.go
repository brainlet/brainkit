// Ported from: packages/provider/src/video-model/v3/video-model-v3-file.ts
package videomodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// File represents a video or image file for video editing or image-to-video generation.
// This is a sealed interface; implementations: VideoFileData, VideoFileURL.
type File interface {
	videoModelFileType() string
}

// VideoFileData represents a file with inline data.
type VideoFileData struct {
	// MediaType is the IANA media type of the file.
	// Video types: "video/mp4", "video/webm", "video/quicktime"
	// Image types: "image/png", "image/jpeg", "image/webp"
	MediaType string

	// Data is file data as base64 encoded string or binary data.
	Data VideoFileDataContent

	// ProviderOptions is optional provider-specific metadata for the file part.
	ProviderOptions shared.ProviderMetadata
}

func (VideoFileData) videoModelFileType() string { return "file" }

// VideoFileURL represents a file referenced by URL.
type VideoFileURL struct {
	// URL is the URL of the video or image file.
	URL string

	// ProviderOptions is optional provider-specific metadata for the file part.
	ProviderOptions shared.ProviderMetadata
}

func (VideoFileURL) videoModelFileType() string { return "url" }

// VideoFileDataContent is file data that can be a string or bytes.
type VideoFileDataContent interface {
	videoFileData()
}

// VideoFileDataString represents base64 encoded file data.
type VideoFileDataString struct {
	Value string
}

func (VideoFileDataString) videoFileData() {}

// VideoFileDataBytes represents binary file data.
type VideoFileDataBytes struct {
	Data []byte
}

func (VideoFileDataBytes) videoFileData() {}

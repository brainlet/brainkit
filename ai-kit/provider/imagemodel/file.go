// Ported from: packages/provider/src/image-model/v3/image-model-v3-file.ts
package imagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// File represents an image file that can be used for image editing or variation generation.
// This is a sealed interface; implementations: FileData, FileURL.
type File interface {
	imageModelFileType() string
}

// FileData represents an image file with inline data.
type FileData struct {
	// MediaType is the IANA media type of the file, e.g. "image/png".
	MediaType string

	// Data is generated file data as base64 encoded string or binary data.
	Data ImageFileData

	// ProviderOptions is optional provider-specific metadata for the file part.
	ProviderOptions shared.ProviderMetadata
}

func (FileData) imageModelFileType() string { return "file" }

// FileURL represents an image file referenced by URL.
type FileURL struct {
	// URL is the URL of the image file.
	URL string

	// ProviderOptions is optional provider-specific metadata for the file part.
	ProviderOptions shared.ProviderMetadata
}

func (FileURL) imageModelFileType() string { return "url" }

// ImageFileData is file data that can be a string or bytes.
type ImageFileData interface {
	imageFileData()
}

// ImageFileDataString represents base64 encoded file data.
type ImageFileDataString struct {
	Value string
}

func (ImageFileDataString) imageFileData() {}

// ImageFileDataBytes represents binary file data.
type ImageFileDataBytes struct {
	Data []byte
}

func (ImageFileDataBytes) imageFileData() {}

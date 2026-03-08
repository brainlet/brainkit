// Ported from: packages/provider-utils/src/convert-image-model-file-to-data-uri.ts
package providerutils

import "fmt"

// ImageModelFileURL represents an image file referenced by URL.
type ImageModelFileURL struct {
	Type string // "url"
	URL  string
}

// ImageModelFileData represents an image file with inline data.
type ImageModelFileData struct {
	Type      string // "data"
	MediaType string
	// Data is either a base64 string or raw bytes.
	Data interface{}
}

// ImageModelFile is a union type for image model files.
// It can be either URL-based or data-based.
type ImageModelFile struct {
	Type      string
	URL       string
	MediaType string
	Data      interface{}
}

// ConvertImageModelFileToDataUri converts an ImageModelFile to a URL or data URI string.
// If the file is a URL, it returns the URL as-is.
// If the file is base64 data, it returns a data URI with the base64 data.
// If the file is a []byte, it converts it to base64 and returns a data URI.
func ConvertImageModelFileToDataUri(file ImageModelFile) string {
	if file.Type == "url" {
		return file.URL
	}

	switch data := file.Data.(type) {
	case string:
		return fmt.Sprintf("data:%s;base64,%s", file.MediaType, data)
	case []byte:
		return fmt.Sprintf("data:%s;base64,%s", file.MediaType, ConvertBytesToBase64(data))
	default:
		return fmt.Sprintf("data:%s;base64,%v", file.MediaType, data)
	}
}

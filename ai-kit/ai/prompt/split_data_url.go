// Ported from: packages/ai/src/prompt/split-data-url.ts
package prompt

import "strings"

// SplitDataURLResult holds the parsed components of a data URL.
type SplitDataURLResult struct {
	// MediaType is the MIME type extracted from the data URL.
	MediaType *string
	// Base64Content is the base64-encoded content from the data URL.
	Base64Content *string
}

// SplitDataURL parses a data URL into its media type and base64 content.
func SplitDataURL(dataURL string) SplitDataURLResult {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return SplitDataURLResult{}
	}

	header := parts[0]
	base64Content := parts[1]

	// Extract media type from header like "data:image/png;base64"
	headerParts := strings.SplitN(header, ";", 2)
	if len(headerParts) == 0 {
		return SplitDataURLResult{}
	}

	colonParts := strings.SplitN(headerParts[0], ":", 2)
	if len(colonParts) != 2 {
		return SplitDataURLResult{}
	}

	mediaType := colonParts[1]
	return SplitDataURLResult{
		MediaType:     &mediaType,
		Base64Content: &base64Content,
	}
}

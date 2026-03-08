// Ported from: packages/core/src/stream/aisdk/v5/compat/content.ts
package compat

import (
	"fmt"
	"net/url"
	"strings"
)

// ---------------------------------------------------------------------------
// splitDataUrl
// ---------------------------------------------------------------------------

// DataURLParts holds the parsed components of a data URL.
type DataURLParts struct {
	MediaType     string
	Base64Content string
}

// splitDataUrl parses a data URL into its media type and base64 content.
// Returns empty strings if parsing fails.
func splitDataUrl(dataURL string) DataURLParts {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return DataURLParts{}
	}

	header := parts[0]
	base64Content := parts[1]

	// Parse media type from header like "data:image/png;base64"
	headerParts := strings.SplitN(header, ";", 2)
	if len(headerParts) == 0 {
		return DataURLParts{}
	}

	colonParts := strings.SplitN(headerParts[0], ":", 2)
	if len(colonParts) != 2 {
		return DataURLParts{}
	}

	return DataURLParts{
		MediaType:     colonParts[1],
		Base64Content: base64Content,
	}
}

// ---------------------------------------------------------------------------
// DataContentResult
// ---------------------------------------------------------------------------

// DataContentResult holds the converted data content and optional media type.
type DataContentResult struct {
	// Data is the content as either a base64 string or raw bytes.
	Data any // string or []byte
	// MediaType is the IANA media type, if detected from a data URL.
	MediaType string
}

// ---------------------------------------------------------------------------
// ConvertToDataContent
// ---------------------------------------------------------------------------

// ConvertToDataContent converts various content types to a uniform data representation.
//
// It handles:
//   - []byte: passed through directly
//   - string: checked if it's a URL (data: URLs are parsed for media type and base64)
//   - Other: returned as-is with no media type
//
// This mirrors the TS convertToDataContent function which handles
// Uint8Array, ArrayBuffer, URL, data URLs, and plain strings.
func ConvertToDataContent(content any) (DataContentResult, error) {
	// Handle byte slices (equivalent to Uint8Array/Buffer in TS)
	if b, ok := content.([]byte); ok {
		return DataContentResult{Data: b}, nil
	}

	// Handle string content
	if s, ok := content.(string); ok {
		// Try to parse as URL
		parsed, err := url.Parse(s)
		if err == nil && parsed.Scheme == "data" {
			// It's a data URL - extract media type and base64 content
			parts := splitDataUrl(s)
			if parts.MediaType == "" || parts.Base64Content == "" {
				return DataContentResult{}, fmt.Errorf("invalid data URL format in content %s", s)
			}
			return DataContentResult{
				Data:      parts.Base64Content,
				MediaType: parts.MediaType,
			}, nil
		}

		// Not a data URL, return as-is
		return DataContentResult{Data: content}, nil
	}

	// Other types: return as-is
	return DataContentResult{Data: content}, nil
}

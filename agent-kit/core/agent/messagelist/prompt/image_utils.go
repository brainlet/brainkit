// Ported from: packages/core/src/agent/message-list/prompt/image-utils.ts
package prompt

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// DataUriParts represents the parsed components of a data URI.
type DataUriParts struct {
	MimeType      string
	Base64Content string
	IsDataUri     bool
}

// ParseDataUri parses a data URI string into its components.
// Format: data:[<mediatype>][;base64],<data>
func ParseDataUri(dataUri string) DataUriParts {
	if !strings.HasPrefix(dataUri, "data:") {
		return DataUriParts{
			IsDataUri:     false,
			Base64Content: dataUri,
		}
	}

	base64Index := strings.Index(dataUri, ",")
	if base64Index == -1 {
		// Malformed data URI, return as-is
		return DataUriParts{
			IsDataUri:     true,
			Base64Content: dataUri,
		}
	}

	header := dataUri[5:base64Index] // Skip "data:" prefix
	base64Content := dataUri[base64Index+1:]

	// Extract MIME type from header (before ";base64" or ";")
	semicolonIndex := strings.Index(header, ";")
	mimeType := ""
	if semicolonIndex != -1 {
		mimeType = header[:semicolonIndex]
	} else {
		mimeType = header
	}

	return DataUriParts{
		IsDataUri:     true,
		MimeType:      mimeType,
		Base64Content: base64Content,
	}
}

// CreateDataUri creates a data URI from base64 content and MIME type.
func CreateDataUri(base64Content string, mimeType string) string {
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	// If it's already a data URI, return as-is
	if strings.HasPrefix(base64Content, "data:") {
		return base64Content
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Content)
}

// ImageContentToString converts various image data formats to a string representation.
func ImageContentToString(image any, fallbackMimeType ...string) string {
	mimeType := ""
	if len(fallbackMimeType) > 0 {
		mimeType = fallbackMimeType[0]
	}

	switch v := image.(type) {
	case string:
		return v
	case *url.URL:
		return v.String()
	case []byte:
		b64 := ConvertDataContentToBase64String(v)
		if mimeType != "" && !strings.HasPrefix(b64, "data:") {
			return fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
		}
		return b64
	default:
		return fmt.Sprintf("%v", image)
	}
}

// ImageContentToDataUri converts various image data formats to a data URI string.
func ImageContentToDataUri(image any, mimeType string) string {
	if mimeType == "" {
		mimeType = "image/png"
	}
	imageStr := ImageContentToString(image, mimeType)

	if strings.HasPrefix(imageStr, "data:") {
		return imageStr
	}
	if strings.HasPrefix(imageStr, "http://") || strings.HasPrefix(imageStr, "https://") {
		return imageStr
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, imageStr)
}

// GetImageCacheKey gets a stable cache key component for image content.
func GetImageCacheKey(image any) any {
	switch v := image.(type) {
	case *url.URL:
		return v.String()
	case string:
		return len(v)
	case []byte:
		return len(v)
	default:
		return image
	}
}

// IsValidUrl checks if a string is a valid URL (including protocol-relative URLs).
func IsValidUrl(str string) bool {
	_, err := url.ParseRequestURI(str)
	if err == nil && (strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://") ||
		strings.HasPrefix(str, "data:") || strings.HasPrefix(str, "gs:") ||
		strings.HasPrefix(str, "s3:")) {
		return true
	}
	// Try as protocol-relative URL
	if strings.HasPrefix(str, "//") {
		_, err := url.ParseRequestURI("https:" + str)
		return err == nil
	}
	return false
}

// CategorizedFileData represents the result of categorizing file data.
type CategorizedFileData struct {
	Type     string // "url" | "dataUri" | "raw"
	MimeType string
	Data     string
}

// CategorizeFileData categorizes a string as a URL, data URI, or raw data.
func CategorizeFileData(data string, fallbackMimeType string) CategorizedFileData {
	parsed := ParseDataUri(data)
	mimeType := fallbackMimeType
	if parsed.IsDataUri && parsed.MimeType != "" {
		mimeType = parsed.MimeType
	}

	if parsed.IsDataUri {
		return CategorizedFileData{
			Type:     "dataUri",
			MimeType: mimeType,
			Data:     data,
		}
	}

	if IsValidUrl(data) {
		return CategorizedFileData{
			Type:     "url",
			MimeType: mimeType,
			Data:     data,
		}
	}

	return CategorizedFileData{
		Type:     "raw",
		MimeType: mimeType,
		Data:     data,
	}
}

var base64Pattern = regexp.MustCompile(`^[A-Za-z0-9+/\-_]+=*$`)

// ClassifiedFileData represents the result of classifying file data.
type ClassifiedFileData struct {
	Type     string // "url" | "dataUri" | "base64" | "other"
	MimeType string
}

// ClassifyFileData classifies a string as a URL, data URI, base64, or other.
func ClassifyFileData(data string) ClassifiedFileData {
	parsed := ParseDataUri(data)
	if parsed.IsDataUri {
		return ClassifiedFileData{
			Type:     "dataUri",
			MimeType: parsed.MimeType,
		}
	}

	if IsValidUrl(data) {
		return ClassifiedFileData{Type: "url"}
	}

	if base64Pattern.MatchString(data) && len(data) > 20 {
		return ClassifiedFileData{Type: "base64"}
	}

	return ClassifiedFileData{Type: "other"}
}

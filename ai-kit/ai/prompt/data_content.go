// Ported from: packages/ai/src/prompt/data-content.ts
package prompt

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// DataContent represents data that can be a base64 string or raw bytes.
// In TypeScript this is: string | Uint8Array | ArrayBuffer | Buffer.
// In Go, we use either string (base64) or []byte.
type DataContent interface{}

// LanguageModelV4DataContent represents the data content format for the language model.
// It can be a string (base64 or URL), []byte, or *url.URL.
type LanguageModelV4DataContent interface{}

// ConvertToLanguageModelV4DataContentResult holds the conversion result.
type ConvertToLanguageModelV4DataContentResult struct {
	Data      LanguageModelV4DataContent
	MediaType *string
}

// ConvertToLanguageModelV4DataContent converts DataContent or *url.URL to LanguageModelV4DataContent.
func ConvertToLanguageModelV4DataContent(content interface{}) ConvertToLanguageModelV4DataContentResult {
	switch c := content.(type) {
	case []byte:
		return ConvertToLanguageModelV4DataContentResult{Data: c, MediaType: nil}
	case *url.URL:
		return handleURL(c)
	case string:
		// Attempt to parse as URL
		if u, err := url.Parse(c); err == nil && u.Scheme != "" {
			return handleURL(u)
		}
		// Assume it's base64 or raw string data
		return ConvertToLanguageModelV4DataContentResult{Data: c, MediaType: nil}
	default:
		return ConvertToLanguageModelV4DataContentResult{Data: content, MediaType: nil}
	}
}

func handleURL(u *url.URL) ConvertToLanguageModelV4DataContentResult {
	// Extract data from data URL
	if u.Scheme == "data" {
		result := SplitDataURL(u.String())
		if result.MediaType == nil || result.Base64Content == nil {
			// Invalid data URL format
			return ConvertToLanguageModelV4DataContentResult{
				Data:      u,
				MediaType: nil,
			}
		}
		return ConvertToLanguageModelV4DataContentResult{
			Data:      *result.Base64Content,
			MediaType: result.MediaType,
		}
	}

	return ConvertToLanguageModelV4DataContentResult{Data: u, MediaType: nil}
}

// ConvertDataContentToBase64String converts data content to a base64-encoded string.
func ConvertDataContentToBase64String(content DataContent) string {
	switch c := content.(type) {
	case string:
		return c
	case []byte:
		return base64.StdEncoding.EncodeToString(c)
	default:
		return fmt.Sprintf("%v", c)
	}
}

// ConvertDataContentToUint8Array converts data content to a byte slice.
func ConvertDataContentToUint8Array(content DataContent) ([]byte, error) {
	switch c := content.(type) {
	case []byte:
		return c, nil
	case string:
		data, err := base64.StdEncoding.DecodeString(c)
		if err != nil {
			return nil, NewInvalidDataContentError(
				content,
				err,
				"Invalid data content. Content string is not a base64-encoded media.",
			)
		}
		return data, nil
	default:
		return nil, NewInvalidDataContentError(content, nil, "")
	}
}

// ConvertUint8ArrayToText converts a byte slice to a string of text.
func ConvertUint8ArrayToText(data []byte) (string, error) {
	// In Go, []byte is already a valid UTF-8 string (or raw bytes).
	// This is equivalent to new TextDecoder().decode(uint8Array).
	return string(data), nil
}

// IsDataURL checks whether the given string is a data URL.
func IsDataURL(s string) bool {
	return strings.HasPrefix(s, "data:")
}

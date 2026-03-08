// Ported from: packages/provider-utils/src/uint8-utils.ts
package providerutils

import (
	"encoding/base64"
	"strings"
)

// ConvertBase64ToBytes decodes a base64 string (standard or URL-safe) into a byte slice.
func ConvertBase64ToBytes(base64String string) ([]byte, error) {
	// Replace URL-safe characters with standard base64 characters
	b64 := strings.NewReplacer("-", "+", "_", "/").Replace(base64String)
	return base64.StdEncoding.DecodeString(b64)
}

// ConvertBytesToBase64 encodes a byte slice into a standard base64 string.
func ConvertBytesToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// ConvertToBase64 converts a string or byte slice to base64.
// If the input is already a string (assumed base64), it is returned as-is.
func ConvertToBase64String(value string) string {
	return value
}

// ConvertToBase64Bytes converts bytes to base64.
func ConvertToBase64Bytes(value []byte) string {
	return ConvertBytesToBase64(value)
}

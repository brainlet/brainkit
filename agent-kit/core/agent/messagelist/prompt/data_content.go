// Ported from: packages/core/src/agent/message-list/prompt/data-content.ts
package prompt

import (
	"encoding/base64"
)

// DataContent can be a base64-encoded string or raw bytes.
// In Go we represent this as either string or []byte.

// ConvertDataContentToBase64String converts data content to a base64-encoded string.
// If the content is already a string, it's returned as-is (assumed to be base64).
// If it's []byte, it's base64-encoded.
func ConvertDataContentToBase64String(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	default:
		return ""
	}
}

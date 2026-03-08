// Ported from: packages/provider/src/errors/get-error-message.ts
package errors

import "encoding/json"

// GetErrorMessage extracts a human-readable message from an unknown value.
func GetErrorMessage(err any) string {
	if err == nil {
		return "unknown error"
	}
	switch v := err.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		b, jsonErr := json.Marshal(v)
		if jsonErr != nil {
			return "unknown error"
		}
		return string(b)
	}
}

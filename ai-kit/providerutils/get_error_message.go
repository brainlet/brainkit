// Ported from: packages/provider-utils/src/get-error-message.ts
package providerutils

import (
	"encoding/json"
	"fmt"
)

// GetErrorMessage extracts a human-readable message from an unknown error value.
func GetErrorMessage(err interface{}) string {
	if err == nil {
		return "unknown error"
	}

	if s, ok := err.(string); ok {
		return s
	}

	if e, ok := err.(error); ok {
		return e.Error()
	}

	b, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		return fmt.Sprintf("%v", err)
	}
	return string(b)
}

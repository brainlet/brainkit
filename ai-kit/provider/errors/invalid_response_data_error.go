// Ported from: packages/provider/src/errors/invalid-response-data-error.ts
package errors

import (
	"encoding/json"
	"fmt"
)

// InvalidResponseDataError indicates the server returned a response with
// invalid data content. This should be thrown by providers when they cannot
// parse the response from the API.
type InvalidResponseDataError struct {
	AISDKError

	// Data is the invalid response data.
	Data any
}

// NewInvalidResponseDataError creates a new InvalidResponseDataError.
// If message is empty, a default message is generated from the data.
func NewInvalidResponseDataError(data any, message string) *InvalidResponseDataError {
	if message == "" {
		b, _ := json.Marshal(data)
		message = fmt.Sprintf("Invalid response data: %s.", string(b))
	}
	return &InvalidResponseDataError{
		AISDKError: AISDKError{
			Name:    "AI_InvalidResponseDataError",
			Message: message,
		},
		Data: data,
	}
}

// IsInvalidResponseDataError checks if an error is an InvalidResponseDataError.
func IsInvalidResponseDataError(err error) bool {
	var target *InvalidResponseDataError
	return As(err, &target)
}

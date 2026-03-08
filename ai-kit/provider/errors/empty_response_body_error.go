// Ported from: packages/provider/src/errors/empty-response-body-error.ts
package errors

// EmptyResponseBodyError indicates an empty response body was received.
type EmptyResponseBodyError struct {
	AISDKError
}

// NewEmptyResponseBodyError creates a new EmptyResponseBodyError.
func NewEmptyResponseBodyError(message string) *EmptyResponseBodyError {
	if message == "" {
		message = "Empty response body"
	}
	return &EmptyResponseBodyError{
		AISDKError: AISDKError{
			Name:    "AI_EmptyResponseBodyError",
			Message: message,
		},
	}
}

// IsEmptyResponseBodyError checks if an error is an EmptyResponseBodyError.
func IsEmptyResponseBodyError(err error) bool {
	var target *EmptyResponseBodyError
	return As(err, &target)
}

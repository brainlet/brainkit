// Ported from: packages/provider/src/errors/no-content-generated-error.ts
package errors

// NoContentGeneratedError is thrown when the AI provider fails to generate any content.
type NoContentGeneratedError struct {
	AISDKError
}

// NewNoContentGeneratedError creates a new NoContentGeneratedError.
func NewNoContentGeneratedError(message string) *NoContentGeneratedError {
	if message == "" {
		message = "No content generated."
	}
	return &NoContentGeneratedError{
		AISDKError: AISDKError{
			Name:    "AI_NoContentGeneratedError",
			Message: message,
		},
	}
}

// IsNoContentGeneratedError checks if an error is a NoContentGeneratedError.
func IsNoContentGeneratedError(err error) bool {
	var target *NoContentGeneratedError
	return As(err, &target)
}

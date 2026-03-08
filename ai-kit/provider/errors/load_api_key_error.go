// Ported from: packages/provider/src/errors/load-api-key-error.ts
package errors

// LoadAPIKeyError indicates a failure to load an API key.
type LoadAPIKeyError struct {
	AISDKError
}

// NewLoadAPIKeyError creates a new LoadAPIKeyError.
func NewLoadAPIKeyError(message string) *LoadAPIKeyError {
	return &LoadAPIKeyError{
		AISDKError: AISDKError{
			Name:    "AI_LoadAPIKeyError",
			Message: message,
		},
	}
}

// IsLoadAPIKeyError checks if an error is a LoadAPIKeyError.
func IsLoadAPIKeyError(err error) bool {
	var target *LoadAPIKeyError
	return As(err, &target)
}

// Ported from: packages/ai/src/error/invalid-stream-part-error.ts
package aierror

const invalidStreamPartErrorName = "AI_InvalidStreamPartError"
const invalidStreamPartErrorMarker = "vercel.ai.error." + invalidStreamPartErrorName

// InvalidStreamPartError is returned when an invalid stream part is encountered.
type InvalidStreamPartError struct {
	AISDKError

	// Chunk is the stream part that caused the error.
	Chunk SingleRequestTextStreamPart
}

// NewInvalidStreamPartError creates a new InvalidStreamPartError.
func NewInvalidStreamPartError(chunk SingleRequestTextStreamPart, message string) *InvalidStreamPartError {
	return &InvalidStreamPartError{
		AISDKError: AISDKError{
			Name:    invalidStreamPartErrorName,
			Message: message,
		},
		Chunk: chunk,
	}
}

// IsInvalidStreamPartError checks whether the given error is an InvalidStreamPartError.
func IsInvalidStreamPartError(err error) bool {
	_, ok := err.(*InvalidStreamPartError)
	return ok
}

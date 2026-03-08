// Ported from: packages/ai/src/prompt/invalid-data-content-error.ts
package prompt

import "fmt"

const invalidDataContentErrorName = "AI_InvalidDataContentError"
const invalidDataContentErrorMarker = "vercel.ai.error." + invalidDataContentErrorName

// InvalidDataContentError is returned when data content is in an invalid format.
type InvalidDataContentError struct {
	// Name is the error classification.
	Name string
	// Message is the human-readable error message.
	Message string
	// Content is the invalid content that was provided.
	Content interface{}
	// Cause is the underlying cause of the error, if any.
	Cause error
}

func (e *InvalidDataContentError) Error() string {
	return e.Message
}

func (e *InvalidDataContentError) Unwrap() error {
	return e.Cause
}

// NewInvalidDataContentError creates a new InvalidDataContentError.
func NewInvalidDataContentError(content interface{}, cause error, message string) *InvalidDataContentError {
	if message == "" {
		message = fmt.Sprintf(
			"Invalid data content. Expected a base64 string, []byte, but got %T.",
			content,
		)
	}
	return &InvalidDataContentError{
		Name:    invalidDataContentErrorName,
		Message: message,
		Content: content,
		Cause:   cause,
	}
}

// IsInvalidDataContentError checks whether the given error is an InvalidDataContentError.
func IsInvalidDataContentError(err error) bool {
	_, ok := err.(*InvalidDataContentError)
	return ok
}

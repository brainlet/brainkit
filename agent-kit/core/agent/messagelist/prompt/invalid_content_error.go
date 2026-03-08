// Ported from: packages/core/src/agent/message-list/prompt/invalid-content-error.ts
package prompt

import (
	"fmt"
)

// InvalidDataContentError represents an error when data content is in an unexpected format.
// TODO: In TS this extends AISDKError from @internal/ai-sdk-v4.
type InvalidDataContentError struct {
	Content any
	Message string
	Cause   error
}

// NewInvalidDataContentError creates a new InvalidDataContentError.
func NewInvalidDataContentError(content any, cause error, message string) *InvalidDataContentError {
	if message == "" {
		message = fmt.Sprintf("Invalid data content. Expected a base64 string, []byte, but got %T.", content)
	}
	return &InvalidDataContentError{
		Content: content,
		Message: message,
		Cause:   cause,
	}
}

func (e *InvalidDataContentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *InvalidDataContentError) Unwrap() error {
	return e.Cause
}

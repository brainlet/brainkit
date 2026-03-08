// Ported from: packages/provider/src/errors/invalid-prompt-error.ts
package errors

import "fmt"

// InvalidPromptError indicates a prompt is invalid. This error should be
// thrown by providers when they cannot process a prompt.
type InvalidPromptError struct {
	AISDKError

	// Prompt is the invalid prompt value.
	Prompt any
}

// NewInvalidPromptError creates a new InvalidPromptError.
func NewInvalidPromptError(prompt any, message string, cause error) *InvalidPromptError {
	return &InvalidPromptError{
		AISDKError: AISDKError{
			Name:    "AI_InvalidPromptError",
			Message: fmt.Sprintf("Invalid prompt: %s", message),
			Cause:   cause,
		},
		Prompt: prompt,
	}
}

// Error implements the error interface.
func (e *InvalidPromptError) Error() string {
	msg := fmt.Sprintf("%s: %s", e.Name, e.Message)
	if e.Cause != nil {
		msg += fmt.Sprintf(" (cause: %v)", e.Cause)
	}
	return msg
}

// Unwrap returns the underlying cause.
func (e *InvalidPromptError) Unwrap() error {
	return e.Cause
}

// IsInvalidPromptError checks if an error is an InvalidPromptError.
func IsInvalidPromptError(err error) bool {
	var target *InvalidPromptError
	return As(err, &target)
}

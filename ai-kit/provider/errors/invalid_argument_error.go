// Ported from: packages/provider/src/errors/invalid-argument-error.ts
package errors

import "fmt"

// InvalidArgumentError indicates a function argument is invalid.
type InvalidArgumentError struct {
	AISDKError

	// Argument is the name of the invalid argument.
	Argument string
}

// NewInvalidArgumentError creates a new InvalidArgumentError.
func NewInvalidArgumentError(argument, message string, cause error) *InvalidArgumentError {
	return &InvalidArgumentError{
		AISDKError: AISDKError{
			Name:    "AI_InvalidArgumentError",
			Message: message,
			Cause:   cause,
		},
		Argument: argument,
	}
}

// Error implements the error interface.
func (e *InvalidArgumentError) Error() string {
	msg := fmt.Sprintf("%s: %s (argument: %s)", e.Name, e.Message, e.Argument)
	if e.Cause != nil {
		msg += fmt.Sprintf(" (cause: %v)", e.Cause)
	}
	return msg
}

// Unwrap returns the underlying cause.
func (e *InvalidArgumentError) Unwrap() error {
	return e.Cause
}

// IsInvalidArgumentError checks if an error is an InvalidArgumentError.
func IsInvalidArgumentError(err error) bool {
	var target *InvalidArgumentError
	return As(err, &target)
}

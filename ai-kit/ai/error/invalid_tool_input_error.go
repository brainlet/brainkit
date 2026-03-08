// Ported from: packages/ai/src/error/invalid-tool-input-error.ts
package aierror

import "fmt"

const invalidToolInputErrorName = "AI_InvalidToolInputError"
const invalidToolInputErrorMarker = "vercel.ai.error." + invalidToolInputErrorName

// InvalidToolInputError is returned when a tool receives invalid input.
type InvalidToolInputError struct {
	AISDKError

	// ToolName is the name of the tool that received invalid input.
	ToolName string

	// ToolInput is the raw input string that was invalid.
	ToolInput string
}

// InvalidToolInputErrorOptions are the options for creating an InvalidToolInputError.
type InvalidToolInputErrorOptions struct {
	// Message overrides the default error message. Optional.
	Message string
	// ToolInput is the raw input string that was invalid.
	ToolInput string
	// ToolName is the name of the tool that received invalid input.
	ToolName string
	// Cause is the underlying error that caused the validation failure.
	Cause error
}

// NewInvalidToolInputError creates a new InvalidToolInputError.
func NewInvalidToolInputError(opts InvalidToolInputErrorOptions) *InvalidToolInputError {
	message := opts.Message
	if message == "" {
		causeMsg := GetErrorMessage(opts.Cause)
		message = fmt.Sprintf("Invalid input for tool %s: %s", opts.ToolName, causeMsg)
	}

	return &InvalidToolInputError{
		AISDKError: AISDKError{
			Name:    invalidToolInputErrorName,
			Message: message,
			Cause:   opts.Cause,
		},
		ToolName:  opts.ToolName,
		ToolInput: opts.ToolInput,
	}
}

// IsInvalidToolInputError checks whether the given error is an InvalidToolInputError.
func IsInvalidToolInputError(err error) bool {
	_, ok := err.(*InvalidToolInputError)
	return ok
}

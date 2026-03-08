// Ported from: packages/ai/src/error/invalid-argument-error.ts
package aierror

import "fmt"

const invalidArgumentErrorName = "AI_InvalidArgumentError"
const invalidArgumentErrorMarker = "vercel.ai.error." + invalidArgumentErrorName

// InvalidArgumentError is returned when an invalid argument is provided to a function.
type InvalidArgumentError struct {
	AISDKError

	// Parameter is the name of the parameter that received the invalid argument.
	Parameter string

	// Value is the invalid value that was provided.
	Value interface{}
}

// NewInvalidArgumentError creates a new InvalidArgumentError.
func NewInvalidArgumentError(parameter string, value interface{}, message string) *InvalidArgumentError {
	return &InvalidArgumentError{
		AISDKError: AISDKError{
			Name:    invalidArgumentErrorName,
			Message: fmt.Sprintf("Invalid argument for parameter %s: %s", parameter, message),
		},
		Parameter: parameter,
		Value:     value,
	}
}

// IsInvalidArgumentError checks whether the given error is an InvalidArgumentError.
func IsInvalidArgumentError(err error) bool {
	_, ok := err.(*InvalidArgumentError)
	return ok
}

// Ported from: packages/provider/src/errors/unsupported-functionality-error.ts
package errors

import "fmt"

// UnsupportedFunctionalityError indicates a requested functionality is not supported.
type UnsupportedFunctionalityError struct {
	AISDKError

	// Functionality is the name of the unsupported functionality.
	Functionality string
}

// NewUnsupportedFunctionalityError creates a new UnsupportedFunctionalityError.
func NewUnsupportedFunctionalityError(functionality, message string) *UnsupportedFunctionalityError {
	if message == "" {
		message = fmt.Sprintf("'%s' functionality not supported.", functionality)
	}
	return &UnsupportedFunctionalityError{
		AISDKError: AISDKError{
			Name:    "AI_UnsupportedFunctionalityError",
			Message: message,
		},
		Functionality: functionality,
	}
}

// IsUnsupportedFunctionalityError checks if an error is an UnsupportedFunctionalityError.
func IsUnsupportedFunctionalityError(err error) bool {
	var target *UnsupportedFunctionalityError
	return As(err, &target)
}

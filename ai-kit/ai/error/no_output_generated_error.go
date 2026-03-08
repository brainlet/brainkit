// Ported from: packages/ai/src/error/no-output-generated-error.ts
package aierror

const noOutputGeneratedErrorName = "AI_NoOutputGeneratedError"
const noOutputGeneratedErrorMarker = "vercel.ai.error." + noOutputGeneratedErrorName

// NoOutputGeneratedError is returned when no LLM output was generated, e.g. because of errors.
type NoOutputGeneratedError struct {
	AISDKError
}

// NoOutputGeneratedErrorOptions are the options for creating a NoOutputGeneratedError.
type NoOutputGeneratedErrorOptions struct {
	// Message overrides the default error message. Optional.
	Message string
	// Cause is the underlying error. Optional.
	Cause error
}

// NewNoOutputGeneratedError creates a new NoOutputGeneratedError.
func NewNoOutputGeneratedError(opts *NoOutputGeneratedErrorOptions) *NoOutputGeneratedError {
	message := "No output generated."
	var cause error

	if opts != nil {
		if opts.Message != "" {
			message = opts.Message
		}
		cause = opts.Cause
	}

	return &NoOutputGeneratedError{
		AISDKError: AISDKError{
			Name:    noOutputGeneratedErrorName,
			Message: message,
			Cause:   cause,
		},
	}
}

// IsNoOutputGeneratedError checks whether the given error is a NoOutputGeneratedError.
func IsNoOutputGeneratedError(err error) bool {
	_, ok := err.(*NoOutputGeneratedError)
	return ok
}

// Ported from: packages/ai/src/error/no-image-generated-error.ts
package aierror

const noImageGeneratedErrorName = "AI_NoImageGeneratedError"
const noImageGeneratedErrorMarker = "vercel.ai.error." + noImageGeneratedErrorName

// NoImageGeneratedError is returned when no image could be generated.
// This can have multiple causes:
//   - The model failed to generate a response.
//   - The model generated a response that could not be parsed.
type NoImageGeneratedError struct {
	AISDKError

	// Responses contains the response metadata for each call.
	Responses []ImageModelResponseMetadata
}

// NoImageGeneratedErrorOptions are the options for creating a NoImageGeneratedError.
type NoImageGeneratedErrorOptions struct {
	// Message overrides the default error message. Optional.
	Message string
	// Cause is the underlying error. Optional.
	Cause error
	// Responses contains the response metadata for each call. Optional.
	Responses []ImageModelResponseMetadata
}

// NewNoImageGeneratedError creates a new NoImageGeneratedError.
func NewNoImageGeneratedError(opts NoImageGeneratedErrorOptions) *NoImageGeneratedError {
	message := opts.Message
	if message == "" {
		message = "No image generated."
	}

	return &NoImageGeneratedError{
		AISDKError: AISDKError{
			Name:    noImageGeneratedErrorName,
			Message: message,
			Cause:   opts.Cause,
		},
		Responses: opts.Responses,
	}
}

// IsNoImageGeneratedError checks whether the given error is a NoImageGeneratedError.
func IsNoImageGeneratedError(err error) bool {
	_, ok := err.(*NoImageGeneratedError)
	return ok
}

// Ported from: packages/ai/src/error/no-video-generated-error.ts
package aierror

const noVideoGeneratedErrorName = "AI_NoVideoGeneratedError"
const noVideoGeneratedErrorMarker = "vercel.ai.error." + noVideoGeneratedErrorName

// NoVideoGeneratedError is returned when no video was generated.
type NoVideoGeneratedError struct {
	AISDKError

	// Responses contains the response metadata for each call.
	Responses []VideoModelResponseMetadata
}

// NoVideoGeneratedErrorOptions are the options for creating a NoVideoGeneratedError.
type NoVideoGeneratedErrorOptions struct {
	// Message overrides the default error message. Optional.
	Message string
	// Cause is the underlying error. Optional.
	Cause error
	// Responses contains the response metadata for each call.
	Responses []VideoModelResponseMetadata
}

// NewNoVideoGeneratedError creates a new NoVideoGeneratedError.
func NewNoVideoGeneratedError(opts NoVideoGeneratedErrorOptions) *NoVideoGeneratedError {
	message := opts.Message
	if message == "" {
		message = "No video generated."
	}

	return &NoVideoGeneratedError{
		AISDKError: AISDKError{
			Name:    noVideoGeneratedErrorName,
			Message: message,
			Cause:   opts.Cause,
		},
		Responses: opts.Responses,
	}
}

// IsNoVideoGeneratedError checks whether the given error is a NoVideoGeneratedError.
func IsNoVideoGeneratedError(err error) bool {
	_, ok := err.(*NoVideoGeneratedError)
	return ok
}

// Deprecated: Use IsNoVideoGeneratedError instead.
func IsNoVideoGeneratedErrorLegacy(err error) bool {
	e, ok := err.(*NoVideoGeneratedError)
	if !ok {
		return false
	}
	return e.Responses != nil
}

// NoVideoGeneratedErrorJSON is the JSON representation of a NoVideoGeneratedError.
// Deprecated: Do not use this type. It will be removed in the next major version.
type NoVideoGeneratedErrorJSON struct {
	Name      string                     `json:"name"`
	Message   string                     `json:"message"`
	Stack     string                     `json:"stack,omitempty"`
	Cause     error                      `json:"cause,omitempty"`
	Responses []VideoModelResponseMetadata `json:"responses"`
}

// Deprecated: Do not use this method. It will be removed in the next major version.
func (e *NoVideoGeneratedError) ToJSON() NoVideoGeneratedErrorJSON {
	return NoVideoGeneratedErrorJSON{
		Name:      e.Name,
		Message:   e.Message,
		Cause:     e.Cause,
		Responses: e.Responses,
	}
}

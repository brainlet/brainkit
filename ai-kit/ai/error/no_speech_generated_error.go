// Ported from: packages/ai/src/error/no-speech-generated-error.ts
package aierror

const noSpeechGeneratedErrorName = "AI_NoSpeechGeneratedError"
const noSpeechGeneratedErrorMarker = "vercel.ai.error." + noSpeechGeneratedErrorName

// NoSpeechGeneratedError is returned when no speech audio was generated.
type NoSpeechGeneratedError struct {
	AISDKError

	// Responses contains the response metadata for each call.
	Responses []SpeechModelResponseMetadata
}

// NewNoSpeechGeneratedError creates a new NoSpeechGeneratedError.
func NewNoSpeechGeneratedError(responses []SpeechModelResponseMetadata) *NoSpeechGeneratedError {
	return &NoSpeechGeneratedError{
		AISDKError: AISDKError{
			Name:    noSpeechGeneratedErrorName,
			Message: "No speech audio generated.",
		},
		Responses: responses,
	}
}

// IsNoSpeechGeneratedError checks whether the given error is a NoSpeechGeneratedError.
func IsNoSpeechGeneratedError(err error) bool {
	_, ok := err.(*NoSpeechGeneratedError)
	return ok
}

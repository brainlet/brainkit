// Ported from: packages/ai/src/error/no-transcript-generated-error.ts
package aierror

const noTranscriptGeneratedErrorName = "AI_NoTranscriptGeneratedError"
const noTranscriptGeneratedErrorMarker = "vercel.ai.error." + noTranscriptGeneratedErrorName

// NoTranscriptGeneratedError is returned when no transcript was generated.
type NoTranscriptGeneratedError struct {
	AISDKError

	// Responses contains the response metadata for each call.
	Responses []TranscriptionModelResponseMetadata
}

// NewNoTranscriptGeneratedError creates a new NoTranscriptGeneratedError.
func NewNoTranscriptGeneratedError(responses []TranscriptionModelResponseMetadata) *NoTranscriptGeneratedError {
	return &NoTranscriptGeneratedError{
		AISDKError: AISDKError{
			Name:    noTranscriptGeneratedErrorName,
			Message: "No transcript generated.",
		},
		Responses: responses,
	}
}

// IsNoTranscriptGeneratedError checks whether the given error is a NoTranscriptGeneratedError.
func IsNoTranscriptGeneratedError(err error) bool {
	_, ok := err.(*NoTranscriptGeneratedError)
	return ok
}

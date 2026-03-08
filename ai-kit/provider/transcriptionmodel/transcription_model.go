// Ported from: packages/provider/src/transcription-model/v3/transcription-model-v3.ts
package transcriptionmodel

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// Segment represents a portion of transcribed text with timing information.
type Segment struct {
	// Text is the text content of this segment.
	Text string

	// StartSecond is the start time of this segment in seconds.
	StartSecond float64

	// EndSecond is the end time of this segment in seconds.
	EndSecond float64
}

// GenerateResult is the result of a transcription model doGenerate call.
type GenerateResult struct {
	// Text is the complete transcribed text from the audio.
	Text string

	// Segments is an array of transcript segments with timing information.
	Segments []Segment

	// Language is the detected language of the audio content (ISO-639-1 code).
	// May be nil if the language couldn't be detected.
	Language *string

	// DurationInSeconds is the total duration of the audio file in seconds.
	// May be nil if the duration couldn't be determined.
	DurationInSeconds *float64

	// Warnings for the call, e.g. unsupported settings.
	Warnings []shared.Warning

	// Request contains optional request information for telemetry and debugging.
	Request *GenerateResultRequest

	// Response contains response information for telemetry and debugging.
	Response GenerateResultResponse

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata map[string]jsonvalue.JSONObject
}

// GenerateResultRequest contains request information.
type GenerateResultRequest struct {
	// Body is the raw request HTTP body (JSON stringified).
	Body *string
}

// GenerateResultResponse contains response information.
type GenerateResultResponse struct {
	// Timestamp for the start of the generated response.
	Timestamp time.Time

	// ModelID is the response model ID.
	ModelID string

	// Headers are the response headers.
	Headers shared.Headers

	// Body is the response body.
	Body any
}

// TranscriptionModel is the specification for a transcription model (version 3).
type TranscriptionModel interface {
	// SpecificationVersion returns the transcription model interface version.
	// Must return "v3".
	SpecificationVersion() string

	// Provider returns the name of the provider for logging purposes.
	Provider() string

	// ModelID returns the provider-specific model ID for logging purposes.
	ModelID() string

	// DoGenerate generates a transcript.
	DoGenerate(options CallOptions) (GenerateResult, error)
}

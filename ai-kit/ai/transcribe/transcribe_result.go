// Ported from: packages/ai/src/transcribe/transcribe-result.ts
package transcribe

// Warning from the model provider for this call.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type Warning struct {
	Type    string `json:"type"`
	Feature string `json:"feature,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

// TranscriptionModelResponseMetadata holds response metadata from the transcription model provider.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type TranscriptionModelResponseMetadata struct {
	Timestamp any               `json:"timestamp,omitempty"`
	ModelID   string            `json:"modelId,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// TranscriptionSegment represents a portion of the transcribed text with timing information.
type TranscriptionSegment struct {
	// Text is the text content of this segment.
	Text string `json:"text"`
	// StartSecond is the start time of this segment in seconds.
	StartSecond float64 `json:"startSecond"`
	// EndSecond is the end time of this segment in seconds.
	EndSecond float64 `json:"endSecond"`
}

// TranscriptionResult is the result of a transcribe call.
// It contains the transcript and additional information.
type TranscriptionResult struct {
	// Text is the complete transcribed text from the audio.
	Text string

	// Segments are the transcript segments with timing information.
	Segments []TranscriptionSegment

	// Language is the detected language of the audio content (ISO-639-1 code).
	// May be empty if the language couldn't be detected.
	Language string

	// DurationInSeconds is the total duration of the audio file in seconds.
	// May be nil if the duration couldn't be determined.
	DurationInSeconds *float64

	// Warnings for the call, e.g. unsupported settings.
	Warnings []Warning

	// Responses are response metadata from the provider.
	Responses []TranscriptionModelResponseMetadata

	// ProviderMetadata is provider metadata from the provider.
	ProviderMetadata map[string]map[string]any
}

// Ported from: packages/ai/src/generate-speech/generate-speech-result.ts
package generatespeech

// Warning from the model provider for this call.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type Warning struct {
	Type    string `json:"type"`
	Feature string `json:"feature,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

// SpeechModelResponseMetadata holds response metadata from the speech model provider.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type SpeechModelResponseMetadata struct {
	Timestamp any               `json:"timestamp,omitempty"`
	ModelID   string            `json:"modelId,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// SpeechResult is the result of a generateSpeech call.
// It contains the audio data and additional information.
type SpeechResult struct {
	// Audio is the generated audio file with the audio data.
	Audio GeneratedAudioFile

	// Warnings for the call, e.g. unsupported settings.
	Warnings []Warning

	// Responses are response metadata from the provider.
	Responses []SpeechModelResponseMetadata

	// ProviderMetadata is provider metadata from the provider.
	ProviderMetadata map[string]map[string]any
}

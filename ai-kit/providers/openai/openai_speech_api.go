// Ported from: packages/openai/src/speech/openai-speech-api.ts
package openai

// OpenAISpeechAPITypes contains the provider-specific types for the
// OpenAI speech API request body.
type OpenAISpeechAPITypes struct {
	// Voice to use when generating the audio.
	// Supported voices are alloy, ash, ballad, coral, echo, fable, onyx, nova, sage, shimmer, and verse.
	// Default: "alloy"
	Voice *string `json:"voice,omitempty"`

	// Speed of the generated audio.
	// Select a value from 0.25 to 4.0.
	// Default: 1.0
	Speed *float64 `json:"speed,omitempty"`

	// ResponseFormat is the format of the generated audio.
	// Default: "mp3"
	ResponseFormat *string `json:"response_format,omitempty"`

	// Instructions for the speech generation e.g. "Speak in a slow and steady tone".
	// Does not work with tts-1 or tts-1-hd.
	Instructions *string `json:"instructions,omitempty"`
}

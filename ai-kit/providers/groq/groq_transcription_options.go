// Ported from: packages/groq/src/groq-transcription-options.ts
package groq

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GroqTranscriptionModelId is a type alias for Groq transcription model identifiers.
// Known models: "whisper-large-v3-turbo", "whisper-large-v3".
// Any string is accepted.
type GroqTranscriptionModelId = string

// GroqTranscriptionModelOptions contains Groq-specific transcription model options.
type GroqTranscriptionModelOptions struct {
	// Language of the input audio in ISO-639-1 format.
	Language *string `json:"language,omitempty"`

	// Prompt to guide the model's style or specify how to spell unfamiliar words.
	Prompt *string `json:"prompt,omitempty"`

	// ResponseFormat defines the output response format.
	ResponseFormat *string `json:"responseFormat,omitempty"`

	// Temperature between 0 and 1.
	Temperature *float64 `json:"temperature,omitempty"`

	// TimestampGranularities for transcription (word and/or segment).
	TimestampGranularities []string `json:"timestampGranularities,omitempty"`
}

// GroqTranscriptionModelOptionsSchema is the schema for validating GroqTranscriptionModelOptions.
var GroqTranscriptionModelOptionsSchema = &providerutils.Schema[GroqTranscriptionModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[GroqTranscriptionModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[GroqTranscriptionModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts GroqTranscriptionModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[GroqTranscriptionModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[GroqTranscriptionModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

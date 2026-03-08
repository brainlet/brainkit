// Ported from: packages/openai/src/transcription/openai-transcription-options.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAITranscriptionModelID is the identifier for an OpenAI transcription model.
type OpenAITranscriptionModelID = string

// OpenAITranscriptionModelOptions contains provider-specific options for
// OpenAI transcription models.
// https://platform.openai.com/docs/api-reference/audio/createTranscription
type OpenAITranscriptionModelOptions struct {
	// Include specifies additional information to include in the transcription response.
	Include []string `json:"include,omitempty"`

	// Language is the language of the input audio in ISO-639-1 format.
	Language *string `json:"language,omitempty"`

	// Prompt is an optional text to guide the model's style or continue
	// a previous audio segment.
	Prompt *string `json:"prompt,omitempty"`

	// Temperature is the sampling temperature, between 0 and 1. Default: 0.
	Temperature *float64 `json:"temperature,omitempty"`

	// TimestampGranularities specifies the timestamp granularities to populate
	// for this transcription. Default: ["segment"].
	TimestampGranularities []string `json:"timestampGranularities,omitempty"`
}

// openAITranscriptionModelOptionsSchema is the providerutils.Schema used to validate
// and parse OpenAITranscriptionModelOptions from provider options maps.
var openAITranscriptionModelOptionsSchema = &providerutils.Schema[OpenAITranscriptionModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[OpenAITranscriptionModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[OpenAITranscriptionModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts OpenAITranscriptionModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[OpenAITranscriptionModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[OpenAITranscriptionModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

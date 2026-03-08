// Ported from: packages/openai/src/speech/openai-speech-options.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAISpeechModelID is the identifier for an OpenAI speech model.
type OpenAISpeechModelID = string

// OpenAISpeechModelOptions contains provider-specific options for
// OpenAI speech models.
// https://platform.openai.com/docs/api-reference/audio/createSpeech
type OpenAISpeechModelOptions struct {
	// Instructions for the speech generation.
	Instructions *string `json:"instructions,omitempty"`

	// Speed of the generated audio. Range: 0.25 to 4.0. Default: 1.0.
	Speed *float64 `json:"speed,omitempty"`
}

// openaiSpeechModelOptionsSchema is the providerutils.Schema used to validate
// and parse OpenAISpeechModelOptions from provider options maps.
var openaiSpeechModelOptionsSchema = &providerutils.Schema[OpenAISpeechModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[OpenAISpeechModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[OpenAISpeechModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts OpenAISpeechModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[OpenAISpeechModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[OpenAISpeechModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

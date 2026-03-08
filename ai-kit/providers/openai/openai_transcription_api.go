// Ported from: packages/openai/src/transcription/openai-transcription-api.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// openaiTranscriptionResponse is the response structure for the OpenAI
// transcription API.
type openaiTranscriptionResponse struct {
	Text     string                            `json:"text"`
	Language *string                           `json:"language,omitempty"`
	Duration *float64                          `json:"duration,omitempty"`
	Words    []openaiTranscriptionWord         `json:"words,omitempty"`
	Segments []openaiTranscriptionSegment      `json:"segments,omitempty"`
}

type openaiTranscriptionWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type openaiTranscriptionSegment struct {
	ID               int       `json:"id"`
	Seek             int       `json:"seek"`
	Start            float64   `json:"start"`
	End              float64   `json:"end"`
	Text             string    `json:"text"`
	Tokens           []int     `json:"tokens"`
	Temperature      float64   `json:"temperature"`
	AvgLogprob       float64   `json:"avg_logprob"`
	CompressionRatio float64   `json:"compression_ratio"`
	NoSpeechProb     float64   `json:"no_speech_prob"`
}

var openaiTranscriptionResponseSchema = &providerutils.Schema[openaiTranscriptionResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[openaiTranscriptionResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[openaiTranscriptionResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp openaiTranscriptionResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[openaiTranscriptionResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[openaiTranscriptionResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

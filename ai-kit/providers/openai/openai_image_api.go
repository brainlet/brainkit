// Ported from: packages/openai/src/image/openai-image-api.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// openaiImageResponse is the response structure for the OpenAI image API.
type openaiImageResponse struct {
	Created      *int                          `json:"created,omitempty"`
	Data         []openaiImageResponseData     `json:"data"`
	Background   *string                       `json:"background,omitempty"`
	OutputFormat *string                       `json:"output_format,omitempty"`
	Size         *string                       `json:"size,omitempty"`
	Quality      *string                       `json:"quality,omitempty"`
	Usage        *openaiImageResponseUsage     `json:"usage,omitempty"`
}

type openaiImageResponseData struct {
	B64JSON       string  `json:"b64_json"`
	RevisedPrompt *string `json:"revised_prompt,omitempty"`
}

type openaiImageResponseUsage struct {
	InputTokens        *int                                `json:"input_tokens,omitempty"`
	OutputTokens       *int                                `json:"output_tokens,omitempty"`
	TotalTokens        *int                                `json:"total_tokens,omitempty"`
	InputTokensDetails *openaiImageResponseTokenDetails    `json:"input_tokens_details,omitempty"`
}

type openaiImageResponseTokenDetails struct {
	ImageTokens *int `json:"image_tokens,omitempty"`
	TextTokens  *int `json:"text_tokens,omitempty"`
}

var openaiImageResponseSchema = &providerutils.Schema[openaiImageResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[openaiImageResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[openaiImageResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp openaiImageResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[openaiImageResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[openaiImageResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

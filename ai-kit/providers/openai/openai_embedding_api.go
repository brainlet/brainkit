// Ported from: packages/openai/src/embedding/openai-embedding-api.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// openaiTextEmbeddingResponse is the response structure for the OpenAI
// text embedding API.
type openaiTextEmbeddingResponse struct {
	Data  []openaiTextEmbeddingDataItem `json:"data"`
	Usage *openaiTextEmbeddingUsage     `json:"usage,omitempty"`
}

type openaiTextEmbeddingDataItem struct {
	Embedding []float64 `json:"embedding"`
}

type openaiTextEmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
}

var openaiTextEmbeddingResponseSchema = &providerutils.Schema[openaiTextEmbeddingResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[openaiTextEmbeddingResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[openaiTextEmbeddingResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp openaiTextEmbeddingResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[openaiTextEmbeddingResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[openaiTextEmbeddingResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

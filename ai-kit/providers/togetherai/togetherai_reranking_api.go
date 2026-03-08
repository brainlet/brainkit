// Ported from: packages/togetherai/src/reranking/togetherai-reranking-api.ts
package togetherai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// TogetherAIRerankingInput represents the request body for the Together AI rerank API.
// https://docs.together.ai/reference/rerank-1
type TogetherAIRerankingInput struct {
	Model           string        `json:"model"`
	Query           string        `json:"query"`
	Documents       interface{}   `json:"documents"`
	TopN            *int          `json:"top_n,omitempty"`
	ReturnDocuments *bool         `json:"return_documents,omitempty"`
	RankFields      []string      `json:"rank_fields,omitempty"`
}

// togetheraiErrorData represents the Together AI error response structure.
type togetheraiErrorData struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// togetheraiErrorSchema validates the Together AI error response.
var togetheraiErrorSchema = &providerutils.Schema[togetheraiErrorData]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[togetheraiErrorData], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[togetheraiErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		var errData togetheraiErrorData
		if err := json.Unmarshal(data, &errData); err != nil {
			return &providerutils.ValidationResult[togetheraiErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[togetheraiErrorData]{
			Success: true,
			Value:   errData,
		}, nil
	},
}

// togetheraiRerankingResult represents a single result in the reranking response.
type togetheraiRerankingResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

// togetheraiRerankingUsage represents the usage information in the reranking response.
type togetheraiRerankingUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// togetheraiRerankingResponse represents the Together AI reranking API response.
type togetheraiRerankingResponse struct {
	ID      *string                     `json:"id,omitempty"`
	Model   *string                     `json:"model,omitempty"`
	Results []togetheraiRerankingResult `json:"results"`
	Usage   togetheraiRerankingUsage    `json:"usage"`
}

// togetheraiRerankingResponseSchema validates the Together AI reranking response.
var togetheraiRerankingResponseSchema = &providerutils.Schema[togetheraiRerankingResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[togetheraiRerankingResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[togetheraiRerankingResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp togetheraiRerankingResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[togetheraiRerankingResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[togetheraiRerankingResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

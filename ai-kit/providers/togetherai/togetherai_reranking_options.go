// Ported from: packages/togetherai/src/reranking/togetherai-reranking-options.ts
package togetherai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// https://docs.together.ai/docs/serverless-models#rerank-models

// TogetherAIRerankingModelID is the type for Together AI reranking model identifiers.
// Known model IDs are provided as constants; any string is accepted.
type TogetherAIRerankingModelID = string

// Known Together AI reranking model IDs.
const (
	RerankingModelSalesforceLlamaRankV1 TogetherAIRerankingModelID = "Salesforce/Llama-Rank-v1"
	RerankingModelMxbaiRerankLargeV2    TogetherAIRerankingModelID = "mixedbread-ai/Mxbai-Rerank-Large-V2"
)

// TogetherAIRerankingModelOptions contains provider-specific options for reranking.
type TogetherAIRerankingModelOptions struct {
	// RankFields is a list of keys in the JSON Object document to rank by.
	// Defaults to use all supplied keys for ranking.
	//
	// Example: ["title", "text"]
	RankFields []string `json:"rankFields,omitempty"`
}

// togetheraiRerankingModelOptionsSchema validates TogetherAIRerankingModelOptions.
var togetheraiRerankingModelOptionsSchema = &providerutils.Schema[TogetherAIRerankingModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[TogetherAIRerankingModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[TogetherAIRerankingModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts TogetherAIRerankingModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[TogetherAIRerankingModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[TogetherAIRerankingModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

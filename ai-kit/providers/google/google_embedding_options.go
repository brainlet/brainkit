// Ported from: packages/google/src/google-generative-ai-embedding-options.ts
package google

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GoogleEmbeddingModelOptions contains the provider-specific options for Google
// Generative AI embedding models.
type GoogleEmbeddingModelOptions struct {
	// OutputDimensionality is an optional reduced dimension for the output embedding.
	// If set, excessive values in the output embedding are truncated from the end.
	OutputDimensionality *int `json:"outputDimensionality,omitempty"`

	// TaskType specifies the task type for generating embeddings.
	// Supported values: SEMANTIC_SIMILARITY, CLASSIFICATION, CLUSTERING,
	// RETRIEVAL_DOCUMENT, RETRIEVAL_QUERY, QUESTION_ANSWERING,
	// FACT_VERIFICATION, CODE_RETRIEVAL_QUERY.
	TaskType *string `json:"taskType,omitempty"`
}

// GoogleEmbeddingModelOptionsSchema is the providerutils.Schema used to validate
// and parse GoogleEmbeddingModelOptions from provider options maps.
var GoogleEmbeddingModelOptionsSchema = &providerutils.Schema[GoogleEmbeddingModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[GoogleEmbeddingModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[GoogleEmbeddingModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts GoogleEmbeddingModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[GoogleEmbeddingModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[GoogleEmbeddingModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

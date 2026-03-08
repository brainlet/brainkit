// Ported from: packages/openai/src/embedding/openai-embedding-options.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAIEmbeddingModelID is the identifier for an OpenAI embedding model.
type OpenAIEmbeddingModelID = string

// OpenAIEmbeddingModelOptions contains provider-specific options for
// OpenAI embedding models.
type OpenAIEmbeddingModelOptions struct {
	// Dimensions is the number of dimensions the resulting output embeddings
	// should have. Only supported in text-embedding-3 and later models.
	Dimensions *int `json:"dimensions,omitempty"`

	// User is a unique identifier representing the end-user, which can help
	// OpenAI to monitor and detect abuse.
	User *string `json:"user,omitempty"`
}

// openaiEmbeddingModelOptionsSchema is the providerutils.Schema used to validate
// and parse OpenAIEmbeddingModelOptions from provider options maps.
var openaiEmbeddingModelOptionsSchema = &providerutils.Schema[OpenAIEmbeddingModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[OpenAIEmbeddingModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[OpenAIEmbeddingModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts OpenAIEmbeddingModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[OpenAIEmbeddingModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[OpenAIEmbeddingModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

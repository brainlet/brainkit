// Ported from: packages/openai-compatible/src/embedding/openai-compatible-embedding-options.ts
package openaicompatible

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// EmbeddingModelID is the identifier for an OpenAI-compatible embedding model.
type EmbeddingModelID = string

// EmbeddingOptions contains provider-specific options for OpenAI-compatible
// embedding models.
type EmbeddingOptions struct {
	// Dimensions is the number of dimensions the resulting output embeddings
	// should have. Only supported in text-embedding-3 and later models.
	Dimensions *int `json:"dimensions,omitempty"`

	// User is a unique identifier representing the end-user, which can help
	// providers monitor and detect abuse.
	User *string `json:"user,omitempty"`
}

// EmbeddingOptionsSchema is the providerutils.Schema used to validate and parse
// EmbeddingOptions from provider options maps.
var EmbeddingOptionsSchema = &providerutils.Schema[EmbeddingOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[EmbeddingOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[EmbeddingOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts EmbeddingOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[EmbeddingOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[EmbeddingOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

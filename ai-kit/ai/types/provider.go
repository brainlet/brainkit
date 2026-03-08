// Ported from: packages/ai/src/types/provider.ts
package aitypes

// Provider is a provider for language, text embedding, and image models.
type Provider interface {
	// LanguageModel returns the language model with the given id.
	// The model id is then passed to the provider function to get the model.
	//
	// Returns an error if no such model exists.
	LanguageModel(modelID string) (LanguageModel, error)

	// EmbeddingModel returns the text embedding model with the given id.
	// The model id is then passed to the provider function to get the model.
	//
	// Returns an error if no such model exists.
	EmbeddingModel(modelID string) (EmbeddingModel, error)

	// ImageModel returns the image model with the given id.
	// The model id is then passed to the provider function to get the model.
	ImageModel(modelID string) (ImageModel, error)

	// RerankingModel returns the reranking model with the given id.
	// The model id is then passed to the provider function to get the model.
	//
	// Returns an error if no such model exists.
	RerankingModel(modelID string) (RerankingModel, error)
}

// Ported from: packages/fireworks/src/fireworks-embedding-options.ts
package fireworks

// FireworksEmbeddingModelID represents a Fireworks embedding model identifier.
// Below is just a subset of the available models.
type FireworksEmbeddingModelID = string

const (
	FireworksEmbeddingModelNomicEmbedTextV1p5 FireworksEmbeddingModelID = "nomic-ai/nomic-embed-text-v1.5"
)

// FireworksEmbeddingModelOptions are the provider-specific options for Fireworks embedding models.
type FireworksEmbeddingModelOptions struct{}

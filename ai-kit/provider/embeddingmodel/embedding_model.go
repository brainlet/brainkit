// Ported from: packages/provider/src/embedding-model/v3/embedding-model-v3.ts
package embeddingmodel

// EmbeddingModel is the specification for an embedding model that implements
// the embedding model interface version 3.
//
// It is specific to text embeddings.
type EmbeddingModel interface {
	// SpecificationVersion returns the embedding model interface version.
	// Must return "v3".
	SpecificationVersion() string

	// Provider returns the name of the provider for logging purposes.
	Provider() string

	// ModelID returns the provider-specific model ID for logging purposes.
	ModelID() string

	// MaxEmbeddingsPerCall returns the limit of how many embeddings can be
	// generated in a single API call. Returns nil for no limit.
	MaxEmbeddingsPerCall() (*int, error)

	// SupportsParallelCalls returns true if the model can handle multiple
	// embedding calls in parallel.
	SupportsParallelCalls() (bool, error)

	// DoEmbed generates a list of embeddings for the given input text.
	//
	// Naming: "Do" prefix to prevent accidental direct usage of the method
	// by the user.
	DoEmbed(options CallOptions) (Result, error)
}

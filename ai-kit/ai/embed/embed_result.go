// Ported from: packages/ai/src/embed/embed-result.ts
package embed

// Embedding is a vector representation of a value.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type Embedding = []float64

// EmbeddingModelUsage represents token usage for embedding operations.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type EmbeddingModelUsage struct {
	// Tokens is the number of tokens used.
	Tokens int `json:"tokens"`
}

// Warning from the model provider for this call.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type Warning struct {
	Type    string `json:"type"`
	Feature string `json:"feature,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

// ProviderMetadata is additional provider-specific metadata.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type ProviderMetadata = map[string]map[string]any

// EmbedResult is the result of an embed call.
// It contains the embedding, the value, and additional information.
type EmbedResult struct {
	// Value is the value that was embedded.
	Value string

	// Embedding is the embedding of the value.
	Embedding Embedding

	// Usage is the embedding token usage.
	Usage EmbeddingModelUsage

	// Warnings for the call, e.g. unsupported settings.
	Warnings []Warning

	// ProviderMetadata is optional provider-specific metadata.
	ProviderMetadata ProviderMetadata

	// Response is optional response data.
	Response *EmbedResponseData
}

// EmbedResponseData holds optional response data from the provider.
type EmbedResponseData struct {
	// Headers are the response headers.
	Headers map[string]string
	// Body is the response body.
	Body any
}

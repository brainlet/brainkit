// Ported from: packages/provider/src/embedding-model/v3/embedding-model-v3-result.ts
package embeddingmodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// Result is the result of an embedding model doEmbed call.
type Result struct {
	// Embeddings are generated embeddings. They are in the same order as the input values.
	Embeddings []Embedding

	// Usage is token usage. Only input tokens for embeddings.
	Usage *EmbeddingUsage

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata shared.ProviderMetadata

	// Response contains optional response information for debugging purposes.
	Response *ResultResponse

	// Warnings for the call, e.g. unsupported settings.
	Warnings []shared.Warning
}

// EmbeddingUsage contains token usage for an embedding call.
type EmbeddingUsage struct {
	Tokens int
}

// ResultResponse contains response information for debugging.
type ResultResponse struct {
	// Headers are the response headers.
	Headers shared.Headers

	// Body is the response body.
	Body any
}

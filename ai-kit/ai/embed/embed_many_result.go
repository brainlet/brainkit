// Ported from: packages/ai/src/embed/embed-many-result.ts
package embed

// EmbedManyResult is the result of an embedMany call.
// It contains the embeddings, the values, and additional information.
type EmbedManyResult struct {
	// Values are the values that were embedded.
	Values []string

	// Embeddings are the embeddings. They are in the same order as the values.
	Embeddings []Embedding

	// Usage is the embedding token usage.
	Usage EmbeddingModelUsage

	// Warnings for the call, e.g. unsupported settings.
	Warnings []Warning

	// ProviderMetadata is optional provider-specific metadata.
	ProviderMetadata ProviderMetadata

	// Responses are optional raw response data.
	Responses []*EmbedResponseData
}

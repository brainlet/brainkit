// Ported from: packages/ai/src/rerank/rerank-result.ts
package rerank

import "time"

// ProviderMetadata is additional provider-specific metadata.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type ProviderMetadata = map[string]map[string]any

// RankedDocument represents a document with its ranking information.
type RankedDocument struct {
	// OriginalIndex is the index of the document in the original array.
	OriginalIndex int

	// Score is the relevance score assigned by the model.
	Score float64

	// Document is the original document value.
	Document any
}

// ResponseData holds optional response data from the provider.
type ResponseData struct {
	// ID is the response ID if the provider sends one.
	ID string

	// Timestamp of the generated response.
	Timestamp time.Time

	// ModelID is the ID of the model that was used.
	ModelID string

	// Headers are response headers.
	Headers map[string]string

	// Body is the response body.
	Body any
}

// RerankResult is the result of a rerank call.
// It contains the original documents, the reranked documents, and additional information.
type RerankResult struct {
	// OriginalDocuments are the original documents that were reranked.
	OriginalDocuments []any

	// Ranking is a list of objects with the original index,
	// relevance score, and the reranked document.
	// Sorted by relevance score in descending order.
	Ranking []RankedDocument

	// ProviderMetadata is optional provider-specific metadata.
	ProviderMetadata ProviderMetadata

	// Response is optional raw response data.
	Response ResponseData
}

// RerankedDocuments returns the documents sorted by relevance score in descending order.
func (r *RerankResult) RerankedDocuments() []any {
	docs := make([]any, len(r.Ranking))
	for i, ranked := range r.Ranking {
		docs[i] = ranked.Document
	}
	return docs
}

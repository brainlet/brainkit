// Ported from: packages/provider/src/reranking-model/v3/reranking-model-v3.ts
package rerankingmodel

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// RankedDocument represents a document after reranking.
type RankedDocument struct {
	// Index is the index of the document in the original list before reranking.
	Index int

	// RelevanceScore is the relevance score of the document after reranking.
	RelevanceScore float64
}

// RerankResult is the result of a reranking model doRerank call.
type RerankResult struct {
	// Ranking is an ordered list of reranked documents (sorted by descending relevance score).
	Ranking []RankedDocument

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata shared.ProviderMetadata

	// Warnings for the call, e.g. unsupported settings.
	Warnings []shared.Warning

	// Response contains optional response information for debugging purposes.
	Response *RerankResultResponse
}

// RerankResultResponse contains response information for debugging.
type RerankResultResponse struct {
	// ID is the generated response ID, if the provider sends one.
	ID *string

	// Timestamp for the start of the generated response.
	Timestamp *time.Time

	// ModelID is the response model ID.
	ModelID *string

	// Headers are the response headers.
	Headers shared.Headers

	// Body is the response body.
	Body any
}

// RerankingModel is the specification for a reranking model (version 3).
type RerankingModel interface {
	// SpecificationVersion returns the reranking model interface version.
	// Must return "v3".
	SpecificationVersion() string

	// Provider returns the provider ID.
	Provider() string

	// ModelID returns the provider-specific model ID.
	ModelID() string

	// DoRerank reranks a list of documents using the query.
	//
	// Naming: "Do" prefix to prevent accidental direct usage of the method
	// by the user.
	DoRerank(options CallOptions) (RerankResult, error)
}

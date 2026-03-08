// Ported from: packages/core/src/vector/types.ts
package vector

import (
	"github.com/brainlet/brainkit/agent-kit/core/vector/filter"
)

// SparseVector represents a high-dimensional vector with only non-zero values stored.
type SparseVector struct {
	// Indices holds the dimension indices for non-zero values.
	Indices []int `json:"indices"`
	// Values holds the values corresponding to the indices.
	Values []float64 `json:"values"`
}

// QueryResult holds a single result from a vector similarity query.
type QueryResult struct {
	ID    string         `json:"id"`
	Score float64        `json:"score"`
	// Metadata contains optional key-value metadata associated with the vector.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Vector contains the raw vector values, if requested.
	Vector []float64 `json:"vector,omitempty"`
	// Document contains the document content, if available.
	// Note: Currently only supported by Chroma vector store.
	// For other vector stores, documents should be stored in metadata.
	Document string `json:"document,omitempty"`
}

// DistanceMetric represents the distance metric used by a vector index.
type DistanceMetric string

const (
	DistanceMetricCosine     DistanceMetric = "cosine"
	DistanceMetricEuclidean  DistanceMetric = "euclidean"
	DistanceMetricDotProduct DistanceMetric = "dotproduct"
)

// IndexStats holds statistics about a vector index.
type IndexStats struct {
	Dimension int            `json:"dimension"`
	Count     int            `json:"count"`
	Metric    DistanceMetric `json:"metric,omitempty"`
}

// UpsertVectorParams holds parameters for upserting vectors into an index.
type UpsertVectorParams struct {
	IndexName string `json:"indexName"`
	// Vectors is the array of embedding vectors to upsert.
	Vectors [][]float64 `json:"vectors"`
	// Metadata is an optional array of metadata maps, one per vector.
	Metadata []map[string]any `json:"metadata,omitempty"`
	// IDs is an optional array of vector IDs. If omitted, IDs are auto-generated.
	IDs []string `json:"ids,omitempty"`
	// SparseVectors is an optional array of sparse vectors for hybrid search.
	SparseVectors []SparseVector `json:"sparseVectors,omitempty"`
	// DeleteFilter is an optional filter to delete vectors before upserting.
	// Useful for replacing all chunks from a source document.
	// The delete and insert operations happen atomically in a transaction.
	DeleteFilter filter.VectorFilter `json:"deleteFilter,omitempty"`
}

// CreateIndexParams holds parameters for creating a new vector index.
type CreateIndexParams struct {
	IndexName string         `json:"indexName"`
	Dimension int            `json:"dimension"`
	Metric    DistanceMetric `json:"metric,omitempty"`
}

// QueryVectorParams holds parameters for querying vectors from an index.
type QueryVectorParams struct {
	IndexName string `json:"indexName"`
	// QueryVector is the query vector for similarity search.
	// Optional — when omitted, a metadata-only query is performed using Filter.
	// At least one of QueryVector or Filter must be provided.
	//
	// Note: Not all vector store backends support metadata-only queries.
	// Check your store's documentation for support details.
	QueryVector []float64 `json:"queryVector,omitempty"`
	// TopK is the number of results to return.
	TopK int `json:"topK,omitempty"`
	// Filter is an optional metadata filter for the query.
	Filter filter.VectorFilter `json:"filter,omitempty"`
	// IncludeVector controls whether to include the raw vector in results.
	IncludeVector bool `json:"includeVector,omitempty"`
	// SparseVector is an optional sparse vector for hybrid query.
	SparseVector *SparseVector `json:"sparseVector,omitempty"`
}

// DescribeIndexParams holds parameters for describing a vector index.
type DescribeIndexParams struct {
	IndexName string `json:"indexName"`
}

// DeleteIndexParams holds parameters for deleting a vector index.
type DeleteIndexParams struct {
	IndexName string `json:"indexName"`
}

// VectorUpdate holds the fields that can be updated on a vector.
type VectorUpdate struct {
	// Vector is the new embedding vector values.
	Vector []float64 `json:"vector,omitempty"`
	// Metadata is the new metadata to set.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// UpdateVectorParams holds parameters for updating vectors.
// In TypeScript this is a discriminated union enforcing mutual exclusivity of ID and Filter.
// In Go, callers should provide exactly one of ID or Filter.
type UpdateVectorParams struct {
	IndexName string `json:"indexName"`
	// ID targets a single vector by its ID. Mutually exclusive with Filter.
	ID string `json:"id,omitempty"`
	// Filter targets multiple vectors matching the filter. Mutually exclusive with ID.
	Filter filter.VectorFilter `json:"filter,omitempty"`
	// Update contains the fields to update.
	Update VectorUpdate `json:"update"`
}

// DeleteVectorParams holds parameters for deleting a single vector by ID.
type DeleteVectorParams struct {
	IndexName string `json:"indexName"`
	ID        string `json:"id"`
}

// DeleteVectorsParams holds parameters for deleting multiple vectors.
// Provide either IDs or Filter, but not both.
type DeleteVectorsParams struct {
	IndexName string `json:"indexName"`
	// IDs deletes multiple vectors by their IDs. Mutually exclusive with Filter.
	IDs []string `json:"ids,omitempty"`
	// Filter deletes vectors matching a metadata filter. Mutually exclusive with IDs.
	// Uses the same filter syntax as query operations.
	Filter filter.VectorFilter `json:"filter,omitempty"`
}

// Ported from: packages/core/src/workspace/search/search-engine.ts
package search

import (
	"fmt"
	"math"
	"sort"
)

// =============================================================================
// Types
// =============================================================================

// SearchMode represents the search mode.
type SearchMode string

const (
	SearchModeVector SearchMode = "vector"
	SearchModeBM25   SearchMode = "bm25"
	SearchModeHybrid SearchMode = "hybrid"
)

// Embedder is a function that takes text and returns embeddings.
type Embedder func(text string) ([]float64, error)

// MastraVector is the interface for a vector store.
// This is a minimal interface matching the methods used by SearchEngine.
type MastraVector interface {
	Upsert(params VectorUpsertParams) error
	Query(params VectorQueryParams) ([]VectorQueryResult, error)
	DeleteVector(params VectorDeleteParams) error
}

// VectorUpsertParams holds parameters for upserting vectors.
type VectorUpsertParams struct {
	IndexName string
	Vectors   [][]float64
	Metadata  []map[string]interface{}
	IDs       []string
}

// VectorQueryParams holds parameters for querying vectors.
type VectorQueryParams struct {
	IndexName   string
	QueryVector []float64
	TopK        int
	Filter      map[string]interface{}
}

// VectorQueryResult is a result from a vector query.
type VectorQueryResult struct {
	ID       string
	Score    float64
	Metadata map[string]interface{}
}

// VectorDeleteParams holds parameters for deleting a vector.
type VectorDeleteParams struct {
	IndexName string
	ID        string
}

// VectorConfig holds configuration for vector search.
type VectorConfig struct {
	// VectorStore is the vector store for semantic search.
	VectorStore MastraVector
	// Embedder is the function for generating vectors.
	Embedder Embedder
	// IndexName is the index name for the vector store.
	IndexName string
}

// BM25SearchConfig holds configuration for BM25 search.
type BM25SearchConfig struct {
	// BM25 holds BM25 algorithm parameters.
	BM25 *BM25Config
	// Tokenize holds tokenization options.
	Tokenize *TokenizeOptions
}

// IndexDocument is a document to be indexed.
type IndexDocument struct {
	// ID is the unique identifier for this document.
	ID string
	// Content is the text content to index.
	Content string
	// Metadata holds optional metadata to store with the document.
	Metadata map[string]interface{}
	// StartLineOffset is the starting line number of this chunk in the original document.
	// When provided, lineRange in search results will be adjusted. (1-indexed)
	StartLineOffset int
}

// SearchResult is a result from a search operation.
type SearchResult struct {
	// ID is the document identifier.
	ID string
	// Content is the document content.
	Content string
	// Score is the search score (0-1 for normalized results).
	Score float64
	// LineRange is where query terms appear.
	LineRange *LineRange
	// Metadata holds optional metadata.
	Metadata map[string]interface{}
	// ScoreDetails has score breakdown by search type.
	ScoreDetails *ScoreDetails
}

// ScoreDetails holds score breakdown by search type.
type ScoreDetails struct {
	Vector *float64
	BM25   *float64
}

// SearchOptions holds options for searching.
type SearchOptions struct {
	// TopK is the maximum number of results to return.
	TopK int
	// MinScore is the minimum score threshold.
	MinScore *float64
	// Mode is the search mode: 'bm25', 'vector', or 'hybrid'.
	Mode SearchMode
	// VectorWeight is the weight for vector scores in hybrid search (0-1, default 0.5).
	VectorWeight float64
	// Filter is the filter for vector search.
	Filter map[string]interface{}
}

// SearchEngineConfig holds configuration for SearchEngine.
type SearchEngineConfig struct {
	// BM25 enables BM25 search.
	BM25 *BM25SearchConfig
	// Vector enables vector search.
	Vector *VectorConfig
	// LazyVectorIndex uses lazy vector indexing (default: false = eager).
	LazyVectorIndex bool
}

// =============================================================================
// SearchEngine
// =============================================================================

// SearchEngine is a unified search engine supporting BM25, vector, and hybrid search.
type SearchEngine struct {
	bm25Index        *BM25Index
	tokenizeOptions  *TokenizeOptions
	vectorConfig     *VectorConfig
	lazyVectorIndex  bool
	pendingVectorDocs []IndexDocument
	vectorIndexBuilt  bool
}

// NewSearchEngine creates a new SearchEngine.
func NewSearchEngine(config *SearchEngineConfig) *SearchEngine {
	se := &SearchEngine{}

	if config == nil {
		return se
	}

	// Initialize BM25 if configured
	if config.BM25 != nil {
		se.tokenizeOptions = config.BM25.Tokenize
		se.bm25Index = NewBM25Index(config.BM25.BM25, se.tokenizeOptions)
	}

	// Store vector config if provided
	if config.Vector != nil {
		se.vectorConfig = config.Vector
	}

	se.lazyVectorIndex = config.LazyVectorIndex

	return se
}

// =============================================================================
// Public API
// =============================================================================

// Index indexes a document for search.
func (se *SearchEngine) Index(doc IndexDocument) error {
	// Merge startLineOffset into metadata for retrieval at search time
	metadata := make(map[string]interface{})
	for k, v := range doc.Metadata {
		metadata[k] = v
	}
	if doc.StartLineOffset != 0 {
		metadata["_startLineOffset"] = doc.StartLineOffset
	}

	// BM25 indexing
	if se.bm25Index != nil {
		se.bm25Index.Add(doc.ID, doc.Content, metadata)
	}

	// Vector indexing
	if se.vectorConfig != nil {
		docWithMeta := IndexDocument{
			ID:              doc.ID,
			Content:         doc.Content,
			Metadata:        metadata,
			StartLineOffset: doc.StartLineOffset,
		}
		if se.lazyVectorIndex {
			se.pendingVectorDocs = append(se.pendingVectorDocs, docWithMeta)
			se.vectorIndexBuilt = false
		} else {
			if err := se.indexVector(docWithMeta); err != nil {
				return err
			}
		}
	}

	return nil
}

// IndexMany indexes multiple documents.
func (se *SearchEngine) IndexMany(docs []IndexDocument) error {
	for _, doc := range docs {
		if err := se.Index(doc); err != nil {
			return err
		}
	}
	return nil
}

// Remove removes a document from the index.
func (se *SearchEngine) Remove(id string) error {
	if se.bm25Index != nil {
		se.bm25Index.Remove(id)
	}

	if se.vectorConfig != nil {
		_ = se.vectorConfig.VectorStore.DeleteVector(VectorDeleteParams{
			IndexName: se.vectorConfig.IndexName,
			ID:        id,
		})

		if se.lazyVectorIndex {
			filtered := se.pendingVectorDocs[:0]
			for _, d := range se.pendingVectorDocs {
				if d.ID != id {
					filtered = append(filtered, d)
				}
			}
			se.pendingVectorDocs = filtered
		}
	}

	return nil
}

// Clear clears all indexed documents.
func (se *SearchEngine) Clear() {
	if se.bm25Index != nil {
		se.bm25Index.Clear()
	}
	se.pendingVectorDocs = nil
	se.vectorIndexBuilt = false
}

// Search searches for documents.
func (se *SearchEngine) Search(query string, options *SearchOptions) ([]SearchResult, error) {
	opts := &SearchOptions{
		TopK:         10,
		VectorWeight: 0.5,
	}
	if options != nil {
		if options.TopK > 0 {
			opts.TopK = options.TopK
		}
		opts.MinScore = options.MinScore
		opts.Mode = options.Mode
		if options.VectorWeight > 0 {
			opts.VectorWeight = options.VectorWeight
		}
		opts.Filter = options.Filter
	}

	effectiveMode, err := se.determineSearchMode(opts.Mode)
	if err != nil {
		return nil, err
	}

	switch effectiveMode {
	case SearchModeBM25:
		return se.searchBM25(query, opts.TopK, opts.MinScore), nil
	case SearchModeVector:
		return se.searchVector(query, opts.TopK, opts.MinScore, opts.Filter)
	case SearchModeHybrid:
		return se.searchHybrid(query, opts.TopK, opts.MinScore, opts.VectorWeight, opts.Filter)
	default:
		return nil, fmt.Errorf("unknown search mode: %s", effectiveMode)
	}
}

// CanBM25 returns whether BM25 search is available.
func (se *SearchEngine) CanBM25() bool {
	return se.bm25Index != nil
}

// CanVector returns whether vector search is available.
func (se *SearchEngine) CanVector() bool {
	return se.vectorConfig != nil
}

// CanHybrid returns whether hybrid search is available.
func (se *SearchEngine) CanHybrid() bool {
	return se.CanBM25() && se.CanVector()
}

// BM25Index returns the BM25 index (for serialization/debugging).
func (se *SearchEngine) BM25Index() *BM25Index {
	return se.bm25Index
}

// =============================================================================
// Private Methods
// =============================================================================

func (se *SearchEngine) determineSearchMode(requested SearchMode) (SearchMode, error) {
	if requested != "" {
		switch requested {
		case SearchModeVector:
			if !se.CanVector() {
				return "", fmt.Errorf("Vector search requires vector configuration.")
			}
		case SearchModeBM25:
			if !se.CanBM25() {
				return "", fmt.Errorf("BM25 search requires BM25 configuration.")
			}
		case SearchModeHybrid:
			if !se.CanHybrid() {
				return "", fmt.Errorf("Hybrid search requires both vector and BM25 configuration.")
			}
		}
		return requested, nil
	}

	if se.CanHybrid() {
		return SearchModeHybrid, nil
	}
	if se.CanVector() {
		return SearchModeVector, nil
	}
	if se.CanBM25() {
		return SearchModeBM25, nil
	}

	return "", fmt.Errorf("No search configuration available. Provide bm25 or vector config.")
}

func (se *SearchEngine) indexVector(doc IndexDocument) error {
	if se.vectorConfig == nil {
		return nil
	}

	embedding, err := se.vectorConfig.Embedder(doc.Content)
	if err != nil {
		return err
	}

	metadata := map[string]interface{}{
		"id":   doc.ID,
		"text": doc.Content,
	}
	for k, v := range doc.Metadata {
		metadata[k] = v
	}

	return se.vectorConfig.VectorStore.Upsert(VectorUpsertParams{
		IndexName: se.vectorConfig.IndexName,
		Vectors:   [][]float64{embedding},
		Metadata:  []map[string]interface{}{metadata},
		IDs:       []string{doc.ID},
	})
}

func (se *SearchEngine) ensureVectorIndex() error {
	if !se.lazyVectorIndex || se.vectorIndexBuilt || len(se.pendingVectorDocs) == 0 {
		return nil
	}

	for _, doc := range se.pendingVectorDocs {
		if err := se.indexVector(doc); err != nil {
			return err
		}
	}

	se.pendingVectorDocs = nil
	se.vectorIndexBuilt = true
	return nil
}

func (se *SearchEngine) searchBM25(query string, topK int, minScore *float64) []SearchResult {
	if se.bm25Index == nil {
		return nil
	}

	ms := 0.0
	if minScore != nil {
		ms = *minScore
	}

	results := se.bm25Index.Search(query, topK, ms)
	queryTokens := Tokenize(query, se.tokenizeOptions)

	searchResults := make([]SearchResult, 0, len(results))
	for _, result := range results {
		rawLineRange := FindLineRange(result.Content, queryTokens, se.tokenizeOptions)
		lineRange := se.adjustLineRange(rawLineRange, result.Metadata)

		// Clean metadata — remove internal fields
		cleanMeta := make(map[string]interface{})
		for k, v := range result.Metadata {
			if k != "_startLineOffset" {
				cleanMeta[k] = v
			}
		}
		var meta map[string]interface{}
		if len(cleanMeta) > 0 {
			meta = cleanMeta
		}

		score := result.Score
		searchResults = append(searchResults, SearchResult{
			ID:       result.ID,
			Content:  result.Content,
			Score:    score,
			LineRange: lineRange,
			Metadata: meta,
			ScoreDetails: &ScoreDetails{BM25: &score},
		})
	}

	return searchResults
}

func (se *SearchEngine) searchVector(query string, topK int, minScore *float64, filter map[string]interface{}) ([]SearchResult, error) {
	if se.vectorConfig == nil {
		return nil, fmt.Errorf("Vector search requires vector configuration.")
	}

	if err := se.ensureVectorIndex(); err != nil {
		return nil, err
	}

	queryEmbedding, err := se.vectorConfig.Embedder(query)
	if err != nil {
		return nil, err
	}

	vectorResults, err := se.vectorConfig.VectorStore.Query(VectorQueryParams{
		IndexName:   se.vectorConfig.IndexName,
		QueryVector: queryEmbedding,
		TopK:        topK,
		Filter:      filter,
	})
	if err != nil {
		return nil, err
	}

	queryTokens := Tokenize(query, se.tokenizeOptions)
	var results []SearchResult

	for _, result := range vectorResults {
		if minScore != nil && result.Score < *minScore {
			continue
		}

		id := result.ID
		if metaID, ok := result.Metadata["id"].(string); ok && metaID != "" {
			id = metaID
		}

		content := ""
		if metaText, ok := result.Metadata["text"].(string); ok {
			content = metaText
		}

		// Clean metadata
		cleanMeta := make(map[string]interface{})
		for k, v := range result.Metadata {
			if k != "id" && k != "text" && k != "_startLineOffset" {
				cleanMeta[k] = v
			}
		}
		var meta map[string]interface{}
		if len(cleanMeta) > 0 {
			meta = cleanMeta
		}

		rawLineRange := FindLineRange(content, queryTokens, se.tokenizeOptions)
		lineRange := se.adjustLineRange(rawLineRange, result.Metadata)

		score := result.Score
		results = append(results, SearchResult{
			ID:       id,
			Content:  content,
			Score:    score,
			LineRange: lineRange,
			Metadata: meta,
			ScoreDetails: &ScoreDetails{Vector: &score},
		})
	}

	return results, nil
}

func (se *SearchEngine) searchHybrid(query string, topK int, minScore *float64, vectorWeight float64, filter map[string]interface{}) ([]SearchResult, error) {
	expandedTopK := topK * 2
	if expandedTopK > 50 {
		expandedTopK = 50
	}

	vectorResults, err := se.searchVector(query, expandedTopK, nil, filter)
	if err != nil {
		return nil, err
	}
	bm25Results := se.searchBM25(query, expandedTopK, nil)

	// Normalize BM25 scores to 0-1 range
	normalizedBM25 := se.normalizeBM25Scores(bm25Results)

	// Create score maps
	bm25Map := make(map[string]SearchResult)
	for _, r := range normalizedBM25 {
		bm25Map[r.ID] = r
	}

	vectorMap := make(map[string]SearchResult)
	for _, r := range vectorResults {
		vectorMap[r.ID] = r
	}

	// Combine scores
	bm25Weight := 1 - vectorWeight
	allIDs := make(map[string]bool)
	for id := range vectorMap {
		allIDs[id] = true
	}
	for id := range bm25Map {
		allIDs[id] = true
	}

	var combinedResults []SearchResult
	for id := range allIDs {
		vectorResult, hasVector := vectorMap[id]
		bm25Result, hasBM25 := bm25Map[id]

		vectorScore := 0.0
		if hasVector && vectorResult.ScoreDetails != nil && vectorResult.ScoreDetails.Vector != nil {
			vectorScore = *vectorResult.ScoreDetails.Vector
		}
		bm25Score := 0.0
		if hasBM25 {
			bm25Score = bm25Result.Score
		}

		combinedScore := vectorWeight*vectorScore + bm25Weight*bm25Score

		var base SearchResult
		if hasVector {
			base = vectorResult
		} else {
			base = bm25Result
		}

		var lr *LineRange
		if hasBM25 {
			lr = bm25Result.LineRange
		} else if hasVector {
			lr = vectorResult.LineRange
		}

		var vScore, bScore *float64
		if hasVector && vectorResult.ScoreDetails != nil {
			vScore = vectorResult.ScoreDetails.Vector
		}
		if hasBM25 && bm25Result.ScoreDetails != nil {
			bScore = bm25Result.ScoreDetails.BM25
		}

		combinedResults = append(combinedResults, SearchResult{
			ID:        id,
			Content:   base.Content,
			Score:     combinedScore,
			LineRange: lr,
			Metadata:  base.Metadata,
			ScoreDetails: &ScoreDetails{
				Vector: vScore,
				BM25:   bScore,
			},
		})
	}

	sort.Slice(combinedResults, func(i, j int) bool {
		return combinedResults[i].Score > combinedResults[j].Score
	})

	if minScore != nil {
		filtered := combinedResults[:0]
		for _, r := range combinedResults {
			if r.Score >= *minScore {
				filtered = append(filtered, r)
			}
		}
		combinedResults = filtered
	}

	if len(combinedResults) > topK {
		combinedResults = combinedResults[:topK]
	}

	return combinedResults, nil
}

func (se *SearchEngine) normalizeBM25Scores(results []SearchResult) []SearchResult {
	if len(results) == 0 {
		return results
	}

	maxScore := -math.MaxFloat64
	minScoreVal := math.MaxFloat64
	for _, r := range results {
		score := r.Score
		if r.ScoreDetails != nil && r.ScoreDetails.BM25 != nil {
			score = *r.ScoreDetails.BM25
		}
		if score > maxScore {
			maxScore = score
		}
		if score < minScoreVal {
			minScoreVal = score
		}
	}

	scoreRange := maxScore - minScoreVal
	if scoreRange == 0 {
		normalized := make([]SearchResult, len(results))
		for i, r := range results {
			r.Score = 1
			normalized[i] = r
		}
		return normalized
	}

	normalized := make([]SearchResult, len(results))
	for i, r := range results {
		score := r.Score
		if r.ScoreDetails != nil && r.ScoreDetails.BM25 != nil {
			score = *r.ScoreDetails.BM25
		}
		r.Score = (score - minScoreVal) / scoreRange
		normalized[i] = r
	}
	return normalized
}

func (se *SearchEngine) adjustLineRange(lineRange *LineRange, metadata map[string]interface{}) *LineRange {
	if lineRange == nil {
		return nil
	}

	offset, ok := metadata["_startLineOffset"]
	if !ok {
		return lineRange
	}

	startLineOffset, ok := offset.(int)
	if !ok {
		if f, fok := offset.(float64); fok {
			startLineOffset = int(f)
		} else {
			return lineRange
		}
	}

	return &LineRange{
		Start: lineRange.Start + startLineOffset - 1,
		End:   lineRange.End + startLineOffset - 1,
	}
}

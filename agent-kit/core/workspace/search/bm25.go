// Ported from: packages/core/src/workspace/search/bm25.ts
package search

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

// =============================================================================
// BM25 Types
// =============================================================================

// BM25Config holds BM25 algorithm parameters.
type BM25Config struct {
	// K1 controls term frequency saturation.
	// Higher values give more weight to term frequency.
	// Typical range: 1.2 - 2.0. Default: 1.5.
	K1 float64
	// B controls document length normalization.
	// 0 = no length normalization, 1 = full normalization.
	// Default: 0.75.
	B float64
}

// BM25Document represents a document in the BM25 index.
type BM25Document struct {
	// ID is the document identifier.
	ID string
	// Content is the document content.
	Content string
	// Tokens are the pre-computed tokens for the document.
	Tokens []string
	// TermFrequencies maps terms to their frequency in the document.
	TermFrequencies map[string]int
	// Length is the total number of tokens.
	Length int
	// Metadata holds optional metadata.
	Metadata map[string]interface{}
}

// BM25SearchResult is a result from a BM25 search.
type BM25SearchResult struct {
	// ID is the document identifier.
	ID string
	// Content is the document content.
	Content string
	// Score is the BM25 score (higher is more relevant).
	Score float64
	// Metadata holds optional metadata.
	Metadata map[string]interface{}
	// LineRange is where query terms were found (if computed).
	LineRange *LineRange
}

// LineRange represents a range of lines (1-indexed).
type LineRange struct {
	Start int
	End   int
}

// =============================================================================
// Tokenization
// =============================================================================

// TokenizeOptions configures tokenization behavior.
type TokenizeOptions struct {
	// Lowercase converts to lowercase.
	Lowercase bool
	// RemovePunctuation removes punctuation.
	RemovePunctuation bool
	// MinLength is the minimum token length.
	MinLength int
	// Stopwords are words to remove.
	Stopwords map[string]bool
	// SplitPattern is the custom split pattern.
	SplitPattern *regexp.Regexp
}

// DefaultStopwords is the default set of English stopwords.
var DefaultStopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true,
	"at": true, "be": true, "by": true, "for": true, "from": true,
	"has": true, "he": true, "in": true, "is": true, "it": true,
	"its": true, "of": true, "on": true, "or": true, "that": true,
	"the": true, "to": true, "was": true, "were": true, "will": true,
	"with": true,
}

var defaultSplitPattern = regexp.MustCompile(`\s+`)
var punctuationPattern = regexp.MustCompile(`[^\w\s]`)

// DefaultTokenizeOptions returns the default tokenization options.
func DefaultTokenizeOptions() TokenizeOptions {
	return TokenizeOptions{
		Lowercase:         true,
		RemovePunctuation: true,
		MinLength:         2,
		Stopwords:         DefaultStopwords,
		SplitPattern:      defaultSplitPattern,
	}
}

// Tokenize splits text into an array of terms.
func Tokenize(text string, opts *TokenizeOptions) []string {
	o := DefaultTokenizeOptions()
	if opts != nil {
		if opts.MinLength > 0 {
			o.MinLength = opts.MinLength
		}
		if opts.Stopwords != nil {
			o.Stopwords = opts.Stopwords
		}
		if opts.SplitPattern != nil {
			o.SplitPattern = opts.SplitPattern
		}
		// Only override booleans if explicitly set through a non-default opts
		o.Lowercase = opts.Lowercase
		o.RemovePunctuation = opts.RemovePunctuation
	}

	processed := text

	if o.Lowercase {
		processed = strings.ToLower(processed)
	}

	if o.RemovePunctuation {
		processed = punctuationPattern.ReplaceAllString(processed, " ")
	}

	splitPat := o.SplitPattern
	if splitPat == nil {
		splitPat = defaultSplitPattern
	}

	parts := splitPat.Split(processed, -1)
	var tokens []string
	for _, token := range parts {
		if len(token) < o.MinLength {
			continue
		}
		if o.Stopwords != nil && o.Stopwords[token] {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens
}

// FindLineRange finds the line range where query terms appear in content.
// Returns nil if no terms are found.
func FindLineRange(content string, queryTerms []string, opts *TokenizeOptions) *LineRange {
	if len(queryTerms) == 0 {
		return nil
	}

	lines := strings.Split(content, "\n")

	// Default tokenize options for matching
	matchOpts := &TokenizeOptions{
		Lowercase:         true,
		RemovePunctuation: true,
		MinLength:         2,
	}
	if opts != nil {
		matchOpts = opts
	}

	// Normalize query terms for matching
	normalizedTerms := make(map[string]bool)
	for _, t := range queryTerms {
		if matchOpts.Lowercase {
			t = strings.ToLower(t)
		}
		normalizedTerms[t] = true
	}

	firstMatchLine := -1
	lastMatchLine := -1

	for i, line := range lines {
		lineTokens := Tokenize(line, matchOpts)
		for _, token := range lineTokens {
			if normalizedTerms[token] {
				lineNum := i + 1 // 1-indexed
				if firstMatchLine == -1 {
					firstMatchLine = lineNum
				}
				lastMatchLine = lineNum
				break
			}
		}
	}

	if firstMatchLine != -1 && lastMatchLine != -1 {
		return &LineRange{Start: firstMatchLine, End: lastMatchLine}
	}

	return nil
}

// computeTermFrequencies computes term frequencies for a list of tokens.
func computeTermFrequencies(tokens []string) map[string]int {
	frequencies := make(map[string]int)
	for _, token := range tokens {
		frequencies[token]++
	}
	return frequencies
}

// =============================================================================
// BM25Index
// =============================================================================

// BM25Index is a BM25 index for keyword-based document retrieval.
type BM25Index struct {
	// K1 is the BM25 k1 parameter.
	K1 float64
	// B is the BM25 b parameter.
	B float64

	documents         map[string]*BM25Document
	invertedIndex     map[string]map[string]bool
	documentFrequency map[string]int
	avgDocLength      float64
	docCount          int
	tokenizeOptions   *TokenizeOptions
}

// NewBM25Index creates a new BM25Index.
func NewBM25Index(config *BM25Config, tokenizeOpts *TokenizeOptions) *BM25Index {
	k1 := 1.5
	b := 0.75
	if config != nil {
		if config.K1 != 0 {
			k1 = config.K1
		}
		if config.B != 0 {
			b = config.B
		}
	}
	return &BM25Index{
		K1:                k1,
		B:                 b,
		documents:         make(map[string]*BM25Document),
		invertedIndex:     make(map[string]map[string]bool),
		documentFrequency: make(map[string]int),
		tokenizeOptions:   tokenizeOpts,
	}
}

// Add adds a document to the index.
func (idx *BM25Index) Add(id, content string, metadata map[string]interface{}) {
	// Remove existing document if it exists
	if _, exists := idx.documents[id]; exists {
		idx.Remove(id)
	}

	tokens := Tokenize(content, idx.tokenizeOptions)
	termFreqs := computeTermFrequencies(tokens)

	doc := &BM25Document{
		ID:              id,
		Content:         content,
		Tokens:          tokens,
		TermFrequencies: termFreqs,
		Length:          len(tokens),
		Metadata:        metadata,
	}

	idx.documents[id] = doc
	idx.docCount++

	// Update inverted index and document frequency
	for term := range termFreqs {
		if _, exists := idx.invertedIndex[term]; !exists {
			idx.invertedIndex[term] = make(map[string]bool)
		}
		idx.invertedIndex[term][id] = true
		idx.documentFrequency[term]++
	}

	idx.updateAvgDocLength()
}

// Remove removes a document from the index.
func (idx *BM25Index) Remove(id string) bool {
	doc, exists := idx.documents[id]
	if !exists {
		return false
	}

	// Update inverted index and document frequency
	for term := range doc.TermFrequencies {
		docIDs := idx.invertedIndex[term]
		if docIDs != nil {
			delete(docIDs, id)
			if len(docIDs) == 0 {
				delete(idx.invertedIndex, term)
				delete(idx.documentFrequency, term)
			} else {
				idx.documentFrequency[term]--
			}
		}
	}

	delete(idx.documents, id)
	idx.docCount--
	idx.updateAvgDocLength()

	return true
}

// Clear removes all documents from the index.
func (idx *BM25Index) Clear() {
	idx.documents = make(map[string]*BM25Document)
	idx.invertedIndex = make(map[string]map[string]bool)
	idx.documentFrequency = make(map[string]int)
	idx.docCount = 0
	idx.avgDocLength = 0
}

// Search searches for documents matching the query.
func (idx *BM25Index) Search(query string, topK int, minScore float64) []BM25SearchResult {
	if topK <= 0 {
		topK = 10
	}

	queryTokens := Tokenize(query, idx.tokenizeOptions)
	if len(queryTokens) == 0 || idx.docCount == 0 {
		return nil
	}

	scores := make(map[string]float64)

	for _, queryTerm := range queryTokens {
		docIDs := idx.invertedIndex[queryTerm]
		if docIDs == nil {
			continue
		}

		df := idx.documentFrequency[queryTerm]
		idf := idx.computeIDF(df)

		for docID := range docIDs {
			doc := idx.documents[docID]
			tf := doc.TermFrequencies[queryTerm]
			termScore := idx.computeTermScore(float64(tf), float64(doc.Length), idf)
			scores[docID] += termScore
		}
	}

	var results []BM25SearchResult
	for docID, score := range scores {
		if score >= minScore {
			doc := idx.documents[docID]
			results = append(results, BM25SearchResult{
				ID:       docID,
				Content:  doc.Content,
				Score:    score,
				Metadata: doc.Metadata,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

// Get returns a document by ID.
func (idx *BM25Index) Get(id string) *BM25Document {
	return idx.documents[id]
}

// Has checks if a document exists in the index.
func (idx *BM25Index) Has(id string) bool {
	_, exists := idx.documents[id]
	return exists
}

// Size returns the number of documents in the index.
func (idx *BM25Index) Size() int {
	return idx.docCount
}

// DocumentIDs returns all document IDs.
func (idx *BM25Index) DocumentIDs() []string {
	ids := make([]string, 0, len(idx.documents))
	for id := range idx.documents {
		ids = append(ids, id)
	}
	return ids
}

// =============================================================================
// Serialization
// =============================================================================

// SerializedBM25Document is the serialized document format.
type SerializedBM25Document struct {
	ID              string                 `json:"id"`
	Content         string                 `json:"content"`
	Tokens          []string               `json:"tokens"`
	TermFrequencies map[string]int         `json:"termFrequencies"`
	Length          int                    `json:"length"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// BM25IndexData is the serialized index data.
type BM25IndexData struct {
	K1           float64                  `json:"k1"`
	B            float64                  `json:"b"`
	Documents    []SerializedBM25Document `json:"documents"`
	AvgDocLength float64                  `json:"avgDocLength"`
}

// Serialize serializes the index to a JSON-compatible struct.
func (idx *BM25Index) Serialize() *BM25IndexData {
	docs := make([]SerializedBM25Document, 0, len(idx.documents))
	for id, doc := range idx.documents {
		docs = append(docs, SerializedBM25Document{
			ID:              id,
			Content:         doc.Content,
			Tokens:          doc.Tokens,
			TermFrequencies: doc.TermFrequencies,
			Length:          doc.Length,
			Metadata:        doc.Metadata,
		})
	}

	return &BM25IndexData{
		K1:           idx.K1,
		B:            idx.B,
		Documents:    docs,
		AvgDocLength: idx.avgDocLength,
	}
}

// DeserializeBM25Index rebuilds a BM25Index from serialized data.
func DeserializeBM25Index(data *BM25IndexData, tokenizeOpts *TokenizeOptions) *BM25Index {
	idx := NewBM25Index(&BM25Config{K1: data.K1, B: data.B}, tokenizeOpts)

	for _, sdoc := range data.Documents {
		doc := &BM25Document{
			ID:              sdoc.ID,
			Content:         sdoc.Content,
			Tokens:          sdoc.Tokens,
			TermFrequencies: sdoc.TermFrequencies,
			Length:          sdoc.Length,
			Metadata:        sdoc.Metadata,
		}

		idx.documents[sdoc.ID] = doc
		idx.docCount++

		for term := range sdoc.TermFrequencies {
			if _, exists := idx.invertedIndex[term]; !exists {
				idx.invertedIndex[term] = make(map[string]bool)
			}
			idx.invertedIndex[term][sdoc.ID] = true
			idx.documentFrequency[term]++
		}
	}

	idx.avgDocLength = data.AvgDocLength

	return idx
}

// =============================================================================
// Internal Methods
// =============================================================================

func (idx *BM25Index) updateAvgDocLength() {
	if idx.docCount == 0 {
		idx.avgDocLength = 0
		return
	}

	totalLength := 0
	for _, doc := range idx.documents {
		totalLength += doc.Length
	}
	idx.avgDocLength = float64(totalLength) / float64(idx.docCount)
}

// computeIDF computes IDF (Inverse Document Frequency) for a term.
// Uses the Robertson-Sparck Jones IDF formula.
func (idx *BM25Index) computeIDF(df int) float64 {
	return math.Log((float64(idx.docCount)-float64(df)+0.5)/(float64(df)+0.5) + 1)
}

// computeTermScore computes the BM25 score component for a single term.
func (idx *BM25Index) computeTermScore(tf, docLength, idf float64) float64 {
	numerator := tf * (idx.K1 + 1)
	denominator := tf + idx.K1*(1-idx.B+idx.B*(docLength/idx.avgDocLength))
	return idf * (numerator / denominator)
}

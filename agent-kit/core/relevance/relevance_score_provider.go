// Ported from: packages/core/src/relevance/relevance-score-provider.ts
package relevance

import "fmt"

// RelevanceScoreProvider defines the interface for computing semantic
// relevance between two pieces of text.
type RelevanceScoreProvider interface {
	// GetRelevanceScore returns a score between 0 and 1 indicating how
	// semantically similar text1 and text2 are.
	GetRelevanceScore(text1, text2 string) (float64, error)
}

// CreateSimilarityPrompt builds a prompt string used by providers to evaluate
// the semantic similarity between a query and a text passage.
func CreateSimilarityPrompt(query, text string) string {
	return fmt.Sprintf(`Rate the semantic similarity between the following the query and the text on a scale from 0 to 1 (decimals allowed), where 1 means exactly the same meaning and 0 means completely different:

Query: %s

Text: %s

Relevance score (0-1):`, query, text)
}

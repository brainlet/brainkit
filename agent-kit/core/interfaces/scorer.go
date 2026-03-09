package interfaces

// MastraScorer is the shared interface for scorer instances, used by the
// mastra package to interact with scorers without importing the evals package.
//
// This breaks the circular dependency: mastra/hooks.go imports evals (for
// ScoringHookInput, SaveScorePayload), and evals.MastraScorer needs to
// register with mastra. By extracting the scorer interface here, both
// packages can reference it without importing each other.
//
// The real evals.MastraScorer struct satisfies this interface.
type MastraScorer interface {
	// ID returns the scorer's unique identifier.
	ID() string
	// Name returns the scorer's display name (falls back to ID).
	Name() string
	// Source returns the scorer's source ("code" or "stored").
	Source() string
	// SetSource sets the scorer's source.
	SetSource(s string)
	// RegisterMastra registers the Mastra instance with the scorer.
	// Accepts *mastra.Mastra typed as any to avoid import.
	RegisterMastra(m any)
}

// ScorerEntry pairs a scorer with metadata, used in agent/workflow scorer
// listings. Defined here so both mastra and agent packages can reference it
// without circular imports.
type ScorerEntry struct {
	Scorer MastraScorer
}

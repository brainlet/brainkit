// Ported from: packages/core/src/observability/types/scores.ts
package types

import "time"

// ============================================================================
// ScoreInput (User Input)
// ============================================================================

// ScoreInput holds user-provided score data for evaluating span/trace quality.
type ScoreInput struct {
	// ScorerName is the name of the scorer (e.g., "relevance", "accuracy", "toxicity").
	ScorerName string `json:"scorerName"`
	// Score is the numeric score value (typically 0-1 or 0-100).
	Score float64 `json:"score"`
	// Reason is a human-readable explanation of the score.
	Reason string `json:"reason,omitempty"`
	// ExperimentID is the experiment identifier for A/B testing or evaluation runs.
	ExperimentID string `json:"experimentId,omitempty"`
	// Metadata is additional metadata specific to this score.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ============================================================================
// ExportedScore (Event Bus Transport)
// ============================================================================

// ExportedScore is score data transported via the event bus.
// Must be JSON-serializable.
type ExportedScore struct {
	// Timestamp is when the score was recorded.
	Timestamp time.Time `json:"timestamp"`
	// TraceID is the trace being scored.
	TraceID string `json:"traceId"`
	// SpanID is the specific span being scored (empty = trace-level score).
	SpanID string `json:"spanId,omitempty"`
	// ScorerName is the name of the scorer.
	ScorerName string `json:"scorerName"`
	// Score is the numeric score value.
	Score float64 `json:"score"`
	// Reason is the human-readable explanation.
	Reason string `json:"reason,omitempty"`
	// ExperimentID is the experiment identifier for A/B testing.
	ExperimentID string `json:"experimentId,omitempty"`
	// Metadata is user-defined metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ============================================================================
// ScoreEvent (Event Bus Event)
// ============================================================================

// ScoreEvent is a score event emitted to the ObservabilityBus.
type ScoreEvent struct {
	Type  string        `json:"type"` // always "score"
	Score ExportedScore `json:"score"`
}

// NewScoreEvent creates a new ScoreEvent with the type set to "score".
func NewScoreEvent(score ExportedScore) ScoreEvent {
	return ScoreEvent{
		Type:  "score",
		Score: score,
	}
}

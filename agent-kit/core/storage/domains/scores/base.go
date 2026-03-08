// Ported from: packages/core/src/storage/domains/scores/base.ts
package scores

import (
	"context"

	"github.com/brainlet/brainkit/agent-kit/core/evals"
	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Types — canonical imports from evals/types.go
// ---------------------------------------------------------------------------

// ScoringSource identifies the origin of a score (e.g. "LIVE", "TEST").
type ScoringSource = evals.ScoringSource

// ScoreRowData represents a stored score record.
type ScoreRowData = evals.ScoreRowData

// SaveScorePayload is the input for saving a score.
type SaveScorePayload = evals.SaveScorePayload

// ListScoresResponse is the response for listing scores.
type ListScoresResponse = evals.ListScoresResponse

// ---------------------------------------------------------------------------
// ScoresStorage Interface
// ---------------------------------------------------------------------------

// ScoresStorage is the storage interface for the scores domain.
type ScoresStorage interface {
	// Init initializes the storage domain.
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// GetScoreByID retrieves a score by ID.
	GetScoreByID(ctx context.Context, id string) (*ScoreRowData, error)

	// SaveScore saves a new score.
	SaveScore(ctx context.Context, score SaveScorePayload) (*ScoreRowData, error)

	// ListScoresByScorerID lists scores for a specific scorer.
	ListScoresByScorerID(ctx context.Context, args ListScoresByScorerIDArgs) (*ListScoresResponse, error)

	// ListScoresByRunID lists scores for a specific run.
	ListScoresByRunID(ctx context.Context, args ListScoresByRunIDArgs) (*ListScoresResponse, error)

	// ListScoresByEntityID lists scores for a specific entity.
	ListScoresByEntityID(ctx context.Context, args ListScoresByEntityIDArgs) (*ListScoresResponse, error)

	// ListScoresBySpan lists scores for a specific trace/span pair.
	ListScoresBySpan(ctx context.Context, args ListScoresBySpanArgs) (*ListScoresResponse, error)
}

// ListScoresByScorerIDArgs holds the arguments for ListScoresByScorerID.
type ListScoresByScorerIDArgs struct {
	ScorerID   string                    `json:"scorerId"`
	Pagination domains.PaginationArgs    `json:"pagination"`
	EntityID   string                    `json:"entityId,omitempty"`
	EntityType evals.ScoringEntityType   `json:"entityType,omitempty"`
	Source     ScoringSource             `json:"source,omitempty"`
}

// ListScoresByRunIDArgs holds the arguments for ListScoresByRunID.
type ListScoresByRunIDArgs struct {
	RunID      string               `json:"runId"`
	Pagination domains.PaginationArgs `json:"pagination"`
}

// ListScoresByEntityIDArgs holds the arguments for ListScoresByEntityID.
type ListScoresByEntityIDArgs struct {
	EntityID   string                    `json:"entityId"`
	EntityType evals.ScoringEntityType   `json:"entityType"`
	Pagination domains.PaginationArgs    `json:"pagination"`
}

// ListScoresBySpanArgs holds the arguments for ListScoresBySpan.
type ListScoresBySpanArgs struct {
	TraceID    string               `json:"traceId"`
	SpanID     string               `json:"spanId"`
	Pagination domains.PaginationArgs `json:"pagination"`
}

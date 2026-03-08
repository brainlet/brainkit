// Ported from: packages/core/src/storage/domains/scores/inmemory.ts
package scores

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// Compile-time interface check.
var _ ScoresStorage = (*InMemoryScoresStorage)(nil)

// InMemoryScoresStorage is an in-memory implementation of ScoresStorage.
type InMemoryScoresStorage struct {
	mu     sync.RWMutex
	scores map[string]ScoreRowData
}

// NewInMemoryScoresStorage creates a new InMemoryScoresStorage.
func NewInMemoryScoresStorage() *InMemoryScoresStorage {
	return &InMemoryScoresStorage{
		scores: make(map[string]ScoreRowData),
	}
}

// Init is a no-op for the in-memory store.
func (s *InMemoryScoresStorage) Init(_ context.Context) error {
	return nil
}

// DangerouslyClearAll clears all scores.
func (s *InMemoryScoresStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scores = make(map[string]ScoreRowData)
	return nil
}

// GetScoreByID retrieves a score by ID. Returns nil if not found.
func (s *InMemoryScoresStorage) GetScoreByID(_ context.Context, id string) (*ScoreRowData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	score, ok := s.scores[id]
	if !ok {
		return nil, nil
	}
	return &score, nil
}

// SaveScore saves a new score and returns the created record.
func (s *InMemoryScoresStorage) SaveScore(_ context.Context, payload SaveScorePayload) (*ScoreRowData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	score := ScoreRowData{
		ID:       uuid.New().String(),
		ScorerID: payload.ScorerID,
		EntityID: payload.EntityID,

		// From ScoringInputWithExtractStepResultAndScoreAndReason
		RunID:             payload.RunID,
		Input:             payload.Input,
		Output:            payload.Output,
		AdditionalContext: payload.AdditionalContext,
		RequestContext:    payload.RequestContext,
		ExtractStepResult: payload.ExtractStepResult,
		ExtractPrompt:     payload.ExtractPrompt,
		Score:             payload.Score,
		AnalyzeStepResult: payload.AnalyzeStepResult,
		AnalyzePrompt:     payload.AnalyzePrompt,
		Reason:            payload.Reason,
		ReasonPrompt:      payload.ReasonPrompt,

		// From ScoringHookInput
		Scorer:           payload.Scorer,
		Metadata:         payload.Metadata,
		Source:           payload.Source,
		Entity:           payload.Entity,
		EntityType:       payload.EntityType,
		StructuredOutput: payload.StructuredOutput,
		TraceID:          payload.TraceID,
		SpanID:           payload.SpanID,
		ResourceID:       payload.ResourceID,
		ThreadID:         payload.ThreadID,

		// Additional ScoreRowData fields
		PreprocessStepResult: payload.PreprocessStepResult,
		PreprocessPrompt:     payload.PreprocessPrompt,
		GenerateScorePrompt:  payload.GenerateScorePrompt,
		GenerateReasonPrompt: payload.GenerateReasonPrompt,

		// Timestamps
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.scores[score.ID] = score
	return &score, nil
}

// ListScoresByScorerID lists scores for a specific scorer with optional filters.
func (s *InMemoryScoresStorage) ListScoresByScorerID(_ context.Context, args ListScoresByScorerIDArgs) (*ListScoresResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []ScoreRowData
	for _, score := range s.scores {
		if score.ScorerID != args.ScorerID {
			continue
		}
		if args.EntityID != "" && score.EntityID != args.EntityID {
			continue
		}
		if args.EntityType != "" && score.EntityType != args.EntityType {
			continue
		}
		if args.Source != "" && score.Source != args.Source {
			continue
		}
		filtered = append(filtered, score)
	}

	return paginateScores(filtered, args.Pagination), nil
}

// ListScoresByRunID lists scores for a specific run.
func (s *InMemoryScoresStorage) ListScoresByRunID(_ context.Context, args ListScoresByRunIDArgs) (*ListScoresResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []ScoreRowData
	for _, score := range s.scores {
		if score.RunID == args.RunID {
			filtered = append(filtered, score)
		}
	}

	return paginateScores(filtered, args.Pagination), nil
}

// ListScoresByEntityID lists scores for a specific entity.
func (s *InMemoryScoresStorage) ListScoresByEntityID(_ context.Context, args ListScoresByEntityIDArgs) (*ListScoresResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []ScoreRowData
	for _, score := range s.scores {
		if score.EntityID == args.EntityID && score.EntityType == args.EntityType {
			filtered = append(filtered, score)
		}
	}

	return paginateScores(filtered, args.Pagination), nil
}

// ListScoresBySpan lists scores for a specific trace/span pair.
func (s *InMemoryScoresStorage) ListScoresBySpan(_ context.Context, args ListScoresBySpanArgs) (*ListScoresResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []ScoreRowData
	for _, score := range s.scores {
		if score.TraceID == args.TraceID && score.SpanID == args.SpanID {
			filtered = append(filtered, score)
		}
	}

	// Sort by createdAt descending (newest first).
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[j].CreatedAt.Before(filtered[i].CreatedAt)
	})

	return paginateScores(filtered, args.Pagination), nil
}

// paginateScores applies pagination to a slice of scores.
// Mirrors the TS normalizePerPage + calculatePagination logic.
func paginateScores(scores []ScoreRowData, pagination domains.PaginationArgs) *ListScoresResponse {
	if scores == nil {
		scores = []ScoreRowData{}
	}

	total := len(scores)
	page := pagination.Page

	// PerPage <= 0 means "all" (TS uses false → MAX_SAFE_INTEGER).
	perPage := pagination.PerPage
	noPagination := perPage <= 0

	if noPagination {
		perPage = math.MaxInt
	}

	start := page * perPage
	end := start + perPage
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	responsePerPage := perPage
	if noPagination {
		responsePerPage = domains.PerPageDisabled
	}

	hasMore := false
	if !noPagination {
		hasMore = total > end
	}

	return &ListScoresResponse{
		Scores: scores[start:end],
		Pagination: domains.PaginationInfo{
			Total:   total,
			Page:    page,
			PerPage: responsePerPage,
			HasMore: hasMore,
		},
	}
}

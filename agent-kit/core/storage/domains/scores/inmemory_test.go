// Ported from: stores/_test-utils/src/domains/scores/index.ts (createScoresTest)
// and: packages/server/src/server/handlers/scores.test.ts (handler-level tests adapted to storage-level)
//
// The upstream mastra project has no dedicated scores storage test file at
// packages/core/src/storage/domains/scores/scores.test.ts — the canonical
// storage-level tests live in stores/_test-utils/src/domains/scores/index.ts
// and are re-used by each storage adapter. This Go file faithfully ports
// those tests against InMemoryScoresStorage.
package scores

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/evals"
	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// createSampleScore mirrors stores/_test-utils/src/domains/scores/data.ts
// ---------------------------------------------------------------------------

// createSampleScoreOpts holds optional overrides for createSampleScore.
type createSampleScoreOpts struct {
	ScorerID   string
	EntityID   string
	EntityType evals.ScoringEntityType
	Source     evals.ScoringSource
	TraceID    string
	SpanID     string
}

// createSampleScore builds a SaveScorePayload matching the TS helper.
// Fields that are not overridden use the same defaults as the TS version.
func createSampleScore(opts createSampleScoreOpts) SaveScorePayload {
	entityID := opts.EntityID
	if entityID == "" {
		entityID = "eval-agent"
	}
	entityType := opts.EntityType
	if entityType == "" {
		entityType = evals.ScoringEntityTypeAgent // "AGENT"
	}
	source := opts.Source
	if source == "" {
		source = evals.ScoringSourceLive // "LIVE"
	}

	return SaveScorePayload{
		ScorerID:   opts.ScorerID,
		EntityID:   entityID,
		EntityType: entityType,
		RunID:      uuid.New().String(),
		Reason:     "Sample reason",
		PreprocessStepResult: map[string]any{
			"text": "Sample preprocess step result",
		},
		PreprocessPrompt: "Sample preprocess prompt",
		AnalyzeStepResult: map[string]any{
			"text": "Sample analyze step result",
		},
		Score:                0.8,
		AnalyzePrompt:       "Sample analyze prompt",
		GenerateReasonPrompt: "Sample reason prompt",
		Scorer: map[string]any{
			"id":          opts.ScorerID,
			"name":        "my-eval",
			"description": "My eval",
		},
		Input: []any{
			map[string]any{
				"id":    uuid.New().String(),
				"name":  "input-1",
				"value": "Sample input",
			},
		},
		Output: map[string]any{
			"text": "Sample output",
		},
		Source: source,
		Entity: map[string]any{
			"id":   entityID,
			"name": "Sample entity",
		},
		RequestContext: map[string]any{},
		Metadata: map[string]any{
			"scorerVersion": "1.0.0",
			"customField":   "test-value",
		},
		TraceID: opts.TraceID,
		SpanID:  opts.SpanID,
	}
}

// ---------------------------------------------------------------------------
// createScores is a bulk helper matching the TS createScores function.
// It saves multiple scores and returns the saved ScoreRowData slice.
// ---------------------------------------------------------------------------

type scoreConfig struct {
	Count    int
	ScorerID string
	TraceID  string
	SpanID   string
}

func createScores(ctx context.Context, t *testing.T, storage *InMemoryScoresStorage, configs []scoreConfig) []ScoreRowData {
	t.Helper()
	var all []ScoreRowData
	for _, cfg := range configs {
		for i := 0; i < cfg.Count; i++ {
			payload := createSampleScore(createSampleScoreOpts{
				ScorerID: cfg.ScorerID,
				TraceID:  cfg.TraceID,
				SpanID:   cfg.SpanID,
			})
			saved, err := storage.SaveScore(ctx, payload)
			if err != nil {
				t.Fatalf("SaveScore failed: %v", err)
			}
			all = append(all, *saved)
		}
	}
	return all
}

// ===========================================================================
// Tests — ported from stores/_test-utils/src/domains/scores/index.ts
// ===========================================================================

func TestInMemoryScoresStorage_SaveAndRetrieveByScorerID(t *testing.T) {
	// TS: "should retrieve scores by scorer id"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()

	score1 := createSampleScore(createSampleScoreOpts{ScorerID: scorerID})
	score2 := createSampleScore(createSampleScoreOpts{ScorerID: scorerID})
	score3 := createSampleScore(createSampleScoreOpts{ScorerID: scorerID})

	saved1, err := storage.SaveScore(ctx, score1)
	if err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}
	saved2, err := storage.SaveScore(ctx, score2)
	if err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}
	saved3, err := storage.SaveScore(ctx, score3)
	if err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}

	// Retrieve all scores for the scorer.
	result, err := storage.ListScoresByScorerID(ctx, ListScoresByScorerIDArgs{
		ScorerID:   scorerID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresByScorerID returned error: %v", err)
	}
	if len(result.Scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(result.Scores))
	}

	// Verify that all saved runIds are present in the result.
	runIDs := map[string]bool{
		saved1.RunID: false,
		saved2.RunID: false,
		saved3.RunID: false,
	}
	for _, s := range result.Scores {
		runIDs[s.RunID] = true
	}
	for rid, found := range runIDs {
		if !found {
			t.Errorf("expected runId %s to be present in results", rid)
		}
	}

	// Non-existent scorer should return empty.
	nonExistent, err := storage.ListScoresByScorerID(ctx, ListScoresByScorerIDArgs{
		ScorerID:   "non-existent-scorer",
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresByScorerID returned error: %v", err)
	}
	if len(nonExistent.Scores) != 0 {
		t.Errorf("expected 0 scores for non-existent scorer, got %d", len(nonExistent.Scores))
	}
}

func TestInMemoryScoresStorage_ScorePayloadMatches(t *testing.T) {
	// TS: "should return score payload matching the saved score"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	payload := createSampleScore(createSampleScoreOpts{ScorerID: scorerID})

	_, err := storage.SaveScore(ctx, payload)
	if err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}

	result, err := storage.ListScoresByScorerID(ctx, ListScoresByScorerIDArgs{
		ScorerID:   scorerID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresByScorerID returned error: %v", err)
	}
	if len(result.Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(result.Scores))
	}

	returned := result.Scores[0]

	// Verify key fields match the payload (mirrors TS normalizeScore comparison).
	if returned.ScorerID != payload.ScorerID {
		t.Errorf("scorerId mismatch: got %s, want %s", returned.ScorerID, payload.ScorerID)
	}
	if returned.EntityID != payload.EntityID {
		t.Errorf("entityId mismatch: got %s, want %s", returned.EntityID, payload.EntityID)
	}
	if returned.RunID != payload.RunID {
		t.Errorf("runId mismatch: got %s, want %s", returned.RunID, payload.RunID)
	}
	if returned.Score != payload.Score {
		t.Errorf("score mismatch: got %f, want %f", returned.Score, payload.Score)
	}
	if returned.Reason != payload.Reason {
		t.Errorf("reason mismatch: got %s, want %s", returned.Reason, payload.Reason)
	}
	if string(returned.Source) != string(payload.Source) {
		t.Errorf("source mismatch: got %s, want %s", returned.Source, payload.Source)
	}
	if string(returned.EntityType) != string(payload.EntityType) {
		t.Errorf("entityType mismatch: got %s, want %s", returned.EntityType, payload.EntityType)
	}
	if returned.AnalyzePrompt != payload.AnalyzePrompt {
		t.Errorf("analyzePrompt mismatch: got %s, want %s", returned.AnalyzePrompt, payload.AnalyzePrompt)
	}
	if returned.PreprocessPrompt != payload.PreprocessPrompt {
		t.Errorf("preprocessPrompt mismatch: got %s, want %s", returned.PreprocessPrompt, payload.PreprocessPrompt)
	}
	if returned.GenerateReasonPrompt != payload.GenerateReasonPrompt {
		t.Errorf("generateReasonPrompt mismatch: got %s, want %s", returned.GenerateReasonPrompt, payload.GenerateReasonPrompt)
	}

	// ID and timestamps should be populated by SaveScore.
	if returned.ID == "" {
		t.Error("expected ID to be set")
	}
	if returned.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if returned.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestInMemoryScoresStorage_RetrieveBySource(t *testing.T) {
	// TS: "should retrieve scores by source"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	score1 := createSampleScore(createSampleScoreOpts{ScorerID: scorerID, Source: evals.ScoringSourceTest})
	score2 := createSampleScore(createSampleScoreOpts{ScorerID: scorerID, Source: evals.ScoringSourceLive})

	if _, err := storage.SaveScore(ctx, score1); err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}
	if _, err := storage.SaveScore(ctx, score2); err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}

	result, err := storage.ListScoresByScorerID(ctx, ListScoresByScorerIDArgs{
		ScorerID:   scorerID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
		Source:     evals.ScoringSourceTest,
	})
	if err != nil {
		t.Fatalf("ListScoresByScorerID returned error: %v", err)
	}
	if len(result.Scores) != 1 {
		t.Fatalf("expected 1 score with source=TEST, got %d", len(result.Scores))
	}
	if result.Scores[0].Source != evals.ScoringSourceTest {
		t.Errorf("expected source=TEST, got %s", result.Scores[0].Source)
	}
}

func TestInMemoryScoresStorage_SaveScoreAndListByRunID(t *testing.T) {
	// TS: "should save scorer" (saves and retrieves by runId)
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	payload := createSampleScore(createSampleScoreOpts{ScorerID: scorerID})

	if _, err := storage.SaveScore(ctx, payload); err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}

	result, err := storage.ListScoresByRunID(ctx, ListScoresByRunIDArgs{
		RunID:      payload.RunID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresByRunID returned error: %v", err)
	}
	if len(result.Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(result.Scores))
	}
	if result.Pagination.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Pagination.Total)
	}
	if result.Pagination.Page != 0 {
		t.Errorf("expected page=0, got %d", result.Pagination.Page)
	}
	if result.Pagination.PerPage != 10 {
		t.Errorf("expected perPage=10, got %d", result.Pagination.PerPage)
	}
	if result.Pagination.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestInMemoryScoresStorage_GetScoreByID(t *testing.T) {
	// TS: "should retrieve saved score by its returned id"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	payload := createSampleScore(createSampleScoreOpts{ScorerID: scorerID})

	saved, err := storage.SaveScore(ctx, payload)
	if err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected saved score to have an ID")
	}

	retrieved, err := storage.GetScoreByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetScoreByID returned error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected retrieved score to not be nil")
	}
	if retrieved.ID != saved.ID {
		t.Errorf("expected id=%s, got %s", saved.ID, retrieved.ID)
	}
	if retrieved.ScorerID != scorerID {
		t.Errorf("expected scorerId=%s, got %s", scorerID, retrieved.ScorerID)
	}
	if retrieved.RunID != payload.RunID {
		t.Errorf("expected runId=%s, got %s", payload.RunID, retrieved.RunID)
	}
}

func TestInMemoryScoresStorage_GetScoreByID_NotFound(t *testing.T) {
	// Edge case: retrieving a score that does not exist returns nil.
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	result, err := storage.GetScoreByID(ctx, "nonexistent-id")
	if err != nil {
		t.Fatalf("GetScoreByID returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for nonexistent score, got %+v", result)
	}
}

func TestInMemoryScoresStorage_ListScoresByEntityID(t *testing.T) {
	// TS: "listScoresByEntityId should return paginated scores with total count"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	payload := createSampleScore(createSampleScoreOpts{ScorerID: scorerID})

	if _, err := storage.SaveScore(ctx, payload); err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}

	// The entity ID comes from the payload.Entity map and payload.EntityID.
	result, err := storage.ListScoresByEntityID(ctx, ListScoresByEntityIDArgs{
		EntityID:   payload.EntityID,
		EntityType: payload.EntityType,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresByEntityID returned error: %v", err)
	}
	if len(result.Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(result.Scores))
	}
	if result.Pagination.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Pagination.Total)
	}
	if result.Pagination.Page != 0 {
		t.Errorf("expected page=0, got %d", result.Pagination.Page)
	}
	if result.Pagination.PerPage != 10 {
		t.Errorf("expected perPage=10, got %d", result.Pagination.PerPage)
	}
	if result.Pagination.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestInMemoryScoresStorage_ListScoresByEntityID_EmptyResult(t *testing.T) {
	// Edge case: non-existent entity returns empty list.
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	result, err := storage.ListScoresByEntityID(ctx, ListScoresByEntityIDArgs{
		EntityID:   "no-such-entity",
		EntityType: evals.ScoringEntityTypeAgent,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresByEntityID returned error: %v", err)
	}
	if len(result.Scores) != 0 {
		t.Errorf("expected 0 scores, got %d", len(result.Scores))
	}
	if result.Pagination.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Pagination.Total)
	}
}

func TestInMemoryScoresStorage_ListScoresBySpan_Single(t *testing.T) {
	// TS: "should retrieve scores by trace and span id"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	traceID := uuid.New().String()
	spanID := uuid.New().String()

	payload := createSampleScore(createSampleScoreOpts{
		ScorerID: scorerID,
		TraceID:  traceID,
		SpanID:   spanID,
	})
	if _, err := storage.SaveScore(ctx, payload); err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}

	result, err := storage.ListScoresBySpan(ctx, ListScoresBySpanArgs{
		TraceID:    traceID,
		SpanID:     spanID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresBySpan returned error: %v", err)
	}
	if len(result.Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(result.Scores))
	}
	if result.Scores[0].TraceID != traceID {
		t.Errorf("expected traceId=%s, got %s", traceID, result.Scores[0].TraceID)
	}
	if result.Scores[0].SpanID != spanID {
		t.Errorf("expected spanId=%s, got %s", spanID, result.Scores[0].SpanID)
	}
	if result.Pagination.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Pagination.Total)
	}
	if result.Pagination.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestInMemoryScoresStorage_ListScoresBySpan_Multiple(t *testing.T) {
	// TS: "should retrieve multiple scores by trace and span id"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	traceID := uuid.New().String()
	spanID := uuid.New().String()

	for i := 0; i < 3; i++ {
		payload := createSampleScore(createSampleScoreOpts{
			ScorerID: scorerID,
			TraceID:  traceID,
			SpanID:   spanID,
		})
		if _, err := storage.SaveScore(ctx, payload); err != nil {
			t.Fatalf("SaveScore returned error: %v", err)
		}
	}

	result, err := storage.ListScoresBySpan(ctx, ListScoresBySpanArgs{
		TraceID:    traceID,
		SpanID:     spanID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresBySpan returned error: %v", err)
	}
	if len(result.Scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(result.Scores))
	}
	for _, s := range result.Scores {
		if s.TraceID != traceID {
			t.Errorf("expected traceId=%s, got %s", traceID, s.TraceID)
		}
		if s.SpanID != spanID {
			t.Errorf("expected spanId=%s, got %s", spanID, s.SpanID)
		}
	}
	if result.Pagination.Total != 3 {
		t.Errorf("expected total=3, got %d", result.Pagination.Total)
	}
	if result.Pagination.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestInMemoryScoresStorage_ListScoresBySpan_FirstPagePagination(t *testing.T) {
	// TS: "should handle first page pagination correctly"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	traceID := uuid.New().String()
	spanID := uuid.New().String()

	// Create 5 target scores + 3 decoy scores with different trace/span combos.
	createScores(ctx, t, storage, []scoreConfig{
		{Count: 5, ScorerID: scorerID, TraceID: traceID, SpanID: spanID},              // target
		{Count: 1, ScorerID: scorerID, TraceID: uuid.New().String(), SpanID: spanID},   // different trace
		{Count: 1, ScorerID: scorerID, TraceID: traceID, SpanID: uuid.New().String()},  // different span
		{Count: 1, ScorerID: scorerID, TraceID: uuid.New().String(), SpanID: uuid.New().String()}, // both different
	})

	firstPage, err := storage.ListScoresBySpan(ctx, ListScoresBySpanArgs{
		TraceID:    traceID,
		SpanID:     spanID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 2},
	})
	if err != nil {
		t.Fatalf("ListScoresBySpan returned error: %v", err)
	}
	if len(firstPage.Scores) != 2 {
		t.Fatalf("expected 2 scores on first page, got %d", len(firstPage.Scores))
	}
	if firstPage.Pagination.Total != 5 {
		t.Errorf("expected total=5, got %d", firstPage.Pagination.Total)
	}
	if firstPage.Pagination.Page != 0 {
		t.Errorf("expected page=0, got %d", firstPage.Pagination.Page)
	}
	if firstPage.Pagination.PerPage != 2 {
		t.Errorf("expected perPage=2, got %d", firstPage.Pagination.PerPage)
	}
	if !firstPage.Pagination.HasMore {
		t.Error("expected hasMore=true")
	}
	for _, s := range firstPage.Scores {
		if s.TraceID != traceID || s.SpanID != spanID {
			t.Errorf("expected traceId=%s spanId=%s, got traceId=%s spanId=%s", traceID, spanID, s.TraceID, s.SpanID)
		}
	}
}

func TestInMemoryScoresStorage_ListScoresBySpan_MiddlePagePagination(t *testing.T) {
	// TS: "should handle middle page pagination correctly"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	traceID := uuid.New().String()
	spanID := uuid.New().String()

	createScores(ctx, t, storage, []scoreConfig{
		{Count: 5, ScorerID: scorerID, TraceID: traceID, SpanID: spanID},
		{Count: 1, ScorerID: scorerID, TraceID: uuid.New().String(), SpanID: spanID},
		{Count: 1, ScorerID: scorerID, TraceID: traceID, SpanID: uuid.New().String()},
		{Count: 1, ScorerID: scorerID, TraceID: uuid.New().String(), SpanID: uuid.New().String()},
	})

	secondPage, err := storage.ListScoresBySpan(ctx, ListScoresBySpanArgs{
		TraceID:    traceID,
		SpanID:     spanID,
		Pagination: domains.PaginationArgs{Page: 1, PerPage: 2},
	})
	if err != nil {
		t.Fatalf("ListScoresBySpan returned error: %v", err)
	}
	if len(secondPage.Scores) != 2 {
		t.Fatalf("expected 2 scores on second page, got %d", len(secondPage.Scores))
	}
	if secondPage.Pagination.Total != 5 {
		t.Errorf("expected total=5, got %d", secondPage.Pagination.Total)
	}
	if secondPage.Pagination.Page != 1 {
		t.Errorf("expected page=1, got %d", secondPage.Pagination.Page)
	}
	if secondPage.Pagination.PerPage != 2 {
		t.Errorf("expected perPage=2, got %d", secondPage.Pagination.PerPage)
	}
	if !secondPage.Pagination.HasMore {
		t.Error("expected hasMore=true")
	}
	for _, s := range secondPage.Scores {
		if s.TraceID != traceID || s.SpanID != spanID {
			t.Errorf("expected traceId=%s spanId=%s, got traceId=%s spanId=%s", traceID, spanID, s.TraceID, s.SpanID)
		}
	}
}

func TestInMemoryScoresStorage_ListScoresBySpan_LastPagePagination(t *testing.T) {
	// TS: "should handle last page pagination"
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	traceID := uuid.New().String()
	spanID := uuid.New().String()

	otherTraceID1 := uuid.New().String()
	otherTraceID2 := uuid.New().String()
	otherSpanID1 := uuid.New().String()
	otherSpanID2 := uuid.New().String()

	createScores(ctx, t, storage, []scoreConfig{
		{Count: 5, ScorerID: scorerID, TraceID: traceID, SpanID: spanID},            // target
		{Count: 1, ScorerID: scorerID, TraceID: otherTraceID1, SpanID: spanID},       // different trace, same span
		{Count: 1, ScorerID: scorerID, TraceID: traceID, SpanID: otherSpanID1},       // same trace, different span
		{Count: 1, ScorerID: scorerID, TraceID: otherTraceID2, SpanID: otherSpanID2}, // both different
		{Count: 1, ScorerID: scorerID, TraceID: otherTraceID1, SpanID: spanID},       // different trace, same span (again)
		{Count: 1, ScorerID: scorerID, TraceID: traceID, SpanID: otherSpanID2},       // same trace, different span (again)
	})

	// First page
	firstPage, err := storage.ListScoresBySpan(ctx, ListScoresBySpanArgs{
		TraceID:    traceID,
		SpanID:     spanID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 2},
	})
	if err != nil {
		t.Fatalf("ListScoresBySpan (first page) returned error: %v", err)
	}

	// Second page
	secondPage, err := storage.ListScoresBySpan(ctx, ListScoresBySpanArgs{
		TraceID:    traceID,
		SpanID:     spanID,
		Pagination: domains.PaginationArgs{Page: 1, PerPage: 2},
	})
	if err != nil {
		t.Fatalf("ListScoresBySpan (second page) returned error: %v", err)
	}

	// Last page
	lastPage, err := storage.ListScoresBySpan(ctx, ListScoresBySpanArgs{
		TraceID:    traceID,
		SpanID:     spanID,
		Pagination: domains.PaginationArgs{Page: 2, PerPage: 2},
	})
	if err != nil {
		t.Fatalf("ListScoresBySpan (last page) returned error: %v", err)
	}

	if len(lastPage.Scores) != 1 {
		t.Fatalf("expected 1 score on last page, got %d", len(lastPage.Scores))
	}
	if lastPage.Pagination.Total != 5 {
		t.Errorf("expected total=5, got %d", lastPage.Pagination.Total)
	}
	if lastPage.Pagination.Page != 2 {
		t.Errorf("expected page=2, got %d", lastPage.Pagination.Page)
	}
	if lastPage.Pagination.PerPage != 2 {
		t.Errorf("expected perPage=2, got %d", lastPage.Pagination.PerPage)
	}
	if lastPage.Pagination.HasMore {
		t.Error("expected hasMore=false on last page")
	}

	// All pages should only contain scores with the target traceId and spanId.
	allPages := []*ListScoresResponse{firstPage, secondPage, lastPage}
	for pi, page := range allPages {
		for _, s := range page.Scores {
			if s.TraceID != traceID || s.SpanID != spanID {
				t.Errorf("page %d: expected traceId=%s spanId=%s, got traceId=%s spanId=%s",
					pi, traceID, spanID, s.TraceID, s.SpanID)
			}
		}
	}
}

func TestInMemoryScoresStorage_ListScoresByRunID_EmptyResult(t *testing.T) {
	// Edge case: non-existent runId returns empty list with correct pagination.
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	result, err := storage.ListScoresByRunID(ctx, ListScoresByRunIDArgs{
		RunID:      "nonexistent-run",
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresByRunID returned error: %v", err)
	}
	if len(result.Scores) != 0 {
		t.Errorf("expected 0 scores, got %d", len(result.Scores))
	}
	if result.Pagination.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Pagination.Total)
	}
	if result.Pagination.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestInMemoryScoresStorage_ListScoresBySpan_EmptyResult(t *testing.T) {
	// Edge case: non-existent trace/span returns empty list.
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	result, err := storage.ListScoresBySpan(ctx, ListScoresBySpanArgs{
		TraceID:    "nonexistent-trace",
		SpanID:     "nonexistent-span",
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListScoresBySpan returned error: %v", err)
	}
	if len(result.Scores) != 0 {
		t.Errorf("expected 0 scores, got %d", len(result.Scores))
	}
	if result.Pagination.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Pagination.Total)
	}
}

func TestInMemoryScoresStorage_DangerouslyClearAll(t *testing.T) {
	// TS pattern from datasets tests — verify clear removes all data.
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	// Save two scores.
	s1 := createSampleScore(createSampleScoreOpts{ScorerID: "clear-1"})
	s2 := createSampleScore(createSampleScoreOpts{ScorerID: "clear-2"})
	saved1, _ := storage.SaveScore(ctx, s1)
	saved2, _ := storage.SaveScore(ctx, s2)

	// Verify they exist.
	r1, _ := storage.GetScoreByID(ctx, saved1.ID)
	r2, _ := storage.GetScoreByID(ctx, saved2.ID)
	if r1 == nil || r2 == nil {
		t.Fatal("expected both scores to exist before clear")
	}

	// Clear all.
	if err := storage.DangerouslyClearAll(ctx); err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	// Verify they are gone.
	r1, _ = storage.GetScoreByID(ctx, saved1.ID)
	r2, _ = storage.GetScoreByID(ctx, saved2.ID)
	if r1 != nil {
		t.Error("expected first score to be cleared")
	}
	if r2 != nil {
		t.Error("expected second score to be cleared")
	}
}

func TestInMemoryScoresStorage_Init(t *testing.T) {
	// Init is a no-op for in-memory; just verify it doesn't error.
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	if err := storage.Init(ctx); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
}

func TestInMemoryScoresStorage_ListScoresByScorerID_WithEntityFilter(t *testing.T) {
	// TS ListScoresByScorerID supports optional entityId and entityType filters.
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()

	// Score with entityType=AGENT
	agentScore := createSampleScore(createSampleScoreOpts{
		ScorerID:   scorerID,
		EntityID:   "agent-1",
		EntityType: evals.ScoringEntityTypeAgent,
	})
	// Score with entityType=WORKFLOW
	workflowScore := createSampleScore(createSampleScoreOpts{
		ScorerID:   scorerID,
		EntityID:   "workflow-1",
		EntityType: evals.ScoringEntityTypeWorkflow,
	})

	if _, err := storage.SaveScore(ctx, agentScore); err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}
	if _, err := storage.SaveScore(ctx, workflowScore); err != nil {
		t.Fatalf("SaveScore returned error: %v", err)
	}

	// Filter by entityId.
	t.Run("filter by entityId", func(t *testing.T) {
		result, err := storage.ListScoresByScorerID(ctx, ListScoresByScorerIDArgs{
			ScorerID:   scorerID,
			Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
			EntityID:   "agent-1",
		})
		if err != nil {
			t.Fatalf("returned error: %v", err)
		}
		if len(result.Scores) != 1 {
			t.Fatalf("expected 1 score, got %d", len(result.Scores))
		}
		if result.Scores[0].EntityID != "agent-1" {
			t.Errorf("expected entityId=agent-1, got %s", result.Scores[0].EntityID)
		}
	})

	// Filter by entityType.
	t.Run("filter by entityType", func(t *testing.T) {
		result, err := storage.ListScoresByScorerID(ctx, ListScoresByScorerIDArgs{
			ScorerID:   scorerID,
			Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
			EntityType: evals.ScoringEntityTypeWorkflow,
		})
		if err != nil {
			t.Fatalf("returned error: %v", err)
		}
		if len(result.Scores) != 1 {
			t.Fatalf("expected 1 score, got %d", len(result.Scores))
		}
		if result.Scores[0].EntityType != evals.ScoringEntityTypeWorkflow {
			t.Errorf("expected entityType=WORKFLOW, got %s", result.Scores[0].EntityType)
		}
	})

	// No filter returns all.
	t.Run("no filter returns all", func(t *testing.T) {
		result, err := storage.ListScoresByScorerID(ctx, ListScoresByScorerIDArgs{
			ScorerID:   scorerID,
			Pagination: domains.PaginationArgs{Page: 0, PerPage: 10},
		})
		if err != nil {
			t.Fatalf("returned error: %v", err)
		}
		if len(result.Scores) != 2 {
			t.Fatalf("expected 2 scores, got %d", len(result.Scores))
		}
	})
}

func TestInMemoryScoresStorage_PaginationDisabled(t *testing.T) {
	// TS: perPage=false (Go: PerPage<=0) means "return all".
	ctx := context.Background()
	storage := NewInMemoryScoresStorage()

	scorerID := "scorer-" + uuid.New().String()
	for i := 0; i < 5; i++ {
		payload := createSampleScore(createSampleScoreOpts{ScorerID: scorerID})
		if _, err := storage.SaveScore(ctx, payload); err != nil {
			t.Fatalf("SaveScore returned error: %v", err)
		}
	}

	// PerPage=0 means "all" (Go equivalent of TS perPage: false).
	result, err := storage.ListScoresByScorerID(ctx, ListScoresByScorerIDArgs{
		ScorerID:   scorerID,
		Pagination: domains.PaginationArgs{Page: 0, PerPage: 0},
	})
	if err != nil {
		t.Fatalf("ListScoresByScorerID returned error: %v", err)
	}
	if len(result.Scores) != 5 {
		t.Fatalf("expected 5 scores with pagination disabled, got %d", len(result.Scores))
	}
	if result.Pagination.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Pagination.Total)
	}
	// PerPage should be the sentinel PerPageDisabled (-1).
	if result.Pagination.PerPage != domains.PerPageDisabled {
		t.Errorf("expected perPage=%d (disabled), got %d", domains.PerPageDisabled, result.Pagination.PerPage)
	}
	// HasMore should be false when pagination is disabled.
	if result.Pagination.HasMore {
		t.Error("expected hasMore=false when pagination is disabled")
	}
}

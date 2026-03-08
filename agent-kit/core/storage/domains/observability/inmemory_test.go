// Ported from: stores/_test-utils/src/domains/observability/index.ts (createObservabilityTest)
// and: packages/core/src/storage/domains/observability/inmemory.test.ts
//
// The upstream mastra project tests observability storage via the shared test
// harness in stores/_test-utils. This Go file faithfully ports those tests
// against InMemoryObservabilityStorage.
package observability

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// strPtr returns a pointer to a string.
func strPtr(s string) *string { return &s }

// timePtr returns a pointer to a time.Time.
func timePtr(t time.Time) *time.Time { return &t }

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool { return &b }

// spanTypePtr returns a pointer to a SpanType.
func spanTypePtr(st SpanType) *SpanType { return &st }

// traceStatusPtr returns a pointer to a TraceStatus.
func traceStatusPtr(ts TraceStatus) *TraceStatus { return &ts }

// makeRootSpan creates a root span (no ParentSpanID) with sensible defaults.
func makeRootSpan(traceID, spanID, name string) CreateSpanRecord {
	return CreateSpanRecord{
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      name,
		SpanTyp:   SpanType("agent"),
		StartedAt: time.Now(),
	}
}

// makeChildSpan creates a child span with a parent reference.
func makeChildSpan(traceID, spanID, name, parentSpanID string) CreateSpanRecord {
	return CreateSpanRecord{
		TraceID:      traceID,
		SpanID:       spanID,
		Name:         name,
		SpanTyp:      SpanType("tool"),
		StartedAt:    time.Now(),
		ParentSpanID: strPtr(parentSpanID),
	}
}

// ===========================================================================
// Tests — Init & TracingStrategy
// ===========================================================================

func TestInMemoryObservabilityStorage_Init(t *testing.T) {
	// Init is a no-op for in-memory; verify it does not error.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	if err := storage.Init(ctx); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
}

func TestInMemoryObservabilityStorage_TracingStrategy(t *testing.T) {
	// Verify the tracing strategy returns the expected values.
	storage := NewInMemoryObservabilityStorage()
	info := storage.TracingStrategy()

	if info.Preferred != TracingStrategyRealtime {
		t.Errorf("expected preferred=%s, got %s", TracingStrategyRealtime, info.Preferred)
	}
	if len(info.Supported) != 3 {
		t.Fatalf("expected 3 supported strategies, got %d", len(info.Supported))
	}
	expected := []TracingStorageStrategy{
		TracingStrategyRealtime,
		TracingStrategyBatchWithUpdates,
		TracingStrategyInsertOnly,
	}
	for i, s := range info.Supported {
		if s != expected[i] {
			t.Errorf("supported[%d]: expected %s, got %s", i, expected[i], s)
		}
	}
}

// ===========================================================================
// Tests — CreateSpan & GetSpan
// ===========================================================================

func TestInMemoryObservabilityStorage_CreateAndGetSpan(t *testing.T) {
	// TS: "should create and retrieve a span"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	traceID := "trace-1"
	spanID := "span-1"
	span := makeRootSpan(traceID, spanID, "root-span")

	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: span}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	result, err := storage.GetSpan(ctx, GetSpanArgs{TraceID: traceID, SpanID: spanID})
	if err != nil {
		t.Fatalf("GetSpan returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected span to be returned, got nil")
	}
	if result.Span.TraceID != traceID {
		t.Errorf("traceId mismatch: got %s, want %s", result.Span.TraceID, traceID)
	}
	if result.Span.SpanID != spanID {
		t.Errorf("spanId mismatch: got %s, want %s", result.Span.SpanID, spanID)
	}
	if result.Span.Name != "root-span" {
		t.Errorf("name mismatch: got %s, want %s", result.Span.Name, "root-span")
	}
	if result.Span.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if result.Span.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestInMemoryObservabilityStorage_CreateSpan_MissingSpanID(t *testing.T) {
	// Creating a span without a spanID should return an error.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	span := CreateSpanRecord{
		TraceID:   "trace-1",
		SpanID:    "", // missing
		Name:      "bad-span",
		SpanTyp:   SpanType("agent"),
		StartedAt: time.Now(),
	}

	err := storage.CreateSpan(ctx, CreateSpanArgs{Span: span})
	if err == nil {
		t.Fatal("expected error for missing span ID, got nil")
	}
}

func TestInMemoryObservabilityStorage_CreateSpan_MissingTraceID(t *testing.T) {
	// Creating a span without a traceID should return an error.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	span := CreateSpanRecord{
		TraceID:   "", // missing
		SpanID:    "span-1",
		Name:      "bad-span",
		SpanTyp:   SpanType("agent"),
		StartedAt: time.Now(),
	}

	err := storage.CreateSpan(ctx, CreateSpanArgs{Span: span})
	if err == nil {
		t.Fatal("expected error for missing trace ID, got nil")
	}
}

func TestInMemoryObservabilityStorage_GetSpan_NotFound(t *testing.T) {
	// Getting a span from a non-existent trace should return nil.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	result, err := storage.GetSpan(ctx, GetSpanArgs{TraceID: "no-trace", SpanID: "no-span"})
	if err != nil {
		t.Fatalf("GetSpan returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent span, got %+v", result)
	}
}

func TestInMemoryObservabilityStorage_GetSpan_WrongSpanID(t *testing.T) {
	// Getting a span from an existing trace but with wrong spanID should return nil.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	span := makeRootSpan("trace-1", "span-1", "root")
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: span}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	result, err := storage.GetSpan(ctx, GetSpanArgs{TraceID: "trace-1", SpanID: "wrong-span"})
	if err != nil {
		t.Fatalf("GetSpan returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for wrong span ID, got %+v", result)
	}
}

// ===========================================================================
// Tests — GetRootSpan
// ===========================================================================

func TestInMemoryObservabilityStorage_GetRootSpan(t *testing.T) {
	// TS: "should retrieve the root span for a trace"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	traceID := "trace-root"
	rootSpan := makeRootSpan(traceID, "root-span", "my-root")
	childSpan := makeChildSpan(traceID, "child-span", "my-child", "root-span")

	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: rootSpan}); err != nil {
		t.Fatalf("CreateSpan (root) returned error: %v", err)
	}
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: childSpan}); err != nil {
		t.Fatalf("CreateSpan (child) returned error: %v", err)
	}

	result, err := storage.GetRootSpan(ctx, GetRootSpanArgs{TraceID: traceID})
	if err != nil {
		t.Fatalf("GetRootSpan returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected root span to be returned, got nil")
	}
	if result.Span.SpanID != "root-span" {
		t.Errorf("expected root spanId=root-span, got %s", result.Span.SpanID)
	}
	if result.Span.ParentSpanID != nil {
		t.Errorf("expected root span to have nil ParentSpanID, got %v", *result.Span.ParentSpanID)
	}
}

func TestInMemoryObservabilityStorage_GetRootSpan_NotFound(t *testing.T) {
	// Getting root span for non-existent trace should return nil.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	result, err := storage.GetRootSpan(ctx, GetRootSpanArgs{TraceID: "no-trace"})
	if err != nil {
		t.Fatalf("GetRootSpan returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent trace, got %+v", result)
	}
}

func TestInMemoryObservabilityStorage_GetRootSpan_NoRootInTrace(t *testing.T) {
	// A trace containing only child spans (no root) should return nil.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	traceID := "trace-no-root"
	childSpan := makeChildSpan(traceID, "child-only", "orphan-child", "nonexistent-parent")

	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: childSpan}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	result, err := storage.GetRootSpan(ctx, GetRootSpanArgs{TraceID: traceID})
	if err != nil {
		t.Fatalf("GetRootSpan returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil when trace has no root span, got %+v", result)
	}
}

// ===========================================================================
// Tests — GetTrace
// ===========================================================================

func TestInMemoryObservabilityStorage_GetTrace(t *testing.T) {
	// TS: "should retrieve all spans for a trace"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	traceID := "trace-full"
	now := time.Now()

	root := CreateSpanRecord{
		TraceID:   traceID,
		SpanID:    "root",
		Name:      "root-span",
		SpanTyp:   SpanType("agent"),
		StartedAt: now,
	}
	child1 := CreateSpanRecord{
		TraceID:      traceID,
		SpanID:       "child-1",
		Name:         "child-1-span",
		SpanTyp:      SpanType("tool"),
		StartedAt:    now.Add(10 * time.Millisecond),
		ParentSpanID: strPtr("root"),
	}
	child2 := CreateSpanRecord{
		TraceID:      traceID,
		SpanID:       "child-2",
		Name:         "child-2-span",
		SpanTyp:      SpanType("tool"),
		StartedAt:    now.Add(20 * time.Millisecond),
		ParentSpanID: strPtr("root"),
	}

	for _, s := range []CreateSpanRecord{root, child1, child2} {
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: s}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	result, err := storage.GetTrace(ctx, GetTraceArgs{TraceID: traceID})
	if err != nil {
		t.Fatalf("GetTrace returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected trace result, got nil")
	}
	if result.TraceID != traceID {
		t.Errorf("traceId mismatch: got %s, want %s", result.TraceID, traceID)
	}
	if len(result.Spans) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(result.Spans))
	}

	// Spans should be sorted by startedAt.
	for i := 1; i < len(result.Spans); i++ {
		if result.Spans[i].StartedAt.Before(result.Spans[i-1].StartedAt) {
			t.Errorf("spans not sorted by startedAt: span[%d]=%v before span[%d]=%v",
				i, result.Spans[i].StartedAt, i-1, result.Spans[i-1].StartedAt)
		}
	}
}

func TestInMemoryObservabilityStorage_GetTrace_NotFound(t *testing.T) {
	// Non-existent trace should return nil.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	result, err := storage.GetTrace(ctx, GetTraceArgs{TraceID: "no-trace"})
	if err != nil {
		t.Fatalf("GetTrace returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent trace, got %+v", result)
	}
}

// ===========================================================================
// Tests — ListTraces
// ===========================================================================

func TestInMemoryObservabilityStorage_ListTraces_Basic(t *testing.T) {
	// TS: "should list traces with pagination"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	now := time.Now()

	// Create 3 traces with root spans at different start times.
	for i := 0; i < 3; i++ {
		traceID := "trace-list-" + string(rune('A'+i))
		root := CreateSpanRecord{
			TraceID:   traceID,
			SpanID:    "root-" + string(rune('A'+i)),
			Name:      "root-" + string(rune('A'+i)),
			SpanTyp:   SpanType("agent"),
			StartedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 3 {
		t.Fatalf("expected 3 traces, got %d", len(result.Spans))
	}
	if result.Pagination.Total != 3 {
		t.Errorf("expected total=3, got %d", result.Pagination.Total)
	}
	if result.Pagination.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestInMemoryObservabilityStorage_ListTraces_Pagination(t *testing.T) {
	// TS: "should handle pagination correctly"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	now := time.Now()

	// Create 5 traces.
	for i := 0; i < 5; i++ {
		traceID := "trace-page-" + string(rune('A'+i))
		root := CreateSpanRecord{
			TraceID:   traceID,
			SpanID:    "root-" + string(rune('A'+i)),
			Name:      "root-" + string(rune('A'+i)),
			SpanTyp:   SpanType("agent"),
			StartedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	// First page (2 items).
	page1, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 2},
	})
	if err != nil {
		t.Fatalf("ListTraces (page 0) returned error: %v", err)
	}
	if len(page1.Spans) != 2 {
		t.Fatalf("expected 2 spans on page 0, got %d", len(page1.Spans))
	}
	if page1.Pagination.Total != 5 {
		t.Errorf("expected total=5, got %d", page1.Pagination.Total)
	}
	if !page1.Pagination.HasMore {
		t.Error("expected hasMore=true on page 0")
	}

	// Last page.
	page3, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 2, PerPage: 2},
	})
	if err != nil {
		t.Fatalf("ListTraces (page 2) returned error: %v", err)
	}
	if len(page3.Spans) != 1 {
		t.Fatalf("expected 1 span on page 2, got %d", len(page3.Spans))
	}
	if page3.Pagination.HasMore {
		t.Error("expected hasMore=false on last page")
	}
}

func TestInMemoryObservabilityStorage_ListTraces_DefaultPagination(t *testing.T) {
	// When pagination is nil, defaults should be applied (page=0, perPage=10).
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	root := makeRootSpan("trace-default", "root-default", "default-trace")
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	result, err := storage.ListTraces(ctx, ListTracesArgs{})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(result.Spans))
	}
	if result.Pagination.Page != 0 {
		t.Errorf("expected default page=0, got %d", result.Pagination.Page)
	}
	if result.Pagination.PerPage != 10 {
		t.Errorf("expected default perPage=10, got %d", result.Pagination.PerPage)
	}
}

func TestInMemoryObservabilityStorage_ListTraces_SortAscending(t *testing.T) {
	// TS: "should sort traces ascending"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	now := time.Now()
	for i := 0; i < 3; i++ {
		root := CreateSpanRecord{
			TraceID:   "trace-asc-" + string(rune('A'+i)),
			SpanID:    "root-" + string(rune('A'+i)),
			Name:      "root-" + string(rune('A'+i)),
			SpanTyp:   SpanType("agent"),
			StartedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		OrderBy: &TracesOrderBy{
			Field:     TracesOrderByStartedAt,
			Direction: domains.SortASC,
		},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 3 {
		t.Fatalf("expected 3 traces, got %d", len(result.Spans))
	}

	// Verify ascending order by startedAt.
	for i := 1; i < len(result.Spans); i++ {
		if result.Spans[i].StartedAt.Before(result.Spans[i-1].StartedAt) {
			t.Errorf("traces not sorted ASC: span[%d].startedAt=%v before span[%d].startedAt=%v",
				i, result.Spans[i].StartedAt, i-1, result.Spans[i-1].StartedAt)
		}
	}
}

func TestInMemoryObservabilityStorage_ListTraces_SortDescending(t *testing.T) {
	// TS: "should sort traces descending (default)"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	now := time.Now()
	for i := 0; i < 3; i++ {
		root := CreateSpanRecord{
			TraceID:   "trace-desc-" + string(rune('A'+i)),
			SpanID:    "root-" + string(rune('A'+i)),
			Name:      "root-" + string(rune('A'+i)),
			SpanTyp:   SpanType("agent"),
			StartedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		OrderBy: &TracesOrderBy{
			Field:     TracesOrderByStartedAt,
			Direction: domains.SortDESC,
		},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 3 {
		t.Fatalf("expected 3 traces, got %d", len(result.Spans))
	}

	// Verify descending order by startedAt.
	for i := 1; i < len(result.Spans); i++ {
		if result.Spans[i-1].StartedAt.Before(result.Spans[i].StartedAt) {
			t.Errorf("traces not sorted DESC: span[%d].startedAt=%v after span[%d].startedAt=%v",
				i-1, result.Spans[i-1].StartedAt, i, result.Spans[i].StartedAt)
		}
	}
}

func TestInMemoryObservabilityStorage_ListTraces_FilterByEntityType(t *testing.T) {
	// TS: "should filter traces by entity type"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	agentSpan := makeRootSpan("trace-agent", "root-agent", "agent-trace")
	agentSpan.EntityType = strPtr("agent")

	toolSpan := makeRootSpan("trace-tool", "root-tool", "tool-trace")
	toolSpan.EntityType = strPtr("tool")

	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: agentSpan}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: toolSpan}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{EntityType: strPtr("agent")},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 1 {
		t.Fatalf("expected 1 trace with entityType=agent, got %d", len(result.Spans))
	}
	if *result.Spans[0].EntityType != "agent" {
		t.Errorf("expected entityType=agent, got %s", *result.Spans[0].EntityType)
	}
}

func TestInMemoryObservabilityStorage_ListTraces_FilterByStatus(t *testing.T) {
	// TS: "should filter traces by status"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	now := time.Now()

	// Running trace (no endedAt, no error).
	runningSpan := makeRootSpan("trace-running", "root-running", "running-trace")

	// Successful trace (has endedAt, no error).
	successSpan := CreateSpanRecord{
		TraceID:   "trace-success",
		SpanID:    "root-success",
		Name:      "success-trace",
		SpanTyp:   SpanType("agent"),
		StartedAt: now,
		EndedAt:   timePtr(now.Add(time.Second)),
	}

	// Error trace (has error).
	errorSpan := CreateSpanRecord{
		TraceID:   "trace-error",
		SpanID:    "root-error",
		Name:      "error-trace",
		SpanTyp:   SpanType("agent"),
		StartedAt: now,
		Error:     "something went wrong",
	}

	for _, s := range []CreateSpanRecord{runningSpan, successSpan, errorSpan} {
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: s}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	// Filter by running status.
	t.Run("filter running", func(t *testing.T) {
		result, err := storage.ListTraces(ctx, ListTracesArgs{
			Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
			Filters:    &TracesFilter{Status: traceStatusPtr(TraceStatusRunning)},
		})
		if err != nil {
			t.Fatalf("ListTraces returned error: %v", err)
		}
		if len(result.Spans) != 1 {
			t.Fatalf("expected 1 running trace, got %d", len(result.Spans))
		}
		if result.Spans[0].Status != TraceStatusRunning {
			t.Errorf("expected status=running, got %s", result.Spans[0].Status)
		}
	})

	// Filter by success status.
	t.Run("filter success", func(t *testing.T) {
		result, err := storage.ListTraces(ctx, ListTracesArgs{
			Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
			Filters:    &TracesFilter{Status: traceStatusPtr(TraceStatusSuccess)},
		})
		if err != nil {
			t.Fatalf("ListTraces returned error: %v", err)
		}
		if len(result.Spans) != 1 {
			t.Fatalf("expected 1 success trace, got %d", len(result.Spans))
		}
		if result.Spans[0].Status != TraceStatusSuccess {
			t.Errorf("expected status=success, got %s", result.Spans[0].Status)
		}
	})

	// Filter by error status.
	t.Run("filter error", func(t *testing.T) {
		result, err := storage.ListTraces(ctx, ListTracesArgs{
			Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
			Filters:    &TracesFilter{Status: traceStatusPtr(TraceStatusError)},
		})
		if err != nil {
			t.Fatalf("ListTraces returned error: %v", err)
		}
		if len(result.Spans) != 1 {
			t.Fatalf("expected 1 error trace, got %d", len(result.Spans))
		}
		if result.Spans[0].Status != TraceStatusError {
			t.Errorf("expected status=error, got %s", result.Spans[0].Status)
		}
	})
}

func TestInMemoryObservabilityStorage_ListTraces_FilterByTags(t *testing.T) {
	// TS: "should filter traces by tags"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	span1 := makeRootSpan("trace-tag-1", "root-tag-1", "tagged-1")
	span1.Tags = []string{"production", "critical"}

	span2 := makeRootSpan("trace-tag-2", "root-tag-2", "tagged-2")
	span2.Tags = []string{"staging"}

	span3 := makeRootSpan("trace-tag-3", "root-tag-3", "tagged-3")
	span3.Tags = []string{"production", "low-priority"}

	for _, s := range []CreateSpanRecord{span1, span2, span3} {
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: s}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	// Filter by single tag.
	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{Tags: []string{"production"}},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 2 {
		t.Fatalf("expected 2 traces with tag=production, got %d", len(result.Spans))
	}

	// Filter by multiple tags (all must be present).
	result2, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{Tags: []string{"production", "critical"}},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result2.Spans) != 1 {
		t.Fatalf("expected 1 trace with tags=[production, critical], got %d", len(result2.Spans))
	}
}

func TestInMemoryObservabilityStorage_ListTraces_FilterByMetadata(t *testing.T) {
	// TS: "should filter traces by metadata"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	span1 := makeRootSpan("trace-meta-1", "root-meta-1", "meta-1")
	span1.Metadata = map[string]any{"env": "prod", "team": "alpha"}

	span2 := makeRootSpan("trace-meta-2", "root-meta-2", "meta-2")
	span2.Metadata = map[string]any{"env": "staging", "team": "beta"}

	for _, s := range []CreateSpanRecord{span1, span2} {
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: s}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{Metadata: map[string]any{"env": "prod"}},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 1 {
		t.Fatalf("expected 1 trace with metadata.env=prod, got %d", len(result.Spans))
	}
}

func TestInMemoryObservabilityStorage_ListTraces_FilterByDateRange(t *testing.T) {
	// TS: "should filter traces by date range"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		root := CreateSpanRecord{
			TraceID:   "trace-date-" + string(rune('A'+i)),
			SpanID:    "root-" + string(rune('A'+i)),
			Name:      "date-trace",
			SpanTyp:   SpanType("agent"),
			StartedAt: base.Add(time.Duration(i) * 24 * time.Hour),
		}
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	start := base.Add(1 * 24 * time.Hour) // Jan 2
	end := base.Add(3 * 24 * time.Hour)   // Jan 4

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters: &TracesFilter{
			StartedAt: &domains.DateRange{
				Start: timePtr(start),
				End:   timePtr(end),
			},
		},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	// Should include Jan 1, Jan 2, Jan 3 — traces with startedAt in [start, end]
	// Actually: Jan 2 (index 1), Jan 3 (index 2), Jan 4 (index 3)
	// start = Jan 2, end = Jan 4; filter: startedAt >= start AND startedAt <= end
	if len(result.Spans) != 3 {
		t.Fatalf("expected 3 traces in date range, got %d", len(result.Spans))
	}
}

func TestInMemoryObservabilityStorage_ListTraces_FilterByHasChildError(t *testing.T) {
	// TS: "should filter traces by hasChildError"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	// Trace with a child error.
	root1 := makeRootSpan("trace-err-child", "root-1", "root-1")
	child1 := makeChildSpan("trace-err-child", "child-err", "child-err", "root-1")
	child1.Error = "child failed"

	// Trace without child error.
	root2 := makeRootSpan("trace-no-err", "root-2", "root-2")
	child2 := makeChildSpan("trace-no-err", "child-ok", "child-ok", "root-2")

	for _, s := range []CreateSpanRecord{root1, child1, root2, child2} {
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: s}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	// Filter for traces with child error.
	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{HasChildError: boolPtr(true)},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 1 {
		t.Fatalf("expected 1 trace with hasChildError=true, got %d", len(result.Spans))
	}

	// Filter for traces without child error.
	result2, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{HasChildError: boolPtr(false)},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result2.Spans) != 1 {
		t.Fatalf("expected 1 trace with hasChildError=false, got %d", len(result2.Spans))
	}
}

func TestInMemoryObservabilityStorage_ListTraces_Empty(t *testing.T) {
	// Listing with no traces should return empty with correct pagination.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 0 {
		t.Errorf("expected 0 traces, got %d", len(result.Spans))
	}
	if result.Pagination.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Pagination.Total)
	}
	if result.Pagination.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestInMemoryObservabilityStorage_ListTraces_SkipsTracesWithoutRootSpan(t *testing.T) {
	// Traces that only contain child spans (no root) should not appear in listings.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	// Create a trace with only a child span (no root).
	childOnly := makeChildSpan("trace-orphan", "child-orphan", "orphan", "nonexistent-parent")
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: childOnly}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	// Create a trace with a proper root span.
	root := makeRootSpan("trace-with-root", "root-ok", "proper-trace")
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 1 {
		t.Fatalf("expected 1 trace (skipping orphan), got %d", len(result.Spans))
	}
}

// ===========================================================================
// Tests — UpdateSpan
// ===========================================================================

func TestInMemoryObservabilityStorage_UpdateSpan(t *testing.T) {
	// TS: "should update a span"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	span := makeRootSpan("trace-upd", "span-upd", "original-name")
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: span}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	newName := "updated-name"
	endTime := time.Now().Add(5 * time.Second)
	err := storage.UpdateSpan(ctx, UpdateSpanArgs{
		TraceID: "trace-upd",
		SpanID:  "span-upd",
		Updates: UpdateSpanRecord{
			Name:    &newName,
			EndedAt: &endTime,
		},
	})
	if err != nil {
		t.Fatalf("UpdateSpan returned error: %v", err)
	}

	result, err := storage.GetSpan(ctx, GetSpanArgs{TraceID: "trace-upd", SpanID: "span-upd"})
	if err != nil {
		t.Fatalf("GetSpan returned error: %v", err)
	}
	if result.Span.Name != "updated-name" {
		t.Errorf("expected name=updated-name, got %s", result.Span.Name)
	}
	if result.Span.EndedAt == nil {
		t.Fatal("expected endedAt to be set after update")
	}
}

func TestInMemoryObservabilityStorage_UpdateSpan_TraceNotFound(t *testing.T) {
	// Updating a span on a non-existent trace should return an error.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	newName := "updated"
	err := storage.UpdateSpan(ctx, UpdateSpanArgs{
		TraceID: "no-trace",
		SpanID:  "no-span",
		Updates: UpdateSpanRecord{Name: &newName},
	})
	if err == nil {
		t.Fatal("expected error for update on non-existent trace, got nil")
	}
}

func TestInMemoryObservabilityStorage_UpdateSpan_SpanNotFound(t *testing.T) {
	// Updating a non-existent span in an existing trace should return an error.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	root := makeRootSpan("trace-exists", "root-exists", "root")
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	newName := "updated"
	err := storage.UpdateSpan(ctx, UpdateSpanArgs{
		TraceID: "trace-exists",
		SpanID:  "wrong-span",
		Updates: UpdateSpanRecord{Name: &newName},
	})
	if err == nil {
		t.Fatal("expected error for update on non-existent span, got nil")
	}
}

func TestInMemoryObservabilityStorage_UpdateSpan_StatusRecomputed(t *testing.T) {
	// After updating a root span with an error, the trace status should change.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	root := makeRootSpan("trace-status", "root-status", "root")
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	// Initially the trace should be "running" (no endedAt, no error).
	traces, _ := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{Status: traceStatusPtr(TraceStatusRunning)},
	})
	if len(traces.Spans) != 1 {
		t.Fatalf("expected 1 running trace initially, got %d", len(traces.Spans))
	}

	// Update root span with an error.
	errVal := any("test error")
	if err := storage.UpdateSpan(ctx, UpdateSpanArgs{
		TraceID: "trace-status",
		SpanID:  "root-status",
		Updates: UpdateSpanRecord{Error: errVal},
	}); err != nil {
		t.Fatalf("UpdateSpan returned error: %v", err)
	}

	// Now it should be "error".
	traces, _ = storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{Status: traceStatusPtr(TraceStatusError)},
	})
	if len(traces.Spans) != 1 {
		t.Fatalf("expected 1 error trace after update, got %d", len(traces.Spans))
	}
}

// ===========================================================================
// Tests — BatchCreateSpans
// ===========================================================================

func TestInMemoryObservabilityStorage_BatchCreateSpans(t *testing.T) {
	// TS: "should batch create spans"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	now := time.Now()
	spans := []CreateSpanRecord{
		{
			TraceID:   "trace-batch",
			SpanID:    "root-batch",
			Name:      "root",
			SpanTyp:   SpanType("agent"),
			StartedAt: now,
		},
		{
			TraceID:      "trace-batch",
			SpanID:       "child-batch-1",
			Name:         "child-1",
			SpanTyp:      SpanType("tool"),
			StartedAt:    now.Add(10 * time.Millisecond),
			ParentSpanID: strPtr("root-batch"),
		},
		{
			TraceID:      "trace-batch",
			SpanID:       "child-batch-2",
			Name:         "child-2",
			SpanTyp:      SpanType("tool"),
			StartedAt:    now.Add(20 * time.Millisecond),
			ParentSpanID: strPtr("root-batch"),
		},
	}

	if err := storage.BatchCreateSpans(ctx, BatchCreateSpansArgs{Records: spans}); err != nil {
		t.Fatalf("BatchCreateSpans returned error: %v", err)
	}

	// Verify all spans can be retrieved.
	trace, err := storage.GetTrace(ctx, GetTraceArgs{TraceID: "trace-batch"})
	if err != nil {
		t.Fatalf("GetTrace returned error: %v", err)
	}
	if trace == nil {
		t.Fatal("expected trace, got nil")
	}
	if len(trace.Spans) != 3 {
		t.Fatalf("expected 3 spans in trace, got %d", len(trace.Spans))
	}
}

func TestInMemoryObservabilityStorage_BatchCreateSpans_MissingSpanID(t *testing.T) {
	// Batch create should fail if any span is missing spanID.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	spans := []CreateSpanRecord{
		{
			TraceID:   "trace-bad",
			SpanID:    "valid-span",
			Name:      "valid",
			SpanTyp:   SpanType("agent"),
			StartedAt: time.Now(),
		},
		{
			TraceID:   "trace-bad",
			SpanID:    "", // missing
			Name:      "invalid",
			SpanTyp:   SpanType("tool"),
			StartedAt: time.Now(),
		},
	}

	err := storage.BatchCreateSpans(ctx, BatchCreateSpansArgs{Records: spans})
	if err == nil {
		t.Fatal("expected error for batch with missing span ID, got nil")
	}
}

func TestInMemoryObservabilityStorage_BatchCreateSpans_MultipleTraces(t *testing.T) {
	// Batch create can span multiple traces.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	spans := []CreateSpanRecord{
		makeRootSpan("trace-A", "root-A", "root-A"),
		makeRootSpan("trace-B", "root-B", "root-B"),
	}

	if err := storage.BatchCreateSpans(ctx, BatchCreateSpansArgs{Records: spans}); err != nil {
		t.Fatalf("BatchCreateSpans returned error: %v", err)
	}

	traceA, _ := storage.GetTrace(ctx, GetTraceArgs{TraceID: "trace-A"})
	traceB, _ := storage.GetTrace(ctx, GetTraceArgs{TraceID: "trace-B"})
	if traceA == nil {
		t.Error("expected trace-A to exist")
	}
	if traceB == nil {
		t.Error("expected trace-B to exist")
	}
}

// ===========================================================================
// Tests — BatchUpdateSpans
// ===========================================================================

func TestInMemoryObservabilityStorage_BatchUpdateSpans(t *testing.T) {
	// TS: "should batch update spans"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	now := time.Now()
	root := CreateSpanRecord{
		TraceID:   "trace-batch-upd",
		SpanID:    "root",
		Name:      "root",
		SpanTyp:   SpanType("agent"),
		StartedAt: now,
	}
	child := CreateSpanRecord{
		TraceID:      "trace-batch-upd",
		SpanID:       "child",
		Name:         "child",
		SpanTyp:      SpanType("tool"),
		StartedAt:    now.Add(10 * time.Millisecond),
		ParentSpanID: strPtr("root"),
	}

	if err := storage.BatchCreateSpans(ctx, BatchCreateSpansArgs{Records: []CreateSpanRecord{root, child}}); err != nil {
		t.Fatalf("BatchCreateSpans returned error: %v", err)
	}

	rootName := "updated-root"
	childName := "updated-child"
	endTime := now.Add(time.Second)

	err := storage.BatchUpdateSpans(ctx, BatchUpdateSpansArgs{
		Records: []struct {
			TraceID string           `json:"traceId"`
			SpanID  string           `json:"spanId"`
			Updates UpdateSpanRecord `json:"updates"`
		}{
			{
				TraceID: "trace-batch-upd",
				SpanID:  "root",
				Updates: UpdateSpanRecord{Name: &rootName, EndedAt: &endTime},
			},
			{
				TraceID: "trace-batch-upd",
				SpanID:  "child",
				Updates: UpdateSpanRecord{Name: &childName, EndedAt: &endTime},
			},
		},
	})
	if err != nil {
		t.Fatalf("BatchUpdateSpans returned error: %v", err)
	}

	rootResult, _ := storage.GetSpan(ctx, GetSpanArgs{TraceID: "trace-batch-upd", SpanID: "root"})
	childResult, _ := storage.GetSpan(ctx, GetSpanArgs{TraceID: "trace-batch-upd", SpanID: "child"})

	if rootResult.Span.Name != "updated-root" {
		t.Errorf("expected root name=updated-root, got %s", rootResult.Span.Name)
	}
	if childResult.Span.Name != "updated-child" {
		t.Errorf("expected child name=updated-child, got %s", childResult.Span.Name)
	}
}

func TestInMemoryObservabilityStorage_BatchUpdateSpans_TraceNotFound(t *testing.T) {
	// Batch update should fail if the trace does not exist.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	newName := "updated"
	err := storage.BatchUpdateSpans(ctx, BatchUpdateSpansArgs{
		Records: []struct {
			TraceID string           `json:"traceId"`
			SpanID  string           `json:"spanId"`
			Updates UpdateSpanRecord `json:"updates"`
		}{
			{
				TraceID: "no-trace",
				SpanID:  "no-span",
				Updates: UpdateSpanRecord{Name: &newName},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for batch update on non-existent trace, got nil")
	}
}

// ===========================================================================
// Tests — BatchDeleteTraces
// ===========================================================================

func TestInMemoryObservabilityStorage_BatchDeleteTraces(t *testing.T) {
	// TS: "should batch delete traces"
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	// Create 3 traces.
	for _, id := range []string{"delete-A", "delete-B", "delete-C"} {
		root := makeRootSpan(id, "root-"+id, "root-"+id)
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	// Delete two of them.
	if err := storage.BatchDeleteTraces(ctx, BatchDeleteTracesArgs{
		TraceIDs: []string{"delete-A", "delete-C"},
	}); err != nil {
		t.Fatalf("BatchDeleteTraces returned error: %v", err)
	}

	// Verify deleted traces are gone.
	traceA, _ := storage.GetTrace(ctx, GetTraceArgs{TraceID: "delete-A"})
	traceC, _ := storage.GetTrace(ctx, GetTraceArgs{TraceID: "delete-C"})
	if traceA != nil {
		t.Error("expected trace-A to be deleted")
	}
	if traceC != nil {
		t.Error("expected trace-C to be deleted")
	}

	// Verify remaining trace still exists.
	traceB, _ := storage.GetTrace(ctx, GetTraceArgs{TraceID: "delete-B"})
	if traceB == nil {
		t.Error("expected trace-B to still exist")
	}
}

func TestInMemoryObservabilityStorage_BatchDeleteTraces_NonExistent(t *testing.T) {
	// Deleting non-existent traces should not error.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	err := storage.BatchDeleteTraces(ctx, BatchDeleteTracesArgs{
		TraceIDs: []string{"nonexistent-1", "nonexistent-2"},
	})
	if err != nil {
		t.Fatalf("BatchDeleteTraces returned error for non-existent traces: %v", err)
	}
}

// ===========================================================================
// Tests — DangerouslyClearAll
// ===========================================================================

func TestInMemoryObservabilityStorage_DangerouslyClearAll(t *testing.T) {
	// TS pattern from other domains — verify clear removes all data.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	// Create two traces.
	root1 := makeRootSpan("trace-clear-1", "root-1", "root-1")
	root2 := makeRootSpan("trace-clear-2", "root-2", "root-2")
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root1}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}
	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: root2}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	// Verify they exist.
	r1, _ := storage.GetTrace(ctx, GetTraceArgs{TraceID: "trace-clear-1"})
	r2, _ := storage.GetTrace(ctx, GetTraceArgs{TraceID: "trace-clear-2"})
	if r1 == nil || r2 == nil {
		t.Fatal("expected both traces to exist before clear")
	}

	// Clear all.
	if err := storage.DangerouslyClearAll(ctx); err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	// Verify they are gone.
	r1, _ = storage.GetTrace(ctx, GetTraceArgs{TraceID: "trace-clear-1"})
	r2, _ = storage.GetTrace(ctx, GetTraceArgs{TraceID: "trace-clear-2"})
	if r1 != nil {
		t.Error("expected first trace to be cleared")
	}
	if r2 != nil {
		t.Error("expected second trace to be cleared")
	}

	// ListTraces should return empty.
	list, _ := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
	})
	if len(list.Spans) != 0 {
		t.Errorf("expected 0 traces after clear, got %d", len(list.Spans))
	}
}

// ===========================================================================
// Tests — Span with all optional fields
// ===========================================================================

func TestInMemoryObservabilityStorage_CreateSpanWithAllFields(t *testing.T) {
	// Verify that all optional fields are preserved through create and get.
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	now := time.Now()
	endTime := now.Add(time.Second)

	span := CreateSpanRecord{
		TraceID:        "trace-all-fields",
		SpanID:         "span-all-fields",
		Name:           "full-span",
		SpanTyp:        SpanType("workflow"),
		IsEvent:        true,
		StartedAt:      now,
		EntityType:     strPtr("agent"),
		EntityID:       strPtr("agent-1"),
		EntityName:     strPtr("My Agent"),
		UserID:         strPtr("user-1"),
		OrganizationID: strPtr("org-1"),
		ResourceID:     strPtr("resource-1"),
		RunID:          strPtr("run-1"),
		SessionID:      strPtr("session-1"),
		ThreadID:       strPtr("thread-1"),
		RequestID:      strPtr("request-1"),
		Environment:    strPtr("production"),
		Source:         strPtr("api"),
		ServiceName:    strPtr("my-service"),
		Scope:          map[string]any{"module": "auth"},
		Metadata:       map[string]any{"key": "value"},
		Tags:           []string{"tag1", "tag2"},
		Attributes:     map[string]any{"attr": "val"},
		Links:          []any{"link1"},
		Input:          map[string]any{"prompt": "hello"},
		Output:         map[string]any{"response": "world"},
		EndedAt:        &endTime,
	}

	if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: span}); err != nil {
		t.Fatalf("CreateSpan returned error: %v", err)
	}

	result, err := storage.GetSpan(ctx, GetSpanArgs{TraceID: "trace-all-fields", SpanID: "span-all-fields"})
	if err != nil {
		t.Fatalf("GetSpan returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected span, got nil")
	}

	s := result.Span
	if s.Name != "full-span" {
		t.Errorf("name mismatch: got %s", s.Name)
	}
	if s.IsEvent != true {
		t.Error("expected isEvent=true")
	}
	if *s.EntityType != "agent" {
		t.Errorf("entityType mismatch: got %s", *s.EntityType)
	}
	if *s.EntityID != "agent-1" {
		t.Errorf("entityId mismatch: got %s", *s.EntityID)
	}
	if *s.UserID != "user-1" {
		t.Errorf("userId mismatch: got %s", *s.UserID)
	}
	if *s.OrganizationID != "org-1" {
		t.Errorf("organizationId mismatch: got %s", *s.OrganizationID)
	}
	if *s.Environment != "production" {
		t.Errorf("environment mismatch: got %s", *s.Environment)
	}
	if s.EndedAt == nil {
		t.Error("expected endedAt to be set")
	}
	if len(s.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(s.Tags))
	}
}

// ===========================================================================
// Tests — ComputeTraceStatus (unit)
// ===========================================================================

func TestComputeTraceStatus(t *testing.T) {
	now := time.Now()

	t.Run("running (no endedAt, no error)", func(t *testing.T) {
		span := &SpanRecord{StartedAt: now}
		if status := ComputeTraceStatus(span); status != TraceStatusRunning {
			t.Errorf("expected running, got %s", status)
		}
	})

	t.Run("success (endedAt set, no error)", func(t *testing.T) {
		ended := now.Add(time.Second)
		span := &SpanRecord{StartedAt: now, EndedAt: &ended}
		if status := ComputeTraceStatus(span); status != TraceStatusSuccess {
			t.Errorf("expected success, got %s", status)
		}
	})

	t.Run("error (error set, regardless of endedAt)", func(t *testing.T) {
		span := &SpanRecord{StartedAt: now, Error: "bad"}
		if status := ComputeTraceStatus(span); status != TraceStatusError {
			t.Errorf("expected error, got %s", status)
		}
	})

	t.Run("error takes priority over endedAt", func(t *testing.T) {
		ended := now.Add(time.Second)
		span := &SpanRecord{StartedAt: now, EndedAt: &ended, Error: "bad"}
		if status := ComputeTraceStatus(span); status != TraceStatusError {
			t.Errorf("expected error (priority), got %s", status)
		}
	})
}

// ===========================================================================
// Tests — ListTraces filter by correlation IDs and deployment context
// ===========================================================================

func TestInMemoryObservabilityStorage_ListTraces_FilterByCorrelationIDs(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	span1 := makeRootSpan("trace-corr-1", "root-corr-1", "corr-1")
	span1.RunID = strPtr("run-123")
	span1.SessionID = strPtr("session-abc")

	span2 := makeRootSpan("trace-corr-2", "root-corr-2", "corr-2")
	span2.RunID = strPtr("run-456")
	span2.SessionID = strPtr("session-xyz")

	for _, s := range []CreateSpanRecord{span1, span2} {
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: s}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	t.Run("filter by runId", func(t *testing.T) {
		result, err := storage.ListTraces(ctx, ListTracesArgs{
			Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
			Filters:    &TracesFilter{RunID: strPtr("run-123")},
		})
		if err != nil {
			t.Fatalf("ListTraces returned error: %v", err)
		}
		if len(result.Spans) != 1 {
			t.Fatalf("expected 1 trace with runId=run-123, got %d", len(result.Spans))
		}
	})

	t.Run("filter by sessionId", func(t *testing.T) {
		result, err := storage.ListTraces(ctx, ListTracesArgs{
			Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
			Filters:    &TracesFilter{SessionID: strPtr("session-xyz")},
		})
		if err != nil {
			t.Fatalf("ListTraces returned error: %v", err)
		}
		if len(result.Spans) != 1 {
			t.Fatalf("expected 1 trace with sessionId=session-xyz, got %d", len(result.Spans))
		}
	})
}

func TestInMemoryObservabilityStorage_ListTraces_FilterByEnvironment(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryObservabilityStorage()

	span1 := makeRootSpan("trace-env-1", "root-env-1", "env-1")
	span1.Environment = strPtr("production")

	span2 := makeRootSpan("trace-env-2", "root-env-2", "env-2")
	span2.Environment = strPtr("staging")

	for _, s := range []CreateSpanRecord{span1, span2} {
		if err := storage.CreateSpan(ctx, CreateSpanArgs{Span: s}); err != nil {
			t.Fatalf("CreateSpan returned error: %v", err)
		}
	}

	result, err := storage.ListTraces(ctx, ListTracesArgs{
		Pagination: &domains.PaginationArgs{Page: 0, PerPage: 10},
		Filters:    &TracesFilter{Environment: strPtr("production")},
	})
	if err != nil {
		t.Fatalf("ListTraces returned error: %v", err)
	}
	if len(result.Spans) != 1 {
		t.Fatalf("expected 1 trace with environment=production, got %d", len(result.Spans))
	}
}

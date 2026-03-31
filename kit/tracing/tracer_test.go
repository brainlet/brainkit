package tracing

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestTracer_StartEndSpan(t *testing.T) {
	store := NewMemoryTraceStore(100)
	tracer := NewTracer(store, 1.0)

	ctx := context.Background()
	span := tracer.StartSpan("test.op", ctx)
	span.SetAttribute("key", "value")
	span.SetSource("test.ts")
	time.Sleep(5 * time.Millisecond)
	span.End(nil)

	traces, err := store.ListTraces(TraceQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].RootSpan != "test.op" {
		t.Fatalf("expected root span 'test.op', got %q", traces[0].RootSpan)
	}
}

func TestTracer_ChildSpan(t *testing.T) {
	store := NewMemoryTraceStore(100)
	tracer := NewTracer(store, 1.0)

	ctx := context.Background()
	parent := tracer.StartSpan("parent", ctx)
	childCtx := parent.ChildContext(ctx)
	child := tracer.StartSpan("child", childCtx)
	child.End(nil)
	parent.End(nil)

	spans, err := store.GetTrace(parent.span.TraceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	// Find child span and verify parent linkage
	for _, s := range spans {
		if s.Name == "child" {
			if s.ParentID != parent.span.SpanID {
				t.Fatalf("child parent mismatch: %q != %q", s.ParentID, parent.span.SpanID)
			}
		}
	}
}

func TestTracer_ErrorSpan(t *testing.T) {
	store := NewMemoryTraceStore(100)
	tracer := NewTracer(store, 1.0)

	span := tracer.StartSpan("failing", context.Background())
	span.End(errForTest("something broke"))

	spans, _ := store.GetTrace(span.span.TraceID)
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status != "error" {
		t.Fatalf("expected status 'error', got %q", spans[0].Status)
	}
	if spans[0].Error != "something broke" {
		t.Fatalf("expected error msg, got %q", spans[0].Error)
	}
}

func TestTracer_NilStore_NoOp(t *testing.T) {
	tracer := NewTracer(nil, 1.0)
	span := tracer.StartSpan("noop", context.Background())
	span.SetAttribute("key", "value")
	span.End(nil) // should not panic
}

func TestMemoryTraceStore_RingBuffer(t *testing.T) {
	store := NewMemoryTraceStore(3)

	for i := 0; i < 5; i++ {
		store.RecordSpan(Span{
			TraceID: "trace-" + string(rune('a'+i)),
			Name:    "op",
		})
	}

	// Ring buffer size 3 — should only have last 3
	traces, _ := store.ListTraces(TraceQuery{})
	if len(traces) > 3 {
		t.Fatalf("expected max 3 traces, got %d", len(traces))
	}
}

func TestMemoryTraceStore_QueryByStatus(t *testing.T) {
	store := NewMemoryTraceStore(100)
	store.RecordSpan(Span{TraceID: "t1", Name: "ok-op", Status: "ok"})
	store.RecordSpan(Span{TraceID: "t2", Name: "err-op", Status: "error", Error: "fail"})

	errors, _ := store.ListTraces(TraceQuery{Status: "error"})
	if len(errors) != 1 {
		t.Fatalf("expected 1 error trace, got %d", len(errors))
	}
	if errors[0].TraceID != "t2" {
		t.Fatalf("expected trace t2, got %q", errors[0].TraceID)
	}
}

func TestTraceContext_Propagation(t *testing.T) {
	ctx := context.Background()
	tc := TraceContext{TraceID: "abc", SpanID: "def", ParentID: ""}
	ctx = WithTraceContext(ctx, tc)

	got, ok := TraceContextFromCtx(ctx)
	if !ok {
		t.Fatal("expected trace context in ctx")
	}
	if got.TraceID != "abc" || got.SpanID != "def" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestSQLiteTraceStore_RecordAndGet(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	span := Span{
		TraceID:   "trace-1",
		SpanID:    "span-1",
		ParentID:  "",
		Name:      "gateway.request",
		Source:    "test.ts",
		StartTime: now,
		Duration:  150 * time.Millisecond,
		Status:    "ok",
		Attributes: map[string]string{"method": "POST", "path": "/api/chat"},
	}
	if err := store.RecordSpan(span); err != nil {
		t.Fatal(err)
	}

	child := Span{
		TraceID:   "trace-1",
		SpanID:    "span-2",
		ParentID:  "span-1",
		Name:      "handler:ts.chatbot.ask",
		Source:    "chatbot.ts",
		StartTime: now.Add(5 * time.Millisecond),
		Duration:  140 * time.Millisecond,
		Status:    "ok",
	}
	if err := store.RecordSpan(child); err != nil {
		t.Fatal(err)
	}

	spans, err := store.GetTrace("trace-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}
	if spans[0].Name != "gateway.request" {
		t.Fatalf("expected first span 'gateway.request', got %q", spans[0].Name)
	}
	if spans[0].Attributes["method"] != "POST" {
		t.Fatalf("expected attribute method=POST, got %q", spans[0].Attributes["method"])
	}
	if spans[1].ParentID != "span-1" {
		t.Fatalf("expected child parentID 'span-1', got %q", spans[1].ParentID)
	}
}

func TestSQLiteTraceStore_ListTraces(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	store.RecordSpan(Span{TraceID: "t1", SpanID: "s1", Name: "root1", StartTime: now, Duration: 100 * time.Millisecond, Status: "ok"})
	store.RecordSpan(Span{TraceID: "t1", SpanID: "s2", ParentID: "s1", Name: "child1", StartTime: now, Duration: 50 * time.Millisecond, Status: "ok"})
	store.RecordSpan(Span{TraceID: "t2", SpanID: "s3", Name: "root2", StartTime: now, Duration: 200 * time.Millisecond, Status: "error", Error: "timeout"})

	// All traces
	all, err := store.ListTraces(TraceQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(all))
	}

	// Error traces only
	errors, _ := store.ListTraces(TraceQuery{Status: "error"})
	if len(errors) != 1 {
		t.Fatalf("expected 1 error trace, got %d", len(errors))
	}

	// MinDuration filter
	slow, _ := store.ListTraces(TraceQuery{MinDuration: 150 * time.Millisecond})
	if len(slow) != 1 {
		t.Fatalf("expected 1 slow trace, got %d", len(slow))
	}

	// Limit
	limited, _ := store.ListTraces(TraceQuery{Limit: 1})
	if len(limited) != 1 {
		t.Fatalf("expected 1 trace with limit, got %d", len(limited))
	}
}

func TestSQLiteTraceStore_QueryBySource(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	store.RecordSpan(Span{TraceID: "t1", SpanID: "s1", Name: "op1", Source: "chatbot.ts", StartTime: now, Status: "ok"})
	store.RecordSpan(Span{TraceID: "t2", SpanID: "s2", Name: "op2", Source: "logger.ts", StartTime: now, Status: "ok"})

	filtered, _ := store.ListTraces(TraceQuery{Source: "chatbot.ts"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 trace for chatbot.ts, got %d", len(filtered))
	}
	if filtered[0].TraceID != "t1" {
		t.Fatalf("expected trace t1, got %q", filtered[0].TraceID)
	}
}

func TestSQLiteTraceStore_WithTracer(t *testing.T) {
	db := openTestDB(t)
	store, err := NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}
	tracer := NewTracer(store, 1.0)

	ctx := context.Background()
	span := tracer.StartSpan("sqlite.test", ctx)
	span.SetAttribute("db", "test")
	span.SetSource("test-service")
	time.Sleep(2 * time.Millisecond)
	span.End(nil)

	spans, err := store.GetTrace(span.span.TraceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Source != "test-service" {
		t.Fatalf("expected source 'test-service', got %q", spans[0].Source)
	}
	if spans[0].Duration < 2*time.Millisecond {
		t.Fatalf("expected duration >= 2ms, got %v", spans[0].Duration)
	}
}

type testErr string

func errForTest(msg string) error { return testErr(msg) }
func (e testErr) Error() string   { return string(e) }

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

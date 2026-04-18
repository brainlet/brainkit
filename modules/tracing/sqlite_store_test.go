package tracing_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	internaltracing "github.com/brainlet/brainkit/internal/tracing"
	modtracing "github.com/brainlet/brainkit/modules/tracing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSQLiteTraceStore_RecordAndGet(t *testing.T) {
	db := openTestDB(t)
	store, err := modtracing.NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	span := internaltracing.Span{
		TraceID:    "trace-1",
		SpanID:     "span-1",
		Name:       "gateway.request",
		Source:     "test.ts",
		StartTime:  now,
		Duration:   150 * time.Millisecond,
		Status:     "ok",
		Attributes: map[string]string{"method": "POST", "path": "/api/chat"},
	}
	if err := store.RecordSpan(span); err != nil {
		t.Fatal(err)
	}

	child := internaltracing.Span{
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
	store, err := modtracing.NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	store.RecordSpan(internaltracing.Span{TraceID: "t1", SpanID: "s1", Name: "root1", StartTime: now, Duration: 100 * time.Millisecond, Status: "ok"})
	store.RecordSpan(internaltracing.Span{TraceID: "t1", SpanID: "s2", ParentID: "s1", Name: "child1", StartTime: now, Duration: 50 * time.Millisecond, Status: "ok"})
	store.RecordSpan(internaltracing.Span{TraceID: "t2", SpanID: "s3", Name: "root2", StartTime: now, Duration: 200 * time.Millisecond, Status: "error", Error: "timeout"})

	all, err := store.ListTraces(internaltracing.TraceQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(all))
	}

	errors, _ := store.ListTraces(internaltracing.TraceQuery{Status: "error"})
	if len(errors) != 1 {
		t.Fatalf("expected 1 error trace, got %d", len(errors))
	}

	slow, _ := store.ListTraces(internaltracing.TraceQuery{MinDuration: 150 * time.Millisecond})
	if len(slow) != 1 {
		t.Fatalf("expected 1 slow trace, got %d", len(slow))
	}

	limited, _ := store.ListTraces(internaltracing.TraceQuery{Limit: 1})
	if len(limited) != 1 {
		t.Fatalf("expected 1 trace with limit, got %d", len(limited))
	}
}

func TestSQLiteTraceStore_QueryBySource(t *testing.T) {
	db := openTestDB(t)
	store, err := modtracing.NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	store.RecordSpan(internaltracing.Span{TraceID: "t1", SpanID: "s1", Name: "op1", Source: "chatbot.ts", StartTime: now, Status: "ok"})
	store.RecordSpan(internaltracing.Span{TraceID: "t2", SpanID: "s2", Name: "op2", Source: "logger.ts", StartTime: now, Status: "ok"})

	filtered, _ := store.ListTraces(internaltracing.TraceQuery{Source: "chatbot.ts"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 trace for chatbot.ts, got %d", len(filtered))
	}
	if filtered[0].TraceID != "t1" {
		t.Fatalf("expected trace t1, got %q", filtered[0].TraceID)
	}
}

func TestSQLiteTraceStore_WithTracer(t *testing.T) {
	db := openTestDB(t)
	store, err := modtracing.NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}
	tracer := internaltracing.NewTracer(store, 1.0)

	ctx := context.Background()
	span := tracer.StartSpan("sqlite.test", ctx)
	span.SetAttribute("db", "test")
	span.SetSource("test-service")
	time.Sleep(2 * time.Millisecond)
	span.End(nil)

	spans, err := store.GetTrace(span.TraceID())
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

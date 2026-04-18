package tracing

import (
	"context"
	"errors"
	"testing"
	"time"
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
	span.End(errors.New("something broke"))

	spans, _ := store.GetTrace(span.TraceID())
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

// SQLiteTraceStore tests moved to modules/tracing/sqlite_store_test.go
// when the store implementation was extracted during session 05.

package tracing

import (
	"context"
	"math/rand"
	"github.com/brainlet/brainkit/internal/syncx"
	"time"

	"github.com/google/uuid"
)






type traceCtxKey struct{}
type sampledCtxKey struct{}

// WithTraceContext injects trace context into a Go context.
func WithTraceContext(ctx context.Context, tc TraceContext) context.Context {
	return context.WithValue(ctx, traceCtxKey{}, tc)
}

// WithSampled marks the context with an explicit sampling decision.
// When sampled=false, StartSpan returns a no-op handle.
func WithSampled(ctx context.Context, sampled bool) context.Context {
	return context.WithValue(ctx, sampledCtxKey{}, sampled)
}

// TraceContextFromCtx extracts trace context from a Go context.
func TraceContextFromCtx(ctx context.Context) (TraceContext, bool) {
	tc, ok := ctx.Value(traceCtxKey{}).(TraceContext)
	return tc, ok
}

// Tracer creates and records spans. No-op if store is nil.
type Tracer struct {
	store      TraceStore
	sampleRate float64
}

// NewTracer creates a tracer. If store is nil, all operations are no-ops.
func NewTracer(store TraceStore, sampleRate float64) *Tracer {
	if sampleRate <= 0 {
		sampleRate = 1.0
	}
	return &Tracer{store: store, sampleRate: sampleRate}
}

// StartSpan begins a new span. Returns a SpanHandle for ending it.
// Respects sampleRate: if < 1.0, only a fraction of NEW traces are sampled.
// Child spans (with existing traceID in context) are always sampled if the parent was.
func (t *Tracer) StartSpan(name string, ctx context.Context) *SpanHandle {
	if t.store == nil {
		return &SpanHandle{}
	}

	// Explicit sampled=false from inbound message — suppress span creation
	if sampled, ok := ctx.Value(sampledCtxKey{}).(bool); ok && !sampled {
		return &SpanHandle{}
	}

	tc, hasTrace := TraceContextFromCtx(ctx)

	// Sample rate check — only for root spans (no existing trace)
	if !hasTrace && t.sampleRate < 1.0 {
		// Deterministic sampling based on a quick random check
		if rand.Intn(1000) >= int(t.sampleRate*1000) {
			return &SpanHandle{} // not sampled — no-op handle
		}
	}

	tc, _ = TraceContextFromCtx(ctx)
	if tc.TraceID == "" {
		tc.TraceID = uuid.NewString()
	}

	spanID := newSpanID()
	return &SpanHandle{
		tracer: t,
		span: Span{
			TraceID:    tc.TraceID,
			SpanID:     spanID,
			ParentID:   tc.SpanID,
			Name:       name,
			StartTime:  time.Now(),
			Status:     "ok",
			Attributes: make(map[string]string),
		},
	}
}

// SpanHandle tracks an in-flight span.
type SpanHandle struct {
	tracer *Tracer
	span   Span
}

// SetAttribute adds metadata to the span.
func (h *SpanHandle) SetAttribute(key, value string) {
	if h.tracer == nil || h.tracer.store == nil {
		return
	}
	h.span.Attributes[key] = value
}

// SetSource sets the deployment source.
func (h *SpanHandle) SetSource(source string) {
	if h.tracer == nil {
		return
	}
	h.span.Source = source
}

// End completes the span and records it.
func (h *SpanHandle) End(err error) {
	if h.tracer == nil || h.tracer.store == nil {
		return
	}
	h.span.Duration = time.Since(h.span.StartTime)
	if err != nil {
		h.span.Status = "error"
		h.span.Error = err.Error()
	}
	h.tracer.store.RecordSpan(h.span)
}

// ChildContext creates a context with this span as parent.
func (h *SpanHandle) ChildContext(ctx context.Context) context.Context {
	if h.tracer == nil {
		return ctx
	}
	return WithTraceContext(ctx, TraceContext{
		TraceID:  h.span.TraceID,
		SpanID:   h.span.SpanID,
		ParentID: h.span.ParentID,
	})
}

func newSpanID() string {
	return uuid.NewString()[:16]
}

// ── MemoryTraceStore — ring buffer, fast, lost on restart ──

// MemoryTraceStore stores spans in a ring buffer.
type MemoryTraceStore struct {
	mu     syncx.Mutex
	spans  []Span
	maxLen int
	pos    int
	full   bool
}

// NewMemoryTraceStore creates a memory-backed trace store.
func NewMemoryTraceStore(maxSpans int) *MemoryTraceStore {
	if maxSpans <= 0 {
		maxSpans = 10000
	}
	return &MemoryTraceStore{
		spans:  make([]Span, maxSpans),
		maxLen: maxSpans,
	}
}

func (s *MemoryTraceStore) RecordSpan(span Span) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spans[s.pos] = span
	s.pos++
	if s.pos >= s.maxLen {
		s.pos = 0
		s.full = true
	}
	return nil
}

func (s *MemoryTraceStore) GetTrace(traceID string) ([]Span, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []Span
	for _, span := range s.allSpans() {
		if span.TraceID == traceID {
			result = append(result, span)
		}
	}
	return result, nil
}

func (s *MemoryTraceStore) ListTraces(query TraceQuery) ([]TraceSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	traces := make(map[string]*TraceSummary)
	for _, span := range s.allSpans() {
		if !query.Since.IsZero() && span.StartTime.Before(query.Since) {
			continue
		}
		if !query.Until.IsZero() && span.StartTime.After(query.Until) {
			continue
		}
		if query.Source != "" && span.Source != query.Source {
			continue
		}
		if query.Status != "" && span.Status != query.Status {
			continue
		}

		ts, ok := traces[span.TraceID]
		if !ok {
			ts = &TraceSummary{
				TraceID:   span.TraceID,
				StartTime: span.StartTime,
				Status:    "ok",
			}
			traces[span.TraceID] = ts
		}
		ts.SpanCount++
		if span.ParentID == "" {
			ts.RootSpan = span.Name
			ts.Duration = span.Duration
		}
		if span.Status == "error" {
			ts.Status = "error"
		}
	}

	result := make([]TraceSummary, 0, len(traces))
	for _, ts := range traces {
		if query.MinDuration > 0 && ts.Duration < query.MinDuration {
			continue
		}
		result = append(result, *ts)
	}

	if query.Limit > 0 && len(result) > query.Limit {
		result = result[:query.Limit]
	}
	return result, nil
}

func (s *MemoryTraceStore) Close() error { return nil }

func (s *MemoryTraceStore) allSpans() []Span {
	if s.full {
		return s.spans
	}
	return s.spans[:s.pos]
}

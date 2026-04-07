package types

import "time"

// Span represents one operation in a trace.
type Span struct {
	TraceID    string            `json:"traceId"`
	SpanID     string            `json:"spanId"`
	ParentID   string            `json:"parentId,omitempty"`
	Name       string            `json:"name"`
	Source     string            `json:"source,omitempty"`
	StartTime  time.Time         `json:"startTime"`
	Duration   time.Duration     `json:"duration"`
	Status     string            `json:"status"` // "ok", "error"
	Error      string            `json:"error,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// TraceContext carries trace propagation data through context.
type TraceContext struct {
	TraceID  string
	SpanID   string
	ParentID string
}

// TraceStore persists spans for querying.
type TraceStore interface {
	RecordSpan(span Span) error
	GetTrace(traceID string) ([]Span, error)
	ListTraces(query TraceQuery) ([]TraceSummary, error)
	Close() error
}

// TraceQuery filters trace lookups.
type TraceQuery struct {
	Since       time.Time
	Until       time.Time
	Source      string
	MinDuration time.Duration
	Status      string
	Limit       int
}

// TraceSummary is the abbreviated view of a trace.
type TraceSummary struct {
	TraceID   string        `json:"traceId"`
	RootSpan  string        `json:"rootSpan"`
	SpanCount int           `json:"spanCount"`
	Duration  time.Duration `json:"duration"`
	Status    string        `json:"status"`
	StartTime time.Time     `json:"startTime"`
}

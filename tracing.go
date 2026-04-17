package brainkit

import (
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/types"
)

// TraceStore persists spans for querying.
type TraceStore = types.TraceStore

// Span represents one operation in a trace.
type Span = types.Span

// NewMemoryTraceStore creates a ring-buffer trace store. Lost on restart.
// For durable storage, use modules/tracing.NewSQLiteTraceStore.
func NewMemoryTraceStore(maxSpans int) TraceStore {
	return tracing.NewMemoryTraceStore(maxSpans)
}

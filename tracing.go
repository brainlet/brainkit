package brainkit

import (
	"database/sql"

	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/types"
)

// TraceStore persists spans for querying.
type TraceStore = types.TraceStore

// Span represents one operation in a trace.
type Span = types.Span

// NewMemoryTraceStore creates a ring-buffer trace store. Lost on restart.
func NewMemoryTraceStore(maxSpans int) TraceStore {
	return tracing.NewMemoryTraceStore(maxSpans)
}

// NewSQLiteTraceStore creates a persistent SQLite-backed trace store.
func NewSQLiteTraceStore(db *sql.DB) (TraceStore, error) {
	return tracing.NewSQLiteTraceStore(db)
}

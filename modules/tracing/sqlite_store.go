package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	coretracing "github.com/brainlet/brainkit/internal/tracing"
)

// Re-export core types so sqlite_store.go can use short names.
type (
	Span         = coretracing.Span
	TraceQuery   = coretracing.TraceQuery
	TraceSummary = coretracing.TraceSummary
	TraceStore   = coretracing.TraceStore
)

const sqliteTraceSchema = `
CREATE TABLE IF NOT EXISTS traces (
    trace_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    parent_id TEXT,
    name TEXT NOT NULL,
    source TEXT,
    start_time TEXT NOT NULL,
    duration_ns INTEGER NOT NULL,
    status TEXT NOT NULL,
    error TEXT,
    attributes TEXT,
    PRIMARY KEY (trace_id, span_id)
);
CREATE INDEX IF NOT EXISTS idx_traces_start ON traces(start_time);
CREATE INDEX IF NOT EXISTS idx_traces_source ON traces(source);
`

// SQLiteTraceStore stores spans in a SQLite database. Survives restarts.
// Optional retention: auto-deletes spans older than the configured duration.
type SQLiteTraceStore struct {
	db        *sql.DB
	retention time.Duration
	stopClean chan struct{}
	cleanWG   sync.WaitGroup
	closeMu   sync.Mutex
	cleanWait sync.Once
	cleanDone chan struct{}

	closeStarted   bool
	closing        bool
	closed         bool
	cleanupRunning atomic.Bool
}

// SQLiteTraceStoreOption configures a SQLiteTraceStore.
type SQLiteTraceStoreOption func(*SQLiteTraceStore)

// WithRetention sets automatic cleanup of spans older than d.
// A background goroutine runs every hour to delete expired spans.
func WithRetention(d time.Duration) SQLiteTraceStoreOption {
	return func(s *SQLiteTraceStore) { s.retention = d }
}

// NewSQLiteTraceStore creates a persistent trace store backed by a sql.DB.
// Creates the traces table if it doesn't exist.
func NewSQLiteTraceStore(db *sql.DB, opts ...SQLiteTraceStoreOption) (*SQLiteTraceStore, error) {
	if _, err := db.Exec(sqliteTraceSchema); err != nil {
		return nil, fmt.Errorf("tracing: create table: %w", err)
	}
	s := &SQLiteTraceStore{db: db, stopClean: make(chan struct{}), cleanDone: make(chan struct{})}
	for _, opt := range opts {
		opt(s)
	}
	if s.retention > 0 {
		s.cleanupRunning.Store(true)
		s.cleanWG.Add(1)
		go s.cleanupLoop()
	}
	return s, nil
}

func (s *SQLiteTraceStore) cleanupLoop() {
	defer func() {
		s.cleanupRunning.Store(false)
		s.cleanWG.Done()
	}()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-s.retention).Format(time.RFC3339Nano)
			s.db.Exec("DELETE FROM traces WHERE start_time < ?", cutoff)
		case <-s.stopClean:
			return
		}
	}
}

func (s *SQLiteTraceStore) RecordSpan(span Span) error {
	attrsJSON := "{}"
	if len(span.Attributes) > 0 {
		b, _ := json.Marshal(span.Attributes)
		attrsJSON = string(b)
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO traces (trace_id, span_id, parent_id, name, source, start_time, duration_ns, status, error, attributes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		span.TraceID, span.SpanID, span.ParentID, span.Name, span.Source,
		span.StartTime.Format(time.RFC3339Nano), span.Duration.Nanoseconds(),
		span.Status, span.Error, attrsJSON,
	)
	return err
}

func (s *SQLiteTraceStore) GetTrace(traceID string) ([]Span, error) {
	rows, err := s.db.Query(
		`SELECT trace_id, span_id, parent_id, name, source, start_time, duration_ns, status, error, attributes
		 FROM traces WHERE trace_id = ? ORDER BY start_time`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSpans(rows)
}

func (s *SQLiteTraceStore) ListTraces(query TraceQuery) ([]TraceSummary, error) {
	q := `SELECT trace_id, span_id, parent_id, name, source, start_time, duration_ns, status, error, attributes FROM traces WHERE 1=1`
	var args []any

	if !query.Since.IsZero() {
		q += ` AND start_time >= ?`
		args = append(args, query.Since.Format(time.RFC3339Nano))
	}
	if !query.Until.IsZero() {
		q += ` AND start_time <= ?`
		args = append(args, query.Until.Format(time.RFC3339Nano))
	}
	if query.Source != "" {
		q += ` AND source = ?`
		args = append(args, query.Source)
	}
	if query.Status != "" {
		q += ` AND status = ?`
		args = append(args, query.Status)
	}
	q += ` ORDER BY start_time DESC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spans, err := scanSpans(rows)
	if err != nil {
		return nil, err
	}

	// Aggregate spans into trace summaries
	traces := make(map[string]*TraceSummary)
	for _, span := range spans {
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

func (s *SQLiteTraceStore) Close() error {
	return s.CloseContext(context.Background())
}

func (s *SQLiteTraceStore) CloseContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	if s == nil {
		return nil
	}
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return nil
	}
	if !s.closeStarted {
		close(s.stopClean)
		s.closeStarted = true
	}
	s.closing = true
	cleanDone := s.cleanDone
	if cleanDone == nil {
		cleanDone = make(chan struct{})
		s.cleanDone = cleanDone
	}
	s.cleanWait.Do(func() {
		go func() {
			s.cleanWG.Wait()
			close(cleanDone)
		}()
	})
	s.closeMu.Unlock()

	select {
	case <-cleanDone:
	case <-ctx.Done():
		s.closeMu.Lock()
		s.closing = false
		s.closeMu.Unlock()
		return ctx.Err()
	}

	var err error
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		s.closing = false
		return nil
	}
	if s.db != nil {
		err = s.db.Close()
		if err != nil {
			s.closing = false
			return err
		}
		s.db = nil
	}
	s.closed = true
	s.closing = false
	return err
}

func scanSpans(rows *sql.Rows) ([]Span, error) {
	var spans []Span
	for rows.Next() {
		var span Span
		var startStr, attrsStr string
		var durationNs int64
		var parentID, source, spanErr sql.NullString
		if err := rows.Scan(&span.TraceID, &span.SpanID, &parentID, &span.Name, &source,
			&startStr, &durationNs, &span.Status, &spanErr, &attrsStr); err != nil {
			return nil, err
		}
		span.ParentID = parentID.String
		span.Source = source.String
		span.Error = spanErr.String
		span.StartTime, _ = time.Parse(time.RFC3339Nano, startStr)
		span.Duration = time.Duration(durationNs)
		if attrsStr != "" && attrsStr != "{}" {
			json.Unmarshal([]byte(attrsStr), &span.Attributes)
		}
		spans = append(spans, span)
	}
	return spans, rows.Err()
}

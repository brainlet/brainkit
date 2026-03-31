package tracing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
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
	s := &SQLiteTraceStore{db: db, stopClean: make(chan struct{})}
	for _, opt := range opts {
		opt(s)
	}
	if s.retention > 0 {
		go s.cleanupLoop()
	}
	return s, nil
}

func (s *SQLiteTraceStore) cleanupLoop() {
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
	if s.retention > 0 {
		close(s.stopClean)
	}
	return nil
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

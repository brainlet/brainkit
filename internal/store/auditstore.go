package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit/internal/audit"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// SQLiteAuditStore implements audit.Store using database/sql with embedded schema.
type SQLiteAuditStore struct {
	db *sql.DB
}

func NewSQLiteAuditStore(path string) (*SQLiteAuditStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("auditstore: create directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("auditstore: open: %w", err)
	}
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA busy_timeout=5000")

	if _, err := db.Exec(sqliteSchemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("auditstore: create tables: %w", err)
	}

	return &SQLiteAuditStore{db: db}, nil
}

func (s *SQLiteAuditStore) Record(event audit.Event) {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	data := "{}"
	if event.Data != nil {
		data = string(event.Data)
	}

	_, err := s.db.Exec(
		`INSERT INTO audit_events (id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.Timestamp.Format(time.RFC3339Nano), event.Category, event.Type,
		event.Source, event.RuntimeID, event.Namespace, data, event.Duration.Nanoseconds(), event.Error,
	)
	if err != nil {
		slog.Warn("auditstore: record failed", slog.String("type", event.Type), slog.String("error", err.Error()))
	}
}

func (s *SQLiteAuditStore) Query(q audit.Query) ([]audit.Event, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}

	var query string
	var args []any

	switch {
	case q.Category != "":
		query = `SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg FROM audit_events WHERE category = ? ORDER BY created_at DESC LIMIT ?`
		args = []any{q.Category, limit}
	case q.Type != "":
		query = `SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg FROM audit_events WHERE event_type = ? ORDER BY created_at DESC LIMIT ?`
		args = []any{q.Type, limit}
	case q.Source != "":
		query = `SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg FROM audit_events WHERE source = ? ORDER BY created_at DESC LIMIT ?`
		args = []any{q.Source, limit}
	case !q.Since.IsZero():
		query = `SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg FROM audit_events WHERE created_at >= ? ORDER BY created_at DESC LIMIT ?`
		args = []any{q.Since.Format(time.RFC3339Nano), limit}
	default:
		query = `SELECT id, created_at, category, event_type, source, runtime_id, namespace, data, duration, error_msg FROM audit_events ORDER BY created_at DESC LIMIT ?`
		args = []any{limit}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []audit.Event
	for rows.Next() {
		var e audit.Event
		var ts, data string
		var dur int64
		if err := rows.Scan(&e.ID, &ts, &e.Category, &e.Type, &e.Source, &e.RuntimeID, &e.Namespace, &data, &dur, &e.Error); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		e.Duration = time.Duration(dur)
		e.Data = []byte(data)
		events = append(events, e)
	}
	if events == nil {
		events = []audit.Event{}
	}
	return events, rows.Err()
}

func (s *SQLiteAuditStore) Prune(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339Nano)
	_, err := s.db.Exec("DELETE FROM audit_events WHERE created_at < ?", cutoff)
	return err
}

func (s *SQLiteAuditStore) Count() (int64, error) {
	var count int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM audit_events").Scan(&count)
	return count, err
}

func (s *SQLiteAuditStore) CountByCategory() (map[string]int64, error) {
	rows, err := s.db.Query("SELECT category, COUNT(*) AS cnt FROM audit_events GROUP BY category")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int64)
	for rows.Next() {
		var cat string
		var cnt int64
		if err := rows.Scan(&cat, &cnt); err != nil {
			return nil, err
		}
		result[cat] = cnt
	}
	return result, rows.Err()
}

func (s *SQLiteAuditStore) Close() error {
	return s.db.Close()
}

package audit

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS audit_events (
	id         TEXT PRIMARY KEY,
	timestamp  TEXT NOT NULL,
	category   TEXT NOT NULL,
	type       TEXT NOT NULL,
	source     TEXT NOT NULL DEFAULT '',
	runtime_id TEXT NOT NULL DEFAULT '',
	namespace  TEXT NOT NULL DEFAULT '',
	data       TEXT NOT NULL DEFAULT '{}',
	duration   INTEGER NOT NULL DEFAULT 0,
	error      TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_category ON audit_events(category);
CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_events(type);
CREATE INDEX IF NOT EXISTS idx_audit_source ON audit_events(source);
`

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewSQLiteStore creates a new SQLite-backed audit store.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dsn := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db, logger: slog.Default()}, nil
}

func (s *SQLiteStore) Record(event Event) {
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
		`INSERT INTO audit_events (id, timestamp, category, type, source, runtime_id, namespace, data, duration, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.Timestamp.Format(time.RFC3339Nano),
		event.Category,
		event.Type,
		event.Source,
		event.RuntimeID,
		event.Namespace,
		data,
		event.Duration.Nanoseconds(),
		event.Error,
	)
	if err != nil {
		s.logger.Warn("audit: record failed", slog.String("type", event.Type), slog.String("error", err.Error()))
	}
}

func (s *SQLiteStore) Query(q Query) ([]Event, error) {
	query := `SELECT id, timestamp, category, type, source, runtime_id, namespace, data, duration, error
	          FROM audit_events WHERE 1=1`
	var args []any

	if q.Category != "" {
		query += ` AND category = ?`
		args = append(args, q.Category)
	}
	if q.Type != "" {
		query += ` AND type = ?`
		args = append(args, q.Type)
	}
	if q.Source != "" {
		query += ` AND source = ?`
		args = append(args, q.Source)
	}
	if q.RuntimeID != "" {
		query += ` AND runtime_id = ?`
		args = append(args, q.RuntimeID)
	}
	if !q.Since.IsZero() {
		query += ` AND timestamp >= ?`
		args = append(args, q.Since.Format(time.RFC3339Nano))
	}
	if !q.Until.IsZero() {
		query += ` AND timestamp <= ?`
		args = append(args, q.Until.Format(time.RFC3339Nano))
	}

	query += ` ORDER BY timestamp DESC`

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	query += ` LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var ts string
		var dur int64
		var data string
		if err := rows.Scan(&e.ID, &ts, &e.Category, &e.Type, &e.Source, &e.RuntimeID, &e.Namespace, &data, &dur, &e.Error); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		e.Duration = time.Duration(dur)
		e.Data = json.RawMessage(data)
		events = append(events, e)
	}
	if events == nil {
		events = []Event{}
	}
	return events, rows.Err()
}

func (s *SQLiteStore) Prune(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339Nano)
	_, err := s.db.Exec(`DELETE FROM audit_events WHERE timestamp < ?`, cutoff)
	return err
}

func (s *SQLiteStore) Count() (int64, error) {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM audit_events`).Scan(&count)
	return count, err
}

func (s *SQLiteStore) CountByCategory() (map[string]int64, error) {
	rows, err := s.db.Query(`SELECT category, COUNT(*) FROM audit_events GROUP BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int64)
	for rows.Next() {
		var cat string
		var count int64
		if err := rows.Scan(&cat, &count); err != nil {
			return nil, err
		}
		result[cat] = count
	}
	return result, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

package stores

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	pggen "github.com/brainlet/brainkit/internal/store/sqlgen/postgres"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// postgresSchema intentionally carries only the audit_events table; the
// KitStore's Postgres schema (deployments, schedules, plugins) lives in
// internal/store. The module owns its audit table lifecycle.
const postgresSchema = `
CREATE TABLE IF NOT EXISTS audit_events (
    id         TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    category   TEXT NOT NULL,
    event_type TEXT NOT NULL,
    source     TEXT NOT NULL DEFAULT '',
    runtime_id TEXT NOT NULL DEFAULT '',
    namespace  TEXT NOT NULL DEFAULT '',
    data       TEXT NOT NULL DEFAULT '{}',
    duration   BIGINT NOT NULL DEFAULT 0,
    error_msg  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_events(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_category   ON audit_events(category);
CREATE INDEX IF NOT EXISTS idx_audit_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_source     ON audit_events(source);
`

// Postgres persists audit events in a Postgres database using the shared
// sqlc-generated Queries. Ports the former internal/store/auditstore_pg.go
// into the module — schema/queries stay compatible with the pre-module
// wire format so upgrades don't need a migration.
type Postgres struct {
	db      *sql.DB
	queries *pggen.Queries
}

// NewPostgres opens a Postgres audit store. Creates the audit_events
// table if missing.
func NewPostgres(connStr string) (*Postgres, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("audit postgres: open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("audit postgres: ping: %w", err)
	}
	if _, err := db.Exec(postgresSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("audit postgres: create tables: %w", err)
	}
	return &Postgres{db: db, queries: pggen.New(db)}, nil
}

func (s *Postgres) Record(event auditpkg.Event) {
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
	err := s.queries.RecordAuditEvent(context.Background(), pggen.RecordAuditEventParams{
		ID:        event.ID,
		CreatedAt: event.Timestamp,
		Category:  event.Category,
		EventType: event.Type,
		Source:    event.Source,
		RuntimeID: event.RuntimeID,
		Namespace: event.Namespace,
		Data:      data,
		Duration:  event.Duration.Nanoseconds(),
		ErrorMsg:  event.Error,
	})
	if err != nil {
		slog.Warn("audit postgres: record failed", slog.String("type", event.Type), slog.String("error", err.Error()))
	}
}

func (s *Postgres) Query(q auditpkg.Query) ([]auditpkg.Event, error) {
	limit := int32(q.Limit)
	if limit <= 0 {
		limit = 100
	}
	var rows []pggen.AuditEvent
	var err error
	switch {
	case q.Category != "":
		rows, err = s.queries.QueryAuditEventsByCategory(context.Background(), pggen.QueryAuditEventsByCategoryParams{
			Category: q.Category, Limit: limit,
		})
	case q.Type != "":
		rows, err = s.queries.QueryAuditEventsByType(context.Background(), pggen.QueryAuditEventsByTypeParams{
			EventType: q.Type, Limit: limit,
		})
	case q.Source != "":
		rows, err = s.queries.QueryAuditEventsBySource(context.Background(), pggen.QueryAuditEventsBySourceParams{
			Source: q.Source, Limit: limit,
		})
	case !q.Since.IsZero():
		rows, err = s.queries.QueryAuditEventsSince(context.Background(), pggen.QueryAuditEventsSinceParams{
			CreatedAt: q.Since, Limit: limit,
		})
	default:
		rows, err = s.queries.QueryAuditEventsAll(context.Background(), limit)
	}
	if err != nil {
		return nil, err
	}
	events := make([]auditpkg.Event, len(rows))
	for i, r := range rows {
		events[i] = auditpkg.Event{
			ID: r.ID, Timestamp: r.CreatedAt, Category: r.Category, Type: r.EventType,
			Source: r.Source, RuntimeID: r.RuntimeID, Namespace: r.Namespace,
			Data: []byte(r.Data), Duration: time.Duration(r.Duration), Error: r.ErrorMsg,
		}
	}
	return events, nil
}

func (s *Postgres) Prune(olderThan time.Duration) error {
	return s.queries.PruneAuditEvents(context.Background(), time.Now().Add(-olderThan))
}

func (s *Postgres) Count() (int64, error) {
	return s.queries.CountAuditEvents(context.Background())
}

func (s *Postgres) CountByCategory() (map[string]int64, error) {
	rows, err := s.queries.CountAuditEventsByCategory(context.Background())
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Category] = r.Cnt
	}
	return result, nil
}

func (s *Postgres) Close() error { return s.db.Close() }

package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/store/sqlgen/postgres"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// PostgresAuditStore implements audit.Store using sqlc-generated Postgres queries.
type PostgresAuditStore struct {
	db      *sql.DB
	queries *pggen.Queries
}

// NewPostgresAuditStore creates a new Postgres-backed audit store.
func NewPostgresAuditStore(connStr string) (*PostgresAuditStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("auditstore-pg: open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("auditstore-pg: ping: %w", err)
	}

	schemaSQL, _ := postgresSchema()
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("auditstore-pg: create tables: %w", err)
	}

	return &PostgresAuditStore{db: db, queries: pggen.New(db)}, nil
}

func (s *PostgresAuditStore) Record(event audit.Event) {
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

	err := s.queries.RecordAuditEvent(ctx(), pggen.RecordAuditEventParams{
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
		slog.Warn("auditstore-pg: record failed", slog.String("type", event.Type), slog.String("error", err.Error()))
	}
}

func (s *PostgresAuditStore) Query(q audit.Query) ([]audit.Event, error) {
	limit := int32(q.Limit)
	if limit <= 0 {
		limit = 100
	}

	var rows []pggen.AuditEvent
	var err error

	switch {
	case q.Category != "":
		rows, err = s.queries.QueryAuditEventsByCategory(ctx(), pggen.QueryAuditEventsByCategoryParams{
			Category: q.Category, Limit: limit,
		})
	case q.Type != "":
		rows, err = s.queries.QueryAuditEventsByType(ctx(), pggen.QueryAuditEventsByTypeParams{
			EventType: q.Type, Limit: limit,
		})
	case q.Source != "":
		rows, err = s.queries.QueryAuditEventsBySource(ctx(), pggen.QueryAuditEventsBySourceParams{
			Source: q.Source, Limit: limit,
		})
	case !q.Since.IsZero():
		rows, err = s.queries.QueryAuditEventsSince(ctx(), pggen.QueryAuditEventsSinceParams{
			CreatedAt: q.Since, Limit: limit,
		})
	default:
		rows, err = s.queries.QueryAuditEventsAll(ctx(), limit)
	}

	if err != nil {
		return nil, err
	}

	events := make([]audit.Event, len(rows))
	for i, r := range rows {
		events[i] = audit.Event{
			ID: r.ID, Timestamp: r.CreatedAt, Category: r.Category, Type: r.EventType,
			Source: r.Source, RuntimeID: r.RuntimeID, Namespace: r.Namespace,
			Data: []byte(r.Data), Duration: time.Duration(r.Duration), Error: r.ErrorMsg,
		}
	}
	return events, nil
}

func (s *PostgresAuditStore) Prune(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return s.queries.PruneAuditEvents(ctx(), cutoff)
}

func (s *PostgresAuditStore) Count() (int64, error) {
	return s.queries.CountAuditEvents(ctx())
}

func (s *PostgresAuditStore) CountByCategory() (map[string]int64, error) {
	rows, err := s.queries.CountAuditEventsByCategory(ctx())
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Category] = r.Cnt
	}
	return result, nil
}

func (s *PostgresAuditStore) Close() error {
	return s.db.Close()
}

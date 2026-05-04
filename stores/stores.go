// Package stores provides optional KitStore constructors.
package stores

import (
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/stores/postgres"
	"github.com/brainlet/brainkit/stores/sqlite"
)

// KitStore provides persistence for deployments, schedules, and plugins.
type KitStore = types.KitStore

// SQLite implements KitStore using pure Go SQLite.
type SQLite = sqlite.Store

// Postgres implements KitStore using PostgreSQL.
type Postgres = postgres.Store

// NewSQLite creates a SQLite-backed KitStore at path.
func NewSQLite(path string) (*SQLite, error) {
	return sqlite.New(path)
}

// NewPostgres creates a PostgreSQL-backed KitStore for dsn.
func NewPostgres(dsn string) (KitStore, error) {
	return postgres.New(dsn)
}

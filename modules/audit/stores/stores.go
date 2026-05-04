// Package stores provides aggregate audit store constructors.
//
// Prefer modules/audit/stores/sqlite or modules/audit/stores/postgres when a
// binary wants to link only one backend driver.
package stores

import (
	"github.com/brainlet/brainkit/modules/audit/stores/postgres"
	"github.com/brainlet/brainkit/modules/audit/stores/sqlite"
)

// SQLite persists audit events in SQLite.
type SQLite = sqlite.Store

// Postgres persists audit events in PostgreSQL.
type Postgres = postgres.Store

// NewSQLite opens a SQLite audit store.
func NewSQLite(path string) (*SQLite, error) {
	return sqlite.New(path)
}

// NewPostgres opens a PostgreSQL audit store.
func NewPostgres(connStr string) (*Postgres, error) {
	return postgres.New(connStr)
}

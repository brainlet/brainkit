// Package postgres provides the PostgreSQL KitStore constructor without
// linking other KitStore backend drivers.
package postgres

import internalstore "github.com/brainlet/brainkit/internal/store/postgres"

// Store implements brainkit.KitStore using PostgreSQL.
type Store = internalstore.PostgresKitStore

// New creates a PostgreSQL-backed KitStore for dsn.
func New(dsn string) (*Store, error) {
	return internalstore.NewPostgresKitStore(dsn)
}

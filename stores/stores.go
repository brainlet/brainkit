// Package stores provides optional KitStore constructors.
package stores

import (
	internalstore "github.com/brainlet/brainkit/internal/store"
	"github.com/brainlet/brainkit/internal/types"
)

// KitStore provides persistence for deployments, schedules, and plugins.
type KitStore = types.KitStore

// SQLite implements KitStore using pure Go SQLite.
type SQLite = internalstore.SQLiteKitStore

// NewSQLite creates a SQLite-backed KitStore at path.
func NewSQLite(path string) (*SQLite, error) {
	return internalstore.NewSQLiteKitStore(path)
}

// NewPostgres creates a PostgreSQL-backed KitStore for dsn.
func NewPostgres(dsn string) (KitStore, error) {
	return internalstore.NewPostgresKitStore(dsn)
}

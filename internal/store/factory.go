package store

import (
	"fmt"

	"github.com/brainlet/brainkit/internal/store/postgres"
	"github.com/brainlet/brainkit/internal/store/sqlite"
	"github.com/brainlet/brainkit/internal/types"
)

// SQLiteKitStore is the SQLite KitStore backend.
type SQLiteKitStore = sqlite.SQLiteKitStore

// PostgresKitStore is the PostgreSQL KitStore backend.
type PostgresKitStore = postgres.PostgresKitStore

// NewSQLiteKitStore creates a SQLite-backed KitStore at path.
func NewSQLiteKitStore(path string) (*SQLiteKitStore, error) {
	return sqlite.NewSQLiteKitStore(path)
}

// NewPostgresKitStore creates a PostgreSQL-backed KitStore for connStr.
func NewPostgresKitStore(connStr string) (*PostgresKitStore, error) {
	return postgres.NewPostgresKitStore(connStr)
}

// Config configures the store backends.
type Config struct {
	// Backend is the storage engine: "sqlite" (default) or "postgres".
	Backend string

	// SQLitePath is the database file path (for sqlite backend).
	SQLitePath string

	// PostgresURL is the connection string (for postgres backend).
	PostgresURL string
}

// NewKitStore creates a KitStore from configuration.
func NewKitStore(cfg Config) (types.KitStore, error) {
	switch cfg.Backend {
	case "", "sqlite":
		if cfg.SQLitePath == "" {
			return nil, fmt.Errorf("store: sqlite requires SQLitePath")
		}
		return NewSQLiteKitStore(cfg.SQLitePath)
	case "postgres":
		if cfg.PostgresURL == "" {
			return nil, fmt.Errorf("store: postgres requires PostgresURL")
		}
		return NewPostgresKitStore(cfg.PostgresURL)
	default:
		return nil, fmt.Errorf("store: unknown backend %q (supported: sqlite, postgres)", cfg.Backend)
	}
}

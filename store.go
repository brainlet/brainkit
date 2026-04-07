package brainkit

import "github.com/brainlet/brainkit/internal/types"

// KitStore provides persistence for deployments, schedules, and plugins.
type KitStore = types.KitStore

// SQLiteStore implements KitStore using pure Go SQLite.
type SQLiteStore = types.SQLiteStore

// NewSQLiteStore creates a new SQLite-backed store at the given path.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	return types.NewSQLiteStore(path)
}

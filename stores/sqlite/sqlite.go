// Package sqlite provides the SQLite KitStore constructor without linking
// other KitStore backend drivers.
package sqlite

import internalstore "github.com/brainlet/brainkit/internal/store/sqlite"

// Store implements brainkit.KitStore using pure Go SQLite.
type Store = internalstore.SQLiteKitStore

// New creates a SQLite-backed KitStore at path.
func New(path string) (*Store, error) {
	return internalstore.NewSQLiteKitStore(path)
}

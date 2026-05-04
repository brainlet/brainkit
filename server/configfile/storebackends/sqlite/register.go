// Package sqlite registers the SQLite configfile KitStore backend.
package sqlite

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/server/configfile"
	storesqlite "github.com/brainlet/brainkit/stores/sqlite"
)

func init() {
	configfile.RegisterKitStore("sqlite", func(path string) (brainkit.KitStore, error) {
		return storesqlite.New(path)
	})
}

// Package storagebridges links optional runtime storage bridge backends.
//
// Import this package before using brainkit.SQLiteStorage or
// brainkit.SQLiteVector. The root brainkit package does not link libsql/sqlite
// by default.
package storagebridges

import (
	"github.com/brainlet/brainkit/internal/libsql"
	"github.com/brainlet/brainkit/modules/registry/storagehost"
)

func init() {
	storagehost.RegisterBridgeBuilder("sqlite", func(path string) (storagehost.Bridge, error) {
		return libsql.NewServer(path)
	})
}

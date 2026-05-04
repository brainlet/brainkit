// Package sqlite links Brainkit's SQLite/libsql runtime storage bridge.
package sqlite

import (
	"github.com/brainlet/brainkit/internal/libsql"
	"github.com/brainlet/brainkit/modulehost/storagehost"
)

func init() {
	storagehost.RegisterBridgeBuilder("sqlite", func(path string) (storagehost.Bridge, error) {
		return libsql.NewServer(path)
	})
}

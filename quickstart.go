package brainkit

import (
	"fmt"
	"path/filepath"
)

// QuickStart creates a fully-wired Kit with embedded NATS + SQLite persistence.
// Intended as the "batteries-included" path for interactive development and
// demos; library-embedded use should call New with an explicit Config.
//
// fsRoot must be an existing writable directory. QuickStart writes:
//   - <fsRoot>/nats-data/    (embedded NATS JetStream store)
//   - <fsRoot>/kit.db        (deployments + schedules + plugins)
//
// Tracing and the audit store are not wired here yet — session 05 attaches
// those via modules.
func QuickStart(namespace, fsRoot string) (*Kit, error) {
	if fsRoot == "" {
		return nil, fmt.Errorf("brainkit.QuickStart: fsRoot is required")
	}
	store, err := NewSQLiteStore(filepath.Join(fsRoot, "kit.db"))
	if err != nil {
		return nil, fmt.Errorf("brainkit.QuickStart: create store: %w", err)
	}
	return New(Config{
		Namespace: namespace,
		CallerID:  namespace,
		FSRoot:    fsRoot,
		Transport: EmbeddedNATS(),
		Store:     store,
		// TODO(session-05): attach tracing + audit modules here.
	})
}

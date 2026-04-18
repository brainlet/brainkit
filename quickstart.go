package brainkit

import (
	"fmt"
	"path/filepath"
)

// QuickStart creates a bare Kit wired with embedded NATS + SQLite
// persistence — the "batteries-included" library-mode helper for
// demos and interactive development. It does NOT compose the
// standard module set; that lives in `brainkit/server.QuickStart`,
// which returns a Server (Kit + gateway + probes + tracing + audit).
//
// fsRoot must be an existing writable directory. QuickStart writes:
//   - <fsRoot>/nats-data/    (embedded NATS JetStream store)
//   - <fsRoot>/kit.db        (deployments + schedules + plugins)
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
	})
}

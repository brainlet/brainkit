// Package audit is the brainkit.Module form of the audit log. The core
// Recorder (internal/audit) always runs and is nil-safe; this module
// attaches a persistent store and adds the audit.query / audit.stats /
// audit.prune bus commands.
//
// Usage:
//
//	store, _ := auditstores.NewSQLite(path)
//	kit, _ := brainkit.New(brainkit.Config{
//	    Modules: []brainkit.Module{
//	        audit.NewModule(audit.Config{Store: store}),
//	    },
//	})
//
// Without the module the Recorder no-ops and the audit.* bus commands
// are absent.
package audit

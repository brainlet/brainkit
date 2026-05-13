// Package schedules is the bkmodule.Module form of persisted cron-like
// scheduling. The kernel owns the QuickJS job pump (which must always run
// for Promise microtasks); this module owns user-level schedules — creation,
// firing, cancellation, and optional persistence via a Store.
//
// Usage:
//
//	store, _ := storesqlite.New(path) // also implements schedules.Store
//	kit, _ := brainkit.New(brainkit.Config{
//	    Store: store,
//	    Modules: []bkmodule.Module{
//	        schedules.NewModule(schedules.Config{Store: store}),
//	    },
//	})
//
// Without the module, .ts code that calls bus.schedule(...) receives a
// NOT_CONFIGURED error and the schedules.* bus commands are absent.
//
// Config-driven binaries that want `modules.schedules.path` to open a
// dedicated SQLite store should import modules/schedules/standard, or import
// server/standard/automation or server/standard/full for the built-in YAML
// profiles.
package schedules

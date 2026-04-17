// Package schedules is the brainkit.Module form of persisted cron-like
// scheduling. The kernel owns the QuickJS job pump (which must always run
// for Promise microtasks); this module owns user-level schedules — creation,
// firing, cancellation, and optional persistence via a Store.
//
// Usage:
//
//	store, _ := brainkit.NewSQLiteStore(path) // also implements schedules.Store
//	kit, _ := brainkit.New(brainkit.Config{
//	    Store: store,
//	    Modules: []brainkit.Module{
//	        schedules.NewModule(schedules.Config{Store: store}),
//	    },
//	})
//
// Without the module, .ts code that calls bus.schedule(...) receives a
// NOT_CONFIGURED error and the schedule.* bus commands are absent.
package schedules

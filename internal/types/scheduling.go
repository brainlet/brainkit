package types

import "context"

// ScheduleHandler is the narrow surface the QuickJS scheduling bridges and
// the schedules.* bus commands call into. The schedules module implements it
// (modules/schedules.Scheduler); the engine holds a reference via
// (*Kernel).scheduleHandler and dispatches bridge calls to it.
//
// When no handler is attached, the bridges throw NOT_CONFIGURED and the
// schedule.* bus commands are absent (they register via the module's Init).
type ScheduleHandler interface {
	Schedule(ctx context.Context, cfg ScheduleConfig) (string, error)
	Unschedule(ctx context.Context, id string) error
	List() []PersistedSchedule
}

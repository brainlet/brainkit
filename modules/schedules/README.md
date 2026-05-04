# modules/schedules — beta

Persisted cron-like scheduling — fires bus messages on a cadence.
Durable across Kit restarts via the configured `KitStore`.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    bkmodule "github.com/brainlet/brainkit/module"
    "github.com/brainlet/brainkit/modules/schedules"
)

store, _ := brainkit.NewSQLiteStore("./data/kit.db")

brainkit.New(brainkit.Config{
    Store: store,
    Modules: []bkmodule.Module{
        schedules.NewModule(schedules.Config{Store: store}),
    },
})
```

## Bus commands

- `schedules.create` — create a schedule (cron expr + topic +
  payload).
- `schedules.cancel` — cancel by ID.
- `schedules.list` — list active schedules.

## Runtime surface

Deployed `.ts` packages call `bus.schedule(expression, topic, data)`
directly — the module routes it through the same backing store.
Without the module wired, the JS bridge throws NOT_CONFIGURED.

## Capabilities

- Requires: `brainkit.core.schedule_handler_lease`.
- Uses when present: `brainkit.core.kit_store` as the durable schedule store,
  `brainkit.core.runtime_control` for drain-aware firing, and
  `brainkit.core.lifecycle_debug_registry`.
- Provides: schedule dispatch hook through the core schedule handler.

## Runtime resources

Owns `schedules.scheduler` and `schedule-handler`. When lifecycle debug is
available, it registers a scoped `schedules` component with live schedule,
timer, active fire, closing, store, and schedule handler lease counts.

## Hot unmount

Unmounting stops the scheduler, closes the schedule handler lease, unregisters
`schedules.*` commands, and drops runtime-control references. If the module was
built from YAML with its own `path`, that dedicated store is closed on unmount;
stores borrowed from Kit config remain owned by the Kit. Stores that implement
`CloseContext(context.Context)` receive the module unmount context.

Close is retryable. If the schedule handler lease cannot detach, the lease and
owned store remain attached for a later retry. If active schedule fires do not
join before the unmount context expires, the scheduler remains closed but the
owned store is kept open until a later close can join the fires. If an owned
store close fails, the store remains configured so the next close can retry it.
Lifecycle debug keeps reporting closing, active fires, store, and lease state
while teardown is in progress.

## Expressions

Standard cron (`*/5 * * * *`) plus "every N seconds/minutes/hours"
shorthand. See `schedules.ParseExpression` for the full grammar.

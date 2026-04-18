# modules/schedules — beta

Persisted cron-like scheduling — fires bus messages on a cadence.
Durable across Kit restarts via the configured `KitStore`.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/modules/schedules"
)

store, _ := brainkit.NewSQLiteStore("./data/kit.db")

brainkit.New(brainkit.Config{
    Store: store,
    Modules: []brainkit.Module{
        schedules.NewModule(schedules.Config{Store: store}),
    },
})
```

## Bus commands

- `schedule.create` — create a schedule (cron expr + topic +
  payload).
- `schedule.cancel` — cancel by ID.
- `schedule.list` — list active schedules.

## Runtime surface

Deployed `.ts` packages call `bus.schedule(expression, topic, data)`
directly — the module routes it through the same backing store.
Without the module wired, the JS bridge throws NOT_CONFIGURED.

## Expressions

Standard cron (`*/5 * * * *`) plus "every N seconds/minutes/hours"
shorthand. See `schedules.ParseExpression` for the full grammar.

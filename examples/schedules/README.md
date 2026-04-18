# schedules

Cron-style scheduled bus messages — wire
`modules/schedules`, deploy a `.ts` that ticks, schedule an
expression, collect ticks, cancel.

## Run

```sh
go run ./examples/schedules
```

Expected output:

```
scheduled heartbeat every 2s (id=78e16d28-…)
received tick 1
tick 1 at 2026-04-18T04:49:29.546Z source=heartbeat-demo.ts
received tick 2
…
received tick 3
cancelled schedule 78e16d28-…
```

## What it shows

- `modules/schedules.NewModule(Config{Store: store})` wires the
  cron engine, backed by the same `KitStore` used for deployments
  so schedules survive restarts.
- `brainkit.CallScheduleCreate(kit, ctx, ScheduleCreateMsg{
  Expression, Topic, Payload})` installs a schedule. The
  generated wrapper saturates the types so you don't write
  `Call[ScheduleCreateMsg, ScheduleCreateResp](...)`.
- `brainkit.CallScheduleCancel` cancels by ID.
- The `.ts` side also has `bus.schedule(expression, topic, data)`
  for in-deployment scheduling — returns a schedule ID that you
  unschedule with `bus.unschedule(id)`.

## Expression grammar

The scheduler accepts both compact shorthands and cron
expressions:

| Form | Example |
|---|---|
| Every N seconds | `every 30s` |
| Every N minutes | `every 5m` |
| Every N hours | `every 2h` |
| Cron | `0 */15 * * *` (every 15 min) |

See `modules/schedules/` for the parser — grammar is deliberately
conservative so the wire shape doesn't drift.

## Restart survival

The schedule table lives under the `KitStore`. Run this example
twice without cancelling — the second run picks up the existing
schedule and continues ticking from the stored timestamp. In
production, pair that with a persistent `FSRoot` so state lives
across deploys.

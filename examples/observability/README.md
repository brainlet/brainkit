# observability

Audit + tracing query surfaces: wire `modules/audit` +
`modules/tracing` on a Kit, generate events, query both stores,
pretty-print.

## Run

```sh
go run ./examples/observability
```

Expected output:

```
audit.query (last 20 events):
  TIMESTAMP                  CATEGORY  TYPE          SOURCE
  2026-04-18T00:54:15-04:00  deploy    kit.deployed  observability-demo.ts

audit.stats:
  total=1
  deploy: 1

trace.list (last 20):
  19e74889    ok  1 span(s)
  …
```

## What it shows

- `modules/audit.NewModule(Config{Store})` installs the
  audit store; every subsystem's `Record*` calls start
  persisting through `audit.query` / `audit.stats` / `audit.prune`.
- `modules/tracing.New(Config{Store})` installs the trace store
  and registers `trace.list` / `trace.get`. `Config.TraceSampleRate`
  on the Kit controls what gets sampled.
- Generated wrappers saturate the types: `brainkit.CallAuditQuery`,
  `brainkit.CallAuditStats`, `brainkit.CallTraceList`,
  `brainkit.CallTraceGet` — no type-parameter guessing.

## Filter cookbook

```go
// Only audit events from a specific deployment:
brainkit.CallAuditQuery(kit, ctx, sdk.AuditQueryMsg{
    Category: "deploy",
    Source:   "my-pkg.ts",
    Limit:    50,
})

// Traces slower than 500ms, errored only:
brainkit.CallTraceList(kit, ctx, sdk.TraceListMsg{
    Status:      "error",
    MinDuration: 500, // ms
    Limit:       20,
})
```

## Retention

`audit.prune` drops events older than `OlderThanHours`. Pair
with a schedule so old events roll off automatically:

```go
brainkit.CallScheduleCreate(kit, ctx, sdk.ScheduleCreateMsg{
    Expression: "0 4 * * *", // 4am daily
    Topic:      "audit.prune",
    Payload:    json.RawMessage(`{"olderThanHours":720}`), // 30 days
})
```

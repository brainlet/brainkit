# Observability

brainkit exposes observability through three surfaces:

1. **Logging** — tagged console output from `.ts` and the
   runtime, routed through `Config.LogHandler`.
2. **Audit** — `modules/audit` records every bus / runtime /
   lifecycle event into a pluggable store, queryable through the
   `audit.query` / `audit.stats` / `audit.prune` bus commands.
3. **Tracing** — `modules/tracing` records per-request spans into
   a pluggable trace store, queryable through `trace.list` /
   `trace.get`.

End-to-end example:
[`examples/observability/`](../../examples/observability/).

## Logging

Every `.ts` deployment gets a per-source tagged `console`:

```ts
console.log("starting up");
// [my-service.ts] [log] starting up
console.warn("slow query");
console.error("failed", err);
```

Override the default `log.Printf` sink:

```go
kit, err := brainkit.New(brainkit.Config{
    LogHandler: func(entry brainkit.LogEntry) {
        slog.Info(entry.Message,
            "source", entry.Source, // "my-service.ts" or "kernel"
            "level",  entry.Level,  // "log" | "warn" | "error" | "info" | "debug"
            "time",   entry.Time)
    },
})
```

`LogHandler` is called concurrently from multiple goroutines — keep
it safe.

## Audit

Wire `modules/audit` with a store. The shipped stores live in
`modules/audit/stores`:

```go
import (
    "github.com/brainlet/brainkit/modules/audit"
    auditstores "github.com/brainlet/brainkit/modules/audit/stores"
)

store, err := auditstores.NewSQLite("/var/lib/app/audit.db")
if err != nil { return err }

mod := audit.NewModule(audit.Config{
    Store:   store,
    Verbose: false, // true records every bus message; default is the normal tier.
})
```

Add the module to `Config.Modules`. From that point on:

- Every handler invocation, publish, deploy, teardown, tool call,
  plugin registration, and secret mutation records an event.
- The `audit.query`, `audit.stats`, `audit.prune` commands become
  available.

### Query

```go
resp, err := auditmsg.CallAuditQuery(kit, ctx,
    auditmsg.AuditQueryMsg{Limit: 20},
    sdk.WithCallTimeout(3*time.Second))

for _, e := range resp.Events {
    fmt.Println(e.Timestamp, e.Category, e.Type, e.Source)
}
```

Filter with the full `AuditQueryMsg` shape:

```go
auditmsg.AuditQueryMsg{
    Category:  "deploy",
    Type:      "handler.ok",
    Source:    "ts.my-service.ask",
    Since:     time.Now().Add(-1 * time.Hour),
    Until:     time.Now(),
    Limit:     500,
    RuntimeID: "runtime-1",
}
```

### Stats

```go
stats, err := auditmsg.CallAuditStats(kit, ctx,
    auditmsg.AuditStatsMsg{},
    sdk.WithCallTimeout(3*time.Second))

fmt.Printf("total=%d\n", stats.TotalEvents)
for cat, n := range stats.EventsByCategory {
    fmt.Printf("%s: %d\n", cat, n)
}
```

### Prune

```go
_, err = auditmsg.CallAuditPrune(kit, ctx,
    auditmsg.AuditPruneMsg{OlderThanHours: 24},
    sdk.WithCallTimeout(3*time.Second))
```

### Verbose tier

`audit.Config{Verbose: true}` records every bus message including
internal control frames. Runtime audit wiring is owned by
`modules/audit`; embedded callers configure it through the mounted
audit module rather than root Kit setters.

## Tracing

Tracing uses a separate module with its own store. For SQLite:

```go
import (
    "database/sql"
    "github.com/brainlet/brainkit/modules/tracing"
    "github.com/brainlet/brainkit/modules/tracing/tracingmsg"
    _ "modernc.org/sqlite"
)

db, err := sql.Open("sqlite", "/var/lib/app/traces.db")
if err != nil { return err }

store, err := tracing.NewSQLiteTraceStore(db)
if err != nil { return err }

mod := tracing.New(tracing.Config{Store: store})
```

Add to `Config.Modules`. `Config.TraceSampleRate` (0.0–1.0)
controls sampling; default 1.0.

### List traces

```go
resp, err := tracingmsg.CallTraceList(kit, ctx,
    tracingmsg.TraceListMsg{Limit: 20},
    sdk.WithCallTimeout(3*time.Second))

var traces []struct {
    TraceID   string `json:"traceId"`
    Name      string `json:"name"`
    Status    string `json:"status"`
    SpanCount int    `json:"spanCount"`
}
_ = json.Unmarshal(resp.Traces, &traces)
```

`resp.Traces` is a `json.RawMessage` — decode into the shape your
UI wants.

### Fetch a single trace

```go
t, err := tracingmsg.CallTraceGet(kit, ctx,
    tracingmsg.TraceGetMsg{TraceID: "abc123..."},
    sdk.WithCallTimeout(3*time.Second))
// t.Spans carries the span tree JSON.
```

## Combined wiring

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace:       "observability-demo",
    Transport:       brainkit.Memory(),
    FSRoot:          "/tmp/obs",
    TraceSampleRate: 1.0,
    Modules: []brainkit.Module{
        audit.NewModule(audit.Config{Store: auditStore}),
        tracing.New(tracing.Config{Store: traceStore}),
    },
})
```

Order matters for Init/Close — put `audit` before `tracing`, and
both before any module that should be observed.

## Bus commands

| Topic | Request | Response | Wrapper |
|---|---|---|---|
| `audit.query` | `auditmsg.AuditQueryMsg` | `auditmsg.AuditQueryResp` | `auditmsg.CallAuditQuery` |
| `audit.stats` | `auditmsg.AuditStatsMsg` | `auditmsg.AuditStatsResp` | `auditmsg.CallAuditStats` |
| `audit.prune` | `auditmsg.AuditPruneMsg` | `auditmsg.AuditPruneResp` | `auditmsg.CallAuditPrune` |
| `trace.list` | `tracingmsg.TraceListMsg` | `tracingmsg.TraceListResp` | `tracingmsg.CallTraceList` |
| `trace.get` | `tracingmsg.TraceGetMsg` | `tracingmsg.TraceGetResp` | `tracingmsg.CallTraceGet` |

## Probes

`modules/probes` exposes liveness / readiness probes on the HTTP
gateway or over the bus. Wire when the Kit fronts public traffic:

```go
import probesmod "github.com/brainkit/brainkit/modules/probes"

probesmod.NewModule(probesmod.Config{})
```

The `kit.health` bus command is module-owned by `modules/health`; its
typed helpers live in that package (`health.CallKitHealth`, etc.).
The probes module adds scheduled background probing of every provider /
vector store / storage backend.

## What's not shipped

- No HTTP `/metrics` endpoint. Watermill-level counts are internal.
- No OpenTelemetry exporter (the tracing store is the sole sink).
- No UI. The data is in the stores; point a viewer at SQLite for
  local inspection.

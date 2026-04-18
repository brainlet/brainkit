# modules/audit — beta

Query surface for the central audit log. The `Recorder` lives in
`internal/audit` and records events on every subsystem; this module
wires a backing store and exposes the query API over the bus.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/modules/audit"
    auditstores "github.com/brainlet/brainkit/modules/audit/stores"
)

store, _ := auditstores.NewSQLite("/var/brainkit/audit.db")

brainkit.New(brainkit.Config{
    Modules: []brainkit.Module{
        audit.NewModule(audit.Config{Store: store, Verbose: false}),
    },
})
```

## Bus commands

- `audit.query` → `AuditQueryResp` — list events filtered by type /
  category / time range / source.
- `audit.stats` → `AuditStatsResp` — counts bucketed by type +
  category.
- `audit.prune` → `AuditPruneResp` — delete events older than a
  cutoff.

## Stores

- `stores.SQLite` — embedded SQLite backing.
- `stores.Postgres` — Postgres backing built on the shared sqlc
  queries.

Without the module, the Recorder is a no-op and the audit.* bus
commands have no handler.

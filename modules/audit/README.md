# modules/audit — beta

Query surface for the central audit log. The `Recorder` lives in
`internal/audit` and records events on every subsystem; this module
wires a backing store and exposes the query API over the bus.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    bkmodule "github.com/brainlet/brainkit/module"
    "github.com/brainlet/brainkit/modules/audit"
    auditstores "github.com/brainlet/brainkit/modules/audit/stores"
)

store, _ := auditstores.NewSQLite("/var/brainkit/audit.db")

brainkit.New(brainkit.Config{
    Modules: []bkmodule.Module{
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

The `server/standard` YAML factory imports `modules/audit/standard`, which
wires the default SQLite/Postgres store constructors. Direct embedded callers
import only the stores package they need.

## Capabilities

- Requires: `brainkit.core.audit_store_lease`,
  `brainkit.core.audit_verbosity_lease`.
- Optional: `brainkit.core.lifecycle_debug_registry`.
- Provides: none.

## Runtime resources

Owns `audit.store`, the configured audit store attachment. The core recorder
remains in `internal/audit`; this module wires a backing store and query
commands.
When lifecycle debug is available, it registers a scoped `audit` component with
closing, store, ownership, and lease state.

## Hot unmount

Unmounting closes the audit store/verbosity leases, unregisters `audit.*`
commands, and closes the module-owned store when the store implements close.
Stores that implement `CloseContext(context.Context)` receive the module
unmount context.
If detaching the recorder store lease fails, the owned store is left open so
core cannot write into a closed store. If owned store close fails, the store
stays configured for retry and lifecycle inspection.

Without the module, the Recorder is a no-op and the audit.* bus
commands have no handler.

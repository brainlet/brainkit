# modules/tracing — beta

Persistent distributed-trace capture. Mount attaches the module's
store to the Kit's Tracer; Close detaches it. Cross-cutting spans
from bus handlers, plugins, workflow, etc. drop into the store.

## Usage

```go
import (
    "database/sql"

    "github.com/brainlet/brainkit"
    bkmodule "github.com/brainlet/brainkit/module"
    modtracing "github.com/brainlet/brainkit/modules/tracing"
    _ "modernc.org/sqlite"
)

db, _ := sql.Open("sqlite", "/var/brainkit/traces.db")
store, _ := modtracing.NewSQLiteTraceStore(db)

brainkit.New(brainkit.Config{
    Modules: []bkmodule.Module{
        modtracing.New(modtracing.Config{Store: store}),
    },
})
```

## Stores

- `NewSQLiteTraceStore(db)` — embedded SQLite backing; closing the store closes
  the owned database handle.

Core always has a Tracer — without a store wired, spans no-op.
The `server/standard/observability` and `server/standard/full` YAML profiles
import `modules/tracing/standard`, which opens the default SQLite-backed store.
Direct embedded callers import only the store driver they choose.

## Capabilities

- Requires: `brainkit.core.trace_store_lease`.
- Uses when present: `brainkit.core.lifecycle_debug_registry`.
- Provides: none.

## Runtime resources

Owns `tracing.store`, the attached persistent trace store. SQLite trace stores
with retention also own a cleanup loop. When lifecycle debug is available, it
registers a scoped `tracing` component with trace store, store type,
closeability, SQLite retention cleanup, closing, and trace store lease state.

## Hot unmount

Unmounting closes the trace store lease, unregisters
`trace.*` commands, and closes the store when it implements close. Stores that
implement `CloseContext(context.Context)` receive the module unmount context;
the built-in SQLite trace store uses that path to stop its retention cleanup
loop before closing the database handle.
If detaching the trace store lease fails, the store stays attached and open. If
store close fails after the lease detaches, the store stays tracked for retry
and lifecycle inspection.

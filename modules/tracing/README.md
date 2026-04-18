# modules/tracing — beta

Persistent distributed-trace capture. Init attaches the module's
store to the Kit's Tracer; Close detaches it. Cross-cutting spans
from bus handlers, plugins, workflow, etc. drop into the store.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    modtracing "github.com/brainlet/brainkit/modules/tracing"
)

db, _ := sql.Open("sqlite", "/var/brainkit/traces.db")
store, _ := modtracing.NewSQLiteTraceStore(db)

brainkit.New(brainkit.Config{
    TraceStore: store,
    Modules:    []brainkit.Module{modtracing.New(modtracing.Config{})},
})
```

## Stores

- `NewSQLiteTraceStore(db)` — embedded SQLite backing.
- `NewMemoryTraceStore(n)` — ring-buffered in-memory store (rooted
  in `internal/tracing`; useful for tests).

Core always has a Tracer — without a store wired, spans no-op.

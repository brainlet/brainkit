# Storage API

Brainkit embeds SQLite databases as LibSQL-compatible HTTP servers. No external database setup needed — just point to a file path.

## Quick Start

### Go — Configure storages at Kit creation

```go
kit, err := brainkit.New(brainkit.Config{
    Storages: map[string]StorageConfig{
        "default": { Path: "./data.db" },
    },
})
defer kit.Close() // stops all storage bridges
```

### TypeScript — Use storage in agents

```ts
import { agent, LibSQLStore } from "brainlet";

// Connects to Kit's "default" storage automatically
const store = new LibSQLStore({ id: "my-store" });

const myAgent = agent({
    model: "openai/gpt-4o-mini",
    instructions: "Remember everything.",
    memory: {
        thread: "session-1",
        storage: store,
    },
});
```

No URL, no HTTP endpoint to manage, no Docker container. The Kit handles it.

---

## Go API

### Config

```go
type StorageConfig struct {
    // Path to the SQLite database file. Created if it doesn't exist.
    // Use ":memory:" for an in-memory database (lost on close).
    Path string
}
```

### Kit Creation — `Storages` field

```go
kit, err := brainkit.New(brainkit.Config{
    Storages: map[string]StorageConfig{
        "default": { Path: "~/.myapp/data.db" },
        "vectors": { Path: "~/.myapp/vectors.db" },
    },
})
```

- Each entry starts an embedded HTTP server backed by a SQLite file
- The server speaks the Hrana v2/v3 pipeline protocol (what `@libsql/client` expects)
- Pure Go — uses `modernc.org/sqlite`, no CGo, no external binaries
- Auto-assigns a free port on `127.0.0.1`

### Runtime Management

```go
// Add a storage at runtime (immediately available to JS)
err := kit.AddStorage("analytics", StorageConfig{
    Path: "./analytics.db",
})

// Remove a storage (stops the HTTP bridge, JS calls will fail)
err := kit.RemoveStorage("analytics")

// Get the HTTP URL for a storage (for diagnostics or external access)
url := kit.StorageURL("default") // "http://127.0.0.1:54321"
```

### Lifecycle

All storages are closed automatically when `kit.Close()` is called. The SQLite files persist on disk — restarting the Kit with the same path picks up where it left off.

---

## TypeScript API

### LibSQLStore

```ts
import { LibSQLStore } from "brainlet";
```

#### Brainkit mode — auto-connects to Kit's embedded storage

```ts
// Uses the "default" storage (or the first one if no "default" exists)
const store = new LibSQLStore({ id: "my-store" });

// Uses a named storage
const store = new LibSQLStore({ id: "my-store", storage: "vectors" });
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | yes | Store identifier |
| `storage` | `string` | no | Named storage from Kit config. Defaults to `"default"` or the first available. |

#### Mastra mode — explicit URL (passthrough)

```ts
// Remote Turso database
const store = new LibSQLStore({
    id: "my-store",
    url: "libsql://my-db.turso.io",
    authToken: "...",
});

// Any LibSQL-compatible HTTP endpoint
const store = new LibSQLStore({
    id: "my-store",
    url: "http://localhost:8080",
});
```

When `url` is provided, the config is passed directly to Mastra's `LibSQLStore` unchanged. All Mastra options (`authToken`, `maxRetries`, `initialBackoffMs`, `disableInit`) work as documented.

### LibSQLVector

Same pattern as `LibSQLStore`:

```ts
import { LibSQLVector } from "brainlet";

// Brainkit mode
const vectors = new LibSQLVector({ id: "my-vectors" });
const vectors = new LibSQLVector({ id: "my-vectors", storage: "vectors" });

// Mastra mode
const vectors = new LibSQLVector({ id: "my-vectors", url: "http://..." });
```

---

## Multiple Storages

Use separate databases for different concerns:

```go
kit, err := brainkit.New(brainkit.Config{
    Storages: map[string]StorageConfig{
        "default": { Path: "./data.db" },      // agent memory, workflows, traces
        "vectors": { Path: "./vectors.db" },    // vector embeddings
        "scratch": { Path: ":memory:" },        // temporary, lost on restart
    },
})
```

```ts
import { agent, LibSQLStore, LibSQLVector } from "brainlet";

const memory = new LibSQLStore({ id: "mem" });                        // → data.db
const vectors = new LibSQLVector({ id: "vecs", storage: "vectors" }); // → vectors.db
const temp = new LibSQLStore({ id: "tmp", storage: "scratch" });      // → in-memory

const myAgent = agent({
    model: "openai/gpt-4o-mini",
    memory: {
        thread: "session-1",
        storage: memory,
        vector: vectors,
        embedder: { model: "openai/text-embedding-3-small", provider: "openai" },
    },
});
```

---

## How It Works

```
Kit.New(Storages: {"default": {Path: "./data.db"}})
  │
  ├─ Starts libsql.Server on 127.0.0.1:<auto-port>
  │    └─ modernc.org/sqlite opens ./data.db (WAL mode, 5s busy timeout)
  │
  ├─ Injects into JS: globalThis.__brainkit_storages = { "default": "http://127.0.0.1:54321" }
  │
  └─ JS: new LibSQLStore({ id: "x" })
       └─ Wrapper sees no url → resolves from __brainkit_storages["default"]
       └─ Creates real Mastra LibSQLStore({ id: "x", url: "http://127.0.0.1:54321" })
            └─ @libsql/client HTTP mode → Hrana protocol over fetch
                 └─ Hits our Go bridge → SQLite via modernc.org/sqlite
```

The Go bridge speaks the Hrana pipeline protocol — the same wire protocol that Turso's `sqld` and `@libsql/client` use. This means:

- **Mastra's LibSQLStore works unmodified** — it thinks it's talking to a normal LibSQL HTTP server
- **All Mastra migrations run correctly** — table creation, column checks, index management
- **Transactions work** — via the Hrana baton mechanism
- **Batch operations work** — including `store_sql` cached statements

---

## Persistence

| Config | Behavior |
|--------|----------|
| `Path: "./data.db"` | Persistent. Survives Kit restarts. File created automatically. |
| `Path: "/absolute/path/data.db"` | Same, with absolute path. Parent directory created automatically. |
| `Path: ":memory:"` | In-memory only. Lost when the storage is removed or Kit is closed. |

The SQLite file uses WAL journal mode for better concurrent read performance.

---

## Error Handling

```ts
// If no storage is configured and no url is provided:
const store = new LibSQLStore({ id: "x" });
// → Error: LibSQLStore: no url provided and no Kit storage 'default' configured.
//          Either pass { url } or add Storages to Kit config.

// If a named storage doesn't exist:
const store = new LibSQLStore({ id: "x", storage: "nonexistent" });
// → Error: LibSQLStore: no url provided and no Kit storage 'nonexistent' configured.
```

```go
// Adding a storage that already exists:
err := kit.AddStorage("default", StorageConfig{Path: ":memory:"})
// → error: brainkit: storage "default" already exists

// Removing a storage that doesn't exist:
err := kit.RemoveStorage("nonexistent")
// → error: brainkit: storage "nonexistent" not found
```

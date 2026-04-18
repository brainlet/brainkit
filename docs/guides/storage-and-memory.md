# Storage and Memory

brainkit exposes storage backends as a named map on `Config`.
Deployed `.ts` code resolves them by name through `storage("name")`
and hands the result to Mastra classes (`Memory`, `LibSQLStore`,
`PostgresStore`, etc.) and to workflow snapshot persistence.

## Wire storages from Go

```go
import (
    "path/filepath"
    "github.com/brainlet/brainkit"
)

kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: brainkit.Memory(),
    FSRoot:    "/var/lib/my-app",
    Storages: map[string]brainkit.StorageConfig{
        "default": brainkit.SQLiteStorage(filepath.Join("/var/lib/my-app", "kv.db")),
    },
})
```

Five storage builders ship out of the box:

| Builder | Backing store |
|---|---|
| `brainkit.SQLiteStorage(path)` | SQLite via embedded libsql bridge. `":memory:"` works. |
| `brainkit.PostgresStorage(dsn)` | Postgres via `pg` driver over jsbridge. |
| `brainkit.MongoDBStorage(uri, dbName)` | MongoDB via `node-mongodb-native` driver. |
| `brainkit.UpstashStorage(url, token)` | Upstash REST storage. |
| `brainkit.InMemoryStorage()` | Ephemeral in-process, lost on close. |

Manage the registry at runtime via `kit.Storages()`:

```go
kit.Storages().Register("scratch",
    brainkit.StorageType("sqlite"),
    map[string]any{"path": ":memory:"})

for _, s := range kit.Storages().List() {
    fmt.Println(s.Name, s.Type)
}

kit.Storages().Unregister("scratch")
```

## Access from `.ts`

```ts
import { storage, Memory, LibSQLStore } from "kit";

const store = new LibSQLStore({ id: "chat-threads" });
const mem   = new Memory({ storage: store });
```

`new LibSQLStore({ id })` auto-connects to the Kit's embedded
SQLite bridge; no URL or credentials needed. Use `storage("name")`
to resolve a specific backend by its Go-side name:

```ts
const pg = storage("pg");
const mem = new Memory({ storage: pg });
```

Multiple backends:

```go
Storages: map[string]brainkit.StorageConfig{
    "default": brainkit.SQLiteStorage("/var/lib/app/kv.db"),
    "vectors": brainkit.SQLiteStorage("/var/lib/app/vectors.db"),
    "cache":   brainkit.SQLiteStorage(":memory:"),
},
```

## Agent memory

`Memory` stores messages, observations, and reflections keyed by
`threadId` and `resourceId`.

```ts
const mem = new Memory({ storage: new LibSQLStore({ id: "chat" }) });

const chat = new Agent({
    name: "chatter",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Remember what the user told you.",
    memory: mem,
});

await chat.generate("My name is David", {
    threadId:   "thread-1",
    resourceId: "user-1",
});

const r = await chat.generate("What's my name?", {
    threadId:   "thread-1",
    resourceId: "user-1",
});
// r.text should reference "David"
```

`threadId` scopes the conversation; `resourceId` scopes the owner
(a user, a session, whatever you need to separate histories on).
Messages on the same `threadId` are automatically replayed into
subsequent `generate` / `stream` calls.

Swap in Postgres / Mongo / Upstash by changing the storage
resolver:

```ts
const mem = new Memory({ storage: new PostgresStore({ id: "pg-mem" }) });
const mem = new Memory({ storage: new MongoDBStore({ id: "mongo-mem" }) });
const mem = new Memory({ storage: storage("cache") });
```

## Mastra Memory surface

The Memory class exposes methods that work across every storage
backend:

```ts
await mem.saveThread({ thread: {
    id: "t-1",
    title: "first thread",
    resourceId: "demo",
    createdAt: new Date(),
    updatedAt: new Date(),
}});
const t = await mem.getThreadById({ threadId: "t-1" });
await mem.saveMessages({ messages: [...] });
const msgs = await mem.query({ threadId: "t-1", limit: 10 });
```

See [`examples/storage-vectors/`](../../examples/storage-vectors/)
for a program that deploys a `.ts` handler that saves and retrieves
a thread through `Memory` over SQLite storage:

```ts
const store = storage("default");
const mem   = new Memory({ storage: store });

bus.on("put", async (msg) => {
    const thread = {
        id: msg.payload.id,
        title: msg.payload.title,
        resourceId: "demo",
        createdAt: new Date(),
        updatedAt: new Date(),
    };
    await mem.saveThread({ thread });
    msg.reply({ saved: msg.payload.id });
});

bus.on("get", async (msg) => {
    const thread = await mem.getThreadById({ threadId: msg.payload.id });
    msg.reply({ found: thread !== null, thread });
});
```

## Workflow snapshot persistence

Whenever `Config.Storages` is non-empty, brainkit promotes Mastra's
internal storage from the in-memory fallback to a real backend
during init. This makes workflow state durable:

- `workflow.status` reads from storage — survives Kit restart.
- Suspended workflows survive restarts; `workflow.resume` works on
  the new Kit.
- `restartActiveWorkflows` on startup picks up `running` /
  `waiting` runs from the previous process.

No manual step is required — declaring a storage is enough. See
[`examples/workflows/`](../../examples/workflows/).

## Auth matrix

| Backend | Protocol | Auth methods exercised |
|---|---|---|
| SQLite (libsql bridge) | HTTP (Hrana v2/v3) | Embedded + token over container |
| Postgres | TCP | SCRAM-SHA-256, md5, trust |
| MongoDB | TCP | SCRAM-SHA-256, SCRAM-SHA-1, no-auth |
| Upstash | HTTP | Token |

All tested against real infrastructure; no mocks. SCRAM-SHA-256
and Postgres CRAM paths use brainkit's jsbridge polyfills
(`net.Socket` → Go `net.Conn`, WebCrypto `subtle.deriveBits` for
scramming). See the knowledge base in `../brainkit-maps/knowledge/`
for specific test results.

## Choosing a backend

| Scenario | Pick |
|---|---|
| Local development | `SQLiteStorage("./data.db")` — zero setup, persistent. |
| Unit tests | `InMemoryStorage()` or `SQLiteStorage(":memory:")`. |
| Single-node production | SQLite via bridge, or Postgres. |
| Multi-node production | Postgres, or remote Turso (`LibSQLStore` with URL). |
| Existing MongoDB infra | `MongoDBStorage`. |
| Serverless / edge | `UpstashStorage`. |

Vector stores live in `Config.Vectors` — see
[vectors-and-rag.md](vectors-and-rag.md).

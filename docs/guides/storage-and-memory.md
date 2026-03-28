# Storage and Memory

brainkit supports 5 storage backends and 3 vector backends for agent memory, conversation threads, and embeddings. Storage can be embedded (Go-managed SQLite) or external (Postgres, MongoDB, Upstash).

## Embedded Storage (LibSQL Bridge)

The simplest option — brainkit starts a Go HTTP server backed by a local SQLite file. No containers, no connection strings, no setup:

```go
k, err := kit.NewKernel(kit.KernelConfig{
    Storages: map[string]kit.StorageConfig{
        "default": kit.SQLiteStorage("./data.db"),
    },
})
```

Inside .ts code, `new LibSQLStore({ id: "my-store" })` auto-connects to the embedded server — no URL needed. The bridge speaks the Hrana v2/v3 pipeline protocol (same wire format as Turso's `sqld`), so Mastra's `@libsql/client` HTTP mode works unmodified.

The bridge server (`internal/libsql/server.go`):
- Pure Go — `modernc.org/sqlite`, no CGo
- WAL mode + 5s busy timeout
- Auto-assigned port on `127.0.0.1`
- Supports transactions (baton mechanism), batch operations, SQL caching
- Creates parent directories automatically

### Multiple storages

```go
Storages: map[string]kit.StorageConfig{
    "default": kit.SQLiteStorage("./data.db"),      // memory, workflows, traces
    "vectors": kit.SQLiteStorage("./vectors.db"),    // vector embeddings
    "scratch": kit.SQLiteStorage(":memory:"),        // ephemeral, lost on close
},
```

```typescript
const memory = new LibSQLStore({ id: "mem" });                         // → data.db
const vectors = new LibSQLVector({ id: "vecs", storage: "vectors" });  // → vectors.db
const temp = new LibSQLStore({ id: "tmp", storage: "scratch" });       // → in-memory
```

### Runtime management

```go
k.AddStorage("analytics", kit.SQLiteStorage("./analytics.db"))
k.RemoveStorage("analytics")
url := k.StorageURL("default") // "http://127.0.0.1:54321"
```

## Five Storage Providers

| Provider | Constructor | Protocol | Auth Methods Tested |
|----------|------------|----------|---------------------|
| InMemoryStore | `new InMemoryStore()` | N/A | N/A |
| LibSQLStore | `new LibSQLStore({id, url?, authToken?})` | HTTP (Hrana) | embedded + container |
| PostgresStore | `new PostgresStore({id, connectionString})` | TCP | SCRAM-SHA-256, md5, trust |
| MongoDBStore | `new MongoDBStore({id, uri, dbName})` | TCP | SCRAM-SHA-256, SCRAM-SHA-1, no-auth |
| UpstashStore | `new UpstashStore({id, url, token})` | HTTP | token auth |

All tested with real infrastructure — no mocks. Auth matrix in `test/auth/auth_test.go`.

### Go-side provider registry

```go
Storages: map[string]kit.StorageConfig{
    "default": kit.InMemoryStorage(),
    "pg":      kit.PostgresStorage("postgres://..."),
},
```

Then in .ts: `const store = storage("pg");` — resolves from the Go registry, creates a real `PostgresStore` instance.

## Memory with Agents

Mastra's `Memory` class provides conversation thread management on top of any storage backend:

```typescript
// fixtures/ts/agent/with-memory-inmemory/index.ts
const store = new InMemoryStore();
const mem = new Memory({ storage: store });

const agent = new Agent({
    name: "memory-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Remember the user's name.",
    memory: mem,
});

kit.register("agent", "memory-agent", agent);

await agent.generate("My name is David", {
    threadId: "thread-1",
    resourceId: "user-1",
});

const result = await agent.generate("What's my name?", {
    threadId: "thread-1",
    resourceId: "user-1",
});
// result.text contains "David"
```

Memory auto-saves messages to the thread on every `generate`/`stream` call. Messages are recalled automatically when the same `threadId` is used.

### With Postgres

```typescript
// fixtures/ts/agent/with-memory-postgres/index.ts
const store = new PostgresStore({
    id: "pg-mem",
    connectionString: process.env.DATABASE_URL,
});
const mem = new Memory({ storage: store });

const agent = new Agent({
    name: "pg-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You remember everything.",
    memory: mem,
});
```

PostgresStore uses the `pg` npm driver through brainkit's jsbridge polyfills (net.Socket → Go `net.Conn`, crypto → Go `crypto`). SCRAM-SHA-256 auth works through WebCrypto (`crypto.subtle.deriveBits`).

### With MongoDB

```typescript
// fixtures/ts/agent/with-memory-mongodb/index.ts
const store = new MongoDBStore({
    id: "mongo-mem",
    uri: process.env.MONGODB_URI,
    dbName: "brainkit",
});
const mem = new Memory({ storage: store });
```

MongoDBStore uses the `node-mongodb-native` driver through jsbridge polyfills. SCRAM-SHA-256 auth works through Node.js crypto path (`crypto.pbkdf2Sync` + `crypto.createHmac`).

## Three Vector Providers

| Provider | Constructor | Protocol |
|----------|------------|----------|
| LibSQLVector | `new LibSQLVector({id, connectionUrl?, authToken?})` | HTTP (Hrana) |
| PgVector | `new PgVector({id, connectionString})` | TCP |
| MongoDBVector | `new MongoDBVector({id, uri, dbName})` | TCP |

### Go-side registration

```go
Vectors: map[string]kit.VectorConfig{
    "main": kit.PgVectorStore(pgConnStr),
},
```

Then in .ts: `const vs = vectorStore("main");` — creates a real `PgVector` instance.

### Vector operations

```typescript
// fixtures/ts/vector/pgvector-methods/index.ts
const vs = new PgVector({
    id: "test-vectors",
    connectionString: process.env.PG_VECTOR_URL,
});

// Create index
await vs.createIndex("docs", 1536, "cosine");

// Upsert
await vs.upsert("docs", [
    { id: "doc-1", values: embedding1, metadata: { title: "Getting Started" } },
    { id: "doc-2", values: embedding2, metadata: { title: "API Reference" } },
]);

// Query
const results = await vs.query("docs", queryEmbedding, 5);
// results: [{id, score, metadata}, ...]
```

## Observational Memory

LibSQLStore, PostgresStore, and MongoDBStore support 3-tier observational memory:
1. **Messages** → raw conversation turns
2. **Observations** → compressed summaries extracted from messages
3. **Reflections** → higher-level patterns extracted from observations

InMemoryStore and UpstashStore support basic memory (threads + messages) but not the observational compression pipeline.

## Choosing a Provider

| Use Case | Recommended |
|----------|-------------|
| Development | Embedded LibSQL (zero setup, persistent) |
| CI/testing | InMemoryStore (fast, no cleanup) |
| Single-node production | Embedded LibSQL or PostgresStore |
| Multi-node production | PostgresStore or remote Turso |
| Existing MongoDB infra | MongoDBStore |
| Serverless/edge | UpstashStore (HTTP-only) |
| Need observational memory | LibSQL, Postgres, or MongoDB only |

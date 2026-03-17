# Storage in Brainkit

Brainkit uses storage for agent memory (conversation threads, messages), workflow snapshots, observability traces, vector embeddings, and more. This guide covers what's supported, how to choose, and how to use each provider.

---

## Two Modes

### Embedded (brainkit-managed)

Brainkit starts an embedded SQLite database backed by a Hrana HTTP bridge. No external setup. No Docker. No connection strings. Just a file path.

```go
kit, _ := brainkit.New(brainkit.Config{
    Storages: map[string]StorageConfig{
        "default": { Path: "./data.db" },
    },
})
```

```ts
const store = new LibSQLStore({ id: "my-store" });
// That's it. Connects to Kit's embedded SQLite automatically.
```

This is the recommended starting point. It works for development, testing, and single-node production deployments.

### External (mastra-managed)

Pass a `url` to connect to any LibSQL-compatible, PostgreSQL, MongoDB, Upstash, or other supported backend. Brainkit passes the config through to Mastra unchanged.

```ts
const store = new LibSQLStore({ id: "my-store", url: "libsql://my-db.turso.io", authToken: "..." });
const store = new PostgresStore({ connectionString: "postgres://..." });
const store = new MongoDBStore({ url: "mongodb://...", dbName: "myapp" });
```

Use this when you need a managed database, horizontal scaling, or specific infrastructure requirements.

---

## Memory Storage Providers

Memory storage handles conversation threads, messages, working memory, and observational memory. These are the backends Mastra's `Memory` class uses.

### Supported

| Provider | Import | Embedded | Full OM | Use Case |
|----------|--------|----------|---------|----------|
| **InMemoryStore** | `from "kit"` | N/A | no | Development, testing. Data lost on restart. |
| **LibSQLStore** | `from "kit"` | **yes** | **yes** | Default choice. Local SQLite file or remote Turso. |
| **PostgresStore** | `from "kit"` | no | **yes** | Production with existing Postgres infrastructure. |
| **MongoDBStore** | `from "kit"` | no | **yes** | Production with existing MongoDB infrastructure. |
| **UpstashStore** | `from "kit"` | no | no | Serverless Redis. HTTP-only, no TCP needed. |

**Full OM** = supports observational memory (3-tier compression: messages → observations → reflections). Requires resource-scoped message listing, which only LibSQL, Postgres, and MongoDB implement.

### Not Yet Supported

| Provider | Package | Protocol | Why Not Yet |
|----------|---------|----------|-------------|
| ClickHouse | `@mastra/clickhouse` | HTTP | Niche analytics use case. Low demand. |
| Cloudflare D1 | `@mastra/cloudflare-d1` | HTTP | Edge-only. Would work through fetch if bundled. |
| Cloudflare Workers | `@mastra/cloudflare` | HTTP | Edge-only. Same as above. |
| Convex | `@mastra/convex` | HTTP | Managed service. Would work through fetch if bundled. |
| DynamoDB | `@mastra/dynamodb` | HTTP | AWS-specific. Would work through fetch if bundled. |
| Lance | `@mastra/lance` | Embedded | Needs native binding or Go bridge. |
| MSSQL | `@mastra/mssql` | TCP | Needs TCP socket polyfill (same path as Postgres). |

The unsupported HTTP-based providers (ClickHouse, Cloudflare, Convex, DynamoDB) would likely work if their npm packages were added to the bundle — they communicate over HTTP which our fetch polyfill handles. They're not bundled to keep the bundle size manageable (currently 16.5MB).

MSSQL would need the GoSocket TCP polyfill path like Postgres and MongoDB, which is proven but requires integration testing.

---

## Vector Storage Providers

Vector storage handles embeddings for semantic recall, RAG, and similarity search.

### Supported

| Provider | Import | Embedded | Use Case |
|----------|--------|----------|----------|
| **LibSQLVector** | `from "kit"` | **yes** | Default choice. Same SQLite file as memory storage. |
| **PgVector** | `from "kit"` | no | Production Postgres with pgvector extension. |
| **MongoDBVector** | `from "kit"` | no | MongoDB Atlas Vector Search (Atlas-only feature). |

### Not Yet Supported

| Provider | Package | Protocol | Why Not Yet |
|----------|---------|----------|-------------|
| Pinecone | `@mastra/pinecone` | HTTP | Managed service. Would work through fetch. |
| Qdrant | `@mastra/qdrant` | HTTP | Self-hostable. Good candidate for next addition. |
| Chroma | `@mastra/chroma` | HTTP | Self-hostable. Would work through fetch. |
| Astra | `@mastra/astra` | HTTP | DataStax managed. |
| Upstash Vector | `@mastra/upstash` | HTTP | Serverless. Would work through fetch. |
| Cloudflare Vectorize | `@mastra/vectorize` | HTTP | Edge-only. |
| OpenSearch | `@mastra/opensearch` | HTTP/TCP | AWS-managed or self-hosted. |
| Elasticsearch | `@mastra/elasticsearch` | HTTP | Self-hosted or Elastic Cloud. |
| Couchbase | `@mastra/couchbase` | TCP | Needs TCP polyfill. |
| DuckDB | `@mastra/duckdb` | Embedded | Needs native binding or Go bridge. |
| LanceDB | `@mastra/lance` | Embedded | Needs native binding or Go bridge. |
| S3 Vectors | `@mastra/s3vectors` | HTTP | AWS S3-based. |
| Turbopuffer | `@mastra/turbopuffer` | HTTP | Managed service. |

Most unsupported vector providers use HTTP APIs and would work through fetch if bundled. The embedded ones (DuckDB, LanceDB) would need a Go bridge similar to what we built for LibSQL.

---

## Choosing a Provider

### For Development

```go
// In-memory — fastest, no files, data lost on restart
kit, _ := brainkit.New(brainkit.Config{})
```
```ts
const store = new InMemoryStore({ id: "dev" });
```

Or with persistence:

```go
// Embedded SQLite — persistent, no setup
kit, _ := brainkit.New(brainkit.Config{
    Storages: map[string]StorageConfig{
        "default": { Path: "./dev.db" },
    },
})
```
```ts
const store = new LibSQLStore({ id: "dev" });
```

### For Production (Self-Hosted)

```go
// Embedded SQLite for single-node deployments
kit, _ := brainkit.New(brainkit.Config{
    Storages: map[string]StorageConfig{
        "default": { Path: "/var/lib/brainlet/data.db" },
    },
})
```

Or with external Postgres:

```ts
const store = new PostgresStore({
    connectionString: process.env.DATABASE_URL,
});
```

### For Production (Cloud/Multi-Node)

```ts
// Remote Turso (LibSQL cloud)
const store = new LibSQLStore({
    id: "prod",
    url: "libsql://my-db.turso.io",
    authToken: process.env.TURSO_AUTH_TOKEN,
});

// Or PostgreSQL
const store = new PostgresStore({
    connectionString: process.env.DATABASE_URL,
});
```

### Decision Matrix

| Scenario | Recommended | Why |
|----------|-------------|-----|
| Getting started | Embedded LibSQL | Zero setup, persistent |
| Local development | Embedded LibSQL | File-based, survives restarts |
| CI/CD testing | InMemoryStore | Fast, no cleanup needed |
| Single-node production | Embedded LibSQL | Simple, performant, WAL mode |
| Multi-node production | PostgresStore or Turso | Shared state across nodes |
| Need observational memory | LibSQL, Postgres, or MongoDB | Only these 3 support full OM |
| Existing Postgres infra | PostgresStore | Reuse what you have |
| Existing MongoDB infra | MongoDBStore | Reuse what you have |
| Serverless/edge | UpstashStore | HTTP-only, no connection pooling |

---

## Observational Memory Compatibility

Observational memory (3-tier compression) requires a storage backend that supports:
1. Resource-scoped message listing (query messages across threads for the same resource)
2. Observation/reflection table storage
3. Working memory per-resource

Only **LibSQLStore**, **PostgresStore**, and **MongoDBStore** implement the full interface. Using observational memory with InMemoryStore or UpstashStore will result in the observer/reflector not finding messages across threads.

```ts
// Works — full observational memory
const store = new LibSQLStore({ id: "om" });
const mem = createMemory({
    storage: store,
    options: {
        observationalMemory: { enabled: true },
    },
});

// Partial — basic memory works, OM observer won't see cross-thread messages
const store = new UpstashStore({ url: "...", token: "..." });
```

---

## Embedded Storage Details

When using `Storages` in Kit config, brainkit starts a Go HTTP server per storage entry. Each server:

- Uses `modernc.org/sqlite` — pure Go SQLite, no CGo, no external binaries
- Speaks the Hrana v2/v3 pipeline protocol (same as Turso's `sqld`)
- Listens on `127.0.0.1` with an auto-assigned port
- Enables WAL journal mode and 5s busy timeout
- Supports transactions (via Hrana baton mechanism)
- Supports batch operations with conditional execution
- Supports SQL caching (`store_sql`/`close_sql`)
- Creates parent directories automatically

### Multiple Named Storages

Separate databases for different concerns:

```go
kit, _ := brainkit.New(brainkit.Config{
    Storages: map[string]StorageConfig{
        "default": { Path: "./data.db" },
        "vectors": { Path: "./vectors.db" },
        "scratch": { Path: ":memory:" },
    },
})
```

```ts
const memory = new LibSQLStore({ id: "mem" });                        // → data.db
const vectors = new LibSQLVector({ id: "vecs", storage: "vectors" }); // → vectors.db
const temp = new LibSQLStore({ id: "tmp", storage: "scratch" });      // → in-memory
```

### Runtime Storage Management

```go
// Add a storage after Kit creation
kit.AddStorage("analytics", StorageConfig{ Path: "./analytics.db" })

// Remove it (bridge stops, JS calls will error)
kit.RemoveStorage("analytics")

// Get the URL (for diagnostics)
url := kit.StorageURL("default") // "http://127.0.0.1:54321"
```

### Persistence

| Path | Behavior |
|------|----------|
| `"./data.db"` | Persistent file, relative to working directory |
| `"/absolute/path/data.db"` | Persistent file, absolute path |
| `":memory:"` | In-memory only, lost when storage is removed or Kit closes |

---

## Bundle Impact

Each storage provider adds to the JavaScript bundle size. Currently bundled:

| Provider | Bundle Contribution |
|----------|-------------------|
| InMemoryStore | ~0 (part of `@mastra/core`) |
| LibSQLStore + LibSQLVector | ~200KB (`@libsql/client` HTTP mode) |
| PostgresStore + PgVector | ~400KB (`pg` driver + TCP polyfill) |
| MongoDBStore + MongoDBVector | ~600KB (`mongodb` driver + TCP polyfill) |
| UpstashStore | ~50KB (HTTP-only) |

Total storage-related bundle: ~1.25MB of the 16.5MB bundle.

Adding more providers would increase bundle size. This is why cloud-specific providers (Pinecone, Cloudflare, DynamoDB) are not bundled by default — most users don't need them, and each would add 100-500KB.

---

## Testing

All supported providers are tested with real infrastructure:

| Provider | Test Method |
|----------|-------------|
| InMemoryStore | Unit test (no infra) |
| LibSQLStore (embedded) | Embedded bridge + real SQLite file |
| LibSQLStore (remote) | Testcontainer (`ghcr.io/tursodatabase/libsql-server`) |
| PostgresStore (trust) | Testcontainer (Postgres) |
| PostgresStore (SCRAM) | Testcontainer (Postgres with SCRAM-SHA-256) |
| MongoDBStore | Testcontainer (MongoDB) |
| UpstashStore | Real Upstash Redis (HTTP REST API) |
| PgVector | Testcontainer (Postgres + pgvector) |
| MongoDBVector | Testcontainer (MongoDB) |
| LibSQLVector | Testcontainer or embedded bridge |

No mocks. Every test hits a real database.

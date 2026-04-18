# storage-vectors

Persistent KV (Mastra Memory backed by SQLite) + vector store
demo from a deployed `.ts` package. One Kit, two backends behind
the `storage()` and `vectorStore()` runtime APIs.

The example picks its vector backend at runtime:

| `PGVECTOR_URL` env var | Backend used |
|---|---|
| set (docker-compose shipped) | `PgVectorStore` — full similarity search |
| unset (default) | `SQLiteVector` — KV works; vectors gracefully skip when `libsql_vector_idx` isn't in the embedded build |

## Quick run (SQLite path)

```sh
export OPENAI_API_KEY=sk-...      # embeddings need a real provider
go run ./examples/storage-vectors
```

`KV round-trip` completes end-to-end. The vector similarity demo
reports a clean skip when the embedded libsql build lacks the
vector-index functions — no crash, no silent miss.

## Full run with pgvector (docker-compose)

A one-command local Postgres with the `vector` extension is
shipped alongside this example. Stand it up, point the example
at it, and the similarity demo runs for real:

```sh
cd examples/storage-vectors
docker compose up -d              # Postgres 16 + pgvector on localhost:5433
export OPENAI_API_KEY=sk-...
export PGVECTOR_URL="postgres://brainkit:brainkit@127.0.0.1:5433/brainkit?sslmode=disable"
go run .                          # or: go run ./examples/storage-vectors from repo root
docker compose down -v            # tear it down when done
```

Expected output on the full path:

```
vector backend: pgvector (postgres://brainkit:brainkit@127.0.0.1:5433/brainkit?sslmode=disable)
KV round-trip:
{"found":true,"thread":{"id":"t-1","resourceId":"demo","title":"first thread", …}}

vector similarity round-trip:
{"hits":[{"id":"d1","score":0.87,"text":"brainkit is an embeddable runtime…"}, …]}
```

## What it shows

- `brainkit.Config.Storages` + `brainkit.Config.Vectors` wire
  named backends the `.ts` resolves through `storage(name)` and
  `vectorStore(name)`.
- The `.ts` side uses `Memory` (from `agent`) for thread
  persistence and `vectorStore().createIndex` /
  `.upsert` / `.query` for similarity search.
- Embeddings come from `embeddingModel(provider, modelID)` —
  same provider registry the `ai-chat` example uses.
- Backend choice is an env var away — the `.ts` code never
  changes when you swap SQLite ↔ pgvector ↔ MongoDB.

## Backend swap cookbook

Any Mastra-supported vector store drops in by changing the
`Vectors` map entry:

```go
Vectors: map[string]brainkit.VectorConfig{
    "default": brainkit.PgVectorStore(os.Getenv("PGVECTOR_URL")),
    // or MongoDBVectorStore(uri, dbName)
    // or SQLiteVector(path)      — default, no infra
},
```

Matching storage backends: `brainkit.PostgresStorage`,
`brainkit.MongoDBStorage`, `brainkit.UpstashStorage`,
`brainkit.InMemoryStorage`.

## docker-compose contents

- `pgvector/pgvector:pg16` image with the vector extension
  available by default.
- `initdb/01-enable-vector.sql` runs `CREATE EXTENSION vector`
  on first boot so the example doesn't have to.
- Host port `5433` chosen to avoid clashing with any local
  Postgres on `5432`. Change the mapping if you need a different
  port.
- Data persists in the `pgvector_data` volume between runs; add
  `-v` to `docker compose down` to wipe.

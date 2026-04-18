# storage-vectors

Persistent KV (Mastra Memory backed by SQLite) + vector store
demo from a deployed `.ts` package. One Kit, two backends behind
the `storage()` and `vectorStore()` runtime APIs.

## Run

```sh
export OPENAI_API_KEY=sk-...      # embeddings require a real key
go run ./examples/storage-vectors
```

Expected output on a fresh run (OpenAI key set):

```
KV round-trip:
{"found":true,"thread":{"id":"t-1","resourceId":"demo","title":"first thread", …}}

vector similarity round-trip:
{"hits":[{"id":"d1","score":0.87,"text":"brainkit is an embeddable runtime…"}, …]}
```

Without `OPENAI_API_KEY`, the KV half still runs; the vector
half reports that it was skipped (embeddings need a real
provider).

## What it shows

- `brainkit.Config.Storages` + `brainkit.Config.Vectors` wire
  named backends the `.ts` resolves through `storage(name)` and
  `vectorStore(name)`.
- The `.ts` side uses `Memory` (from `agent`) for thread
  persistence and `vectorStore().createIndex` /
  `.upsert` / `.query` for similarity search.
- Embeddings come from `embeddingModel(provider, modelID)` —
  same provider registry the `ai-chat` example uses.

## Backend swap cookbook

SQLite is fine for demos but short of production. Postgres drops
in for both surfaces:

```go
Storages: map[string]brainkit.StorageConfig{
    "default": brainkit.PostgresStorage(os.Getenv("DATABASE_URL")),
},
Vectors: map[string]brainkit.VectorConfig{
    "default": brainkit.PgVectorStore(os.Getenv("PGVECTOR_URL")),
},
```

MongoDB is a one-line swap too — see `brainkit.MongoDBStorage`
and `brainkit.MongoDBVectorStore` on the root package.

## Note on libsql vector extensions

The shipped libsql build may not include the vector-index
functions (`libsql_vector_idx`). The example handles the miss by
printing a clear skip message rather than aborting — the KV path
is unaffected. Use `pgvector` for production vector workloads.

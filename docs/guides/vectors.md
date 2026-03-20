# Vectors

Vector storage handles embeddings for semantic recall, RAG, and similarity search. The bus handlers route through EvalTS to the configured vector store API. See the [storage guide](storage.md) for supported vector store providers.

> **Prerequisite**: A vector store must be configured. Deploy code that creates a vector store instance (e.g., `new LibSQLVector(...)`) and assigns it to `globalThis.__kit_vector_store`.

---

## Bus Topics

| Topic | Payload | Response |
|-------|---------|----------|
| `vectors.createIndex` | `{"name":"docs","dimension":1536,"metric":"cosine"}` | `{"ok":true}` |
| `vectors.listIndexes` | `{}` | `[{"name":"docs","dimension":1536}]` |
| `vectors.upsert` | `{"index":"docs","vectors":[...]}` | `{"ok":true}` |
| `vectors.query` | `{"index":"docs","embedding":[...],"topK":5}` | `{"matches":[...]}` |
| `vectors.deleteIndex` | `{"name":"docs"}` | `{"ok":true}` |

---

## Creating an Index

Indexes define the vector space. Each index has a name, dimension (must match your embedding model's output size), and distance metric.

### Metrics

| Metric | Use case |
|--------|----------|
| `cosine` | Most common. Good for text embeddings (OpenAI, Cohere). |
| `euclidean` | When magnitude matters (spatial data, raw features). |
| `dotProduct` | Normalized vectors where speed matters. |

```go
bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic: "vectors.createIndex",
    Payload: json.RawMessage(`{
        "name": "docs",
        "dimension": 1536,
        "metric": "cosine"
    }`),
})
```

---

## Upserting Vectors

Insert or update vectors by ID. Each vector has an ID, a float array, and optional metadata.

```go
bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic: "vectors.upsert",
    Payload: json.RawMessage(`{
        "index": "docs",
        "vectors": [
            {
                "id": "doc-1",
                "values": [0.1, 0.2, 0.3],
                "metadata": {"title": "Getting Started"}
            },
            {
                "id": "doc-2",
                "values": [0.4, 0.5, 0.6],
                "metadata": {"title": "API Reference"}
            }
        ]
    }`),
})
```

---

## Querying

Find the most similar vectors to a query embedding. Returns matches sorted by score.

```go
resp, _ := bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic: "vectors.query",
    Payload: json.RawMessage(`{
        "index": "docs",
        "embedding": [0.1, 0.2, 0.3],
        "topK": 5
    }`),
})
// resp.Payload: {"matches":[{"id":"doc-1","score":0.99,"metadata":{"title":"Getting Started"}},...]}
```

The `filter` field is optional and depends on the vector store implementation.

---

## Usage from .ts

```typescript
import { bus } from "kit";

// Create index
await bus.ask("vectors.createIndex", {
    name: "knowledge",
    dimension: 1536,
    metric: "cosine",
});

// Upsert
await bus.ask("vectors.upsert", {
    index: "knowledge",
    vectors: [
        { id: "chunk-1", values: embedding, metadata: { source: "docs.md" } },
    ],
});

// Query
const { matches } = await bus.ask("vectors.query", {
    index: "knowledge",
    embedding: queryEmbedding,
    topK: 10,
});
```

---

## Usage from Plugins (via SDK)

```go
// Create index
sdk.Ask[any](client, ctx, messages.VectorCreateIndexMsg{
    Name:      "docs",
    Dimension: 1536,
    Metric:    "cosine",
}, func(_ any, err error) {
    if err != nil { log.Fatal(err) }
})

// Upsert
client.Ask(ctx, messages.VectorUpsertMsg{
    Index: "docs",
    Vectors: []messages.Vector{
        {ID: "doc-1", Values: []float64{0.1, 0.2, 0.3}, Metadata: map[string]string{"title": "Intro"}},
    },
}, func(msg messages.Message) {})

// Query
sdk.Ask[messages.VectorQueryResp](client, ctx, messages.VectorQueryMsg{
    Index:     "docs",
    Embedding: queryVec,
    TopK:      5,
}, func(resp messages.VectorQueryResp, err error) {
    for _, m := range resp.Matches {
        fmt.Printf("  %s (%.3f)\n", m.ID, m.Score)
    }
})
```

---

## Supported Providers

| Provider | Import | Embedded | Use case |
|----------|--------|----------|----------|
| **LibSQLVector** | `from "kit"` | yes | Default. Same SQLite file as memory storage. |
| **PgVector** | `from "kit"` | no | Production Postgres with pgvector extension. |
| **MongoDBVector** | `from "kit"` | no | MongoDB Atlas Vector Search. |

See the [storage guide](storage.md) for setup and configuration details.

---

## Lifecycle

```
vectors.createIndex(name, dimension, metric) --> index created
vectors.upsert(index, vectors) --> vectors stored
vectors.query(index, embedding, topK) --> matches returned
vectors.deleteIndex(name) --> index and all vectors removed
vectors.listIndexes() --> all index metadata returned
```

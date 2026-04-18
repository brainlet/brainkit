# Vectors and RAG

Vector stores are declared alongside storage on `Config.Vectors`
and resolved from `.ts` through `vectorStore("name")`. The bundled
Mastra RAG toolkit (MDocument, chunkers, rerank, GraphRAG,
`createVectorQueryTool`) operates on the resolved store — no
separate setup.

## Wire vector stores from Go

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "rag-demo",
    Transport: brainkit.Memory(),
    FSRoot:    "/var/lib/rag",
    Vectors: map[string]brainkit.VectorConfig{
        "default": brainkit.SQLiteVector("/var/lib/rag/vectors.db"),
    },
    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
    },
})
```

Three constructors ship out of the box:

| Builder | Backend |
|---|---|
| `brainkit.SQLiteVector(path)` | libsql / SQLite via embedded bridge. |
| `brainkit.PgVectorStore(dsn)` | Postgres with the `pgvector` extension. |
| `brainkit.MongoDBVectorStore(uri, dbName)` | MongoDB Atlas Vector Search. |

Runtime management:

```go
for _, v := range kit.Vectors().List() {
    fmt.Println(v.Name, v.Type)
}
```

## Resolve from `.ts`

```ts
import { vectorStore, embeddingModel, MDocument, LibSQLVector } from "kit";

// Registry-backed
const vs = vectorStore("default");

// Direct instantiation — resolves to the Kit's embedded SQLite bridge
const vs2 = new LibSQLVector({ id: "docs" });

// Remote libsql / Turso
const vs3 = new LibSQLVector({ id: "docs", url: process.env.LIBSQL_URL });
```

## Index lifecycle

The vector API uses object arguments everywhere. Positional
arguments are not supported.

```ts
await vs.createIndex({ indexName: "docs", dimension: 1536 });

await vs.upsert({
    indexName: "docs",
    vectors:   [embedding1, embedding2],
    ids:       ["doc-1", "doc-2"],
    metadata:  [
        { title: "Getting Started" },
        { title: "API Reference" },
    ],
});

const hits = await vs.query({
    indexName:   "docs",
    queryVector: queryEmbedding,
    topK:        5,
});
// hits: [{ id, score, metadata }, ...]

const indexes = await vs.listIndexes();
await vs.describeIndex("docs");
await vs.deleteVectors({ indexName: "docs", ids: ["doc-1"] });
await vs.deleteIndex("docs");
```

## End-to-end example

From [`examples/storage-vectors/`](../../examples/storage-vectors/):

```ts
const store = vectorStore("default");
const embed = embeddingModel("openai", "text-embedding-3-small");

let indexReady = false;
async function ensureIndex(dim) {
    if (indexReady) return;
    await store.createIndex({ indexName: "demo", dimension: dim });
    indexReady = true;
}

bus.on("seed", async (msg) => {
    const docs = msg.payload.docs;
    const { embeddings } = await embed.doEmbed({ values: docs.map((d) => d.text) });
    await ensureIndex(embeddings[0].length);
    await store.upsert({
        indexName: "demo",
        vectors:   embeddings,
        ids:       docs.map((d) => d.id),
        metadata:  docs.map((d) => ({ text: d.text })),
    });
    msg.reply({ inserted: docs.length });
});

bus.on("query", async (msg) => {
    const { embeddings } = await embed.doEmbed({ values: [msg.payload.query] });
    const hits = await store.query({
        indexName:   "demo",
        queryVector: embeddings[0],
        topK:        msg.payload.k || 2,
    });
    msg.reply({ hits: hits.map((h) => ({
        id: h.id, score: h.score, text: h.metadata.text,
    })) });
});
```

The Go side publishes `seed` then `query`. Similarity search is
transport-backed — the Go caller never touches the vectors
directly.

## Document processing with MDocument

```ts
const doc = MDocument.fromText(rawText);
const chunks = await doc.chunk({
    strategy: "recursive",
    maxSize:  500,
    overlap:  50,
});
```

Other strategies:

| Strategy | Source | Notes |
|---|---|---|
| `"recursive"` | `MDocument.fromText(...)` | Basic character splitting. |
| `"markdown"` | `MDocument.fromMarkdown(...)` | Splits on headings, preserves header metadata. |
| `"token"` | `MDocument.fromText(...)` | Splits by token count (js-tiktoken). |

Each chunk carries its source text (`chunk.text`) and optional
structure metadata.

## createVectorQueryTool

Wraps a vector store as an agent tool. The agent calls the tool
when it needs semantic retrieval; your code doesn't do the
query/embedding plumbing.

```ts
const queryTool = createVectorQueryTool({
    vectorStoreName: "default",
    indexName:       "docs",
    model:           embeddingModel("openai", "text-embedding-3-small"),
});

const agent = new Agent({
    name:         "rag-agent",
    model:        model("openai", "gpt-4o-mini"),
    instructions: "Use the vector query tool to find relevant information.",
    tools:        { vectorQuery: queryTool },
});
```

## Reranking

Post-retrieval reranking:

```ts
const reranked = await rerank(results, query, {
    model: model("openai", "gpt-4o-mini"),
    topK:  3,
});
```

Custom scorer:

```ts
const reranked = await rerankWithScorer(results, query, {
    scorer: async (result, q) =>
        result.metadata.source === "trusted" ? 1.0 : 0.5,
    topK: 3,
});
```

## GraphRAG

`GraphRAG` extracts entities and relationships, stores them in a
vector store, and retrieves over the resulting knowledge graph.

```ts
const graphRag = new GraphRAG({
    model:          model("openai", "gpt-4o-mini"),
    vectorStore:    vectorStore("default"),
    indexName:      "graph",
    embeddingModel: embeddingModel("openai", "text-embedding-3-small"),
});

await graphRag.addDocuments([
    MDocument.fromText("Alice works at Acme Corp. Bob is Alice's manager."),
]);

const results = await graphRag.query("Who does Alice work with?");
```

Wrap as an agent tool:

```ts
const graphTool = createGraphRAGTool({
    graphRag,
    description: "Query the knowledge graph for entity relationships.",
});
```

## End-to-end RAG pipeline

The full chunk → embed → upsert → query-tool → agent loop is
broken out as [`examples/rag-pipeline/`](../../examples/rag-pipeline/),
compose file + seed corpus + positive / negative evaluation
included. Highlights worth reading inline:

```ts
// 1. Chunk. maxSize small enough that each fact lives in one chunk.
const doc    = MDocument.fromText(text, { docId });
const chunks = await doc.chunk({ strategy: "recursive", maxSize: 400, overlap: 40 });

// 2. Embed + upsert (same as the End-to-end example above).

// 3. Retrieval tool — use the DIRECT-instance shape inside brainkit
//    deployments. The `vectorStoreName: "..."` shape routes through
//    `mastra.getVector(name)`, which isn't wired inside the SES
//    compartment — prefer `vectorStore:`.
const queryTool = createVectorQueryTool({
    vectorStore: vectorStore("docs"),
    indexName:   "docs",
    model:       embeddingModel("openai", "text-embedding-3-small"),
});

const agent = new Agent({
    name:         "rag-bot",
    model:        model("openai", "gpt-4o-mini"),
    instructions: "Answer using the vector query tool. If the tool output doesn't contain the answer, decline.",
    tools:        { queryTool },
});
```

See [`examples/storage-vectors/`](../../examples/storage-vectors/)
for the raw upsert + query path without an Agent, and
[`examples/rag-pipeline/`](../../examples/rag-pipeline/) for the
full agent-driven flow with reranking.

## What's tested

Vector and RAG paths are exercised against real backends (libsql,
pgvector, MongoDB). See fixture inventory under
`fixtures/ts/vector/` and `fixtures/ts/rag/` — any feature with a
fixture there is tested in CI. Features that exist in Mastra but
aren't covered by a fixture should be treated as unverified.

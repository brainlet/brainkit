# Vectors and RAG

brainkit bundles Mastra's RAG toolkit — vector stores, document processing, chunking, reranking, and GraphRAG. All operations run inside the QuickJS runtime via the "agent" module.

## Vector Stores

Three providers, all tested with real containers:

```typescript
// LibSQLVector — embedded or remote
const vs = new LibSQLVector({ id: "docs" });  // embedded bridge
const vs = new LibSQLVector({ id: "docs", connectionUrl: "http://libsql-server:8080" }); // remote

// PgVector — requires Postgres with pgvector extension
const vs = new PgVector({ id: "docs", connectionString: "postgres://..." });

// MongoDBVector — requires MongoDB Atlas Vector Search
const vs = new MongoDBVector({ id: "docs", uri: "mongodb://...", dbName: "myapp" });
```

### Index lifecycle

```typescript
// fixtures/ts/vector/libsql-create-upsert-query/index.ts
await vs.createIndex("knowledge", 1536, "cosine");

await vs.upsert("knowledge", [
    { id: "chunk-1", values: embedding, metadata: { source: "docs.md", page: 1 } },
    { id: "chunk-2", values: embedding2, metadata: { source: "api.md", page: 3 } },
]);

const results = await vs.query("knowledge", queryEmbedding, 5);
// [{id: "chunk-1", score: 0.95, metadata: {source: "docs.md"}}, ...]

const indexes = await vs.listIndexes();
await vs.deleteIndex("knowledge");
```

### From Go registry

```go
Vectors: map[string]brainkit.VectorConfig{
    "main": brainkit.PgVectorStore(pgURL),
},
```

```typescript
const vs = vectorStore("main"); // resolves from Go registry
```

## Document Processing

`MDocument` processes documents into chunks for embedding:

```typescript
// fixtures/ts/rag/chunk-text/index.ts
import { MDocument } from "agent";

const doc = MDocument.fromText("Long document text here...");
const chunks = await doc.chunk({
    strategy: "recursive",
    size: 500,
    overlap: 50,
});
// chunks: MDocument[] — each chunk is a smaller document
```

### Chunking strategies

```typescript
// Text — basic character splitting
await doc.chunk({ strategy: "recursive", size: 1000, overlap: 100 });

// Markdown — splits on headings, preserves structure
// fixtures/ts/rag/chunk-markdown/index.ts
const doc = MDocument.fromMarkdown("# Title\n\nContent here...");
await doc.chunk({ strategy: "markdown", size: 500 });

// Token — splits by token count (uses tiktoken)
// fixtures/ts/rag/chunk-token/index.ts
await doc.chunk({ strategy: "token", size: 200, overlap: 20 });
```

## createVectorQueryTool

Wraps a vector store as an agent tool:

```typescript
// fixtures/ts/rag/vector-query/index.ts
import { createVectorQueryTool } from "agent";

const queryTool = createVectorQueryTool({
    vectorStoreName: "main",        // name from vectorStore() registry
    indexName: "knowledge",
    model: embeddingModel("openai", "text-embedding-3-small"),
});

const agent = new Agent({
    name: "rag-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Use the vector query tool to find relevant information.",
    tools: { vectorQuery: queryTool },
});
```

## Reranking

After vector retrieval, rerank results by relevance:

```typescript
// fixtures/ts/rag/rerank/index.ts
import { rerank } from "agent";

const reranked = await rerank(results, query, {
    model: model("openai", "gpt-4o-mini"),
    topK: 3,
});
```

`rerankWithScorer` uses a custom scorer function:

```typescript
import { rerankWithScorer } from "agent";

const reranked = await rerankWithScorer(results, query, {
    scorer: async (result, query) => {
        // Custom relevance scoring logic
        return result.metadata.source === "trusted" ? 1.0 : 0.5;
    },
    topK: 3,
});
```

## GraphRAG

Knowledge graph extraction + retrieval:

```typescript
// fixtures/ts/rag/graph-rag/index.ts
import { GraphRAG, createGraphRAGTool } from "agent";

const graphRag = new GraphRAG({
    model: model("openai", "gpt-4o-mini"),
    vectorStore: vectorStore("main"),
    indexName: "graph",
    embeddingModel: embeddingModel("openai", "text-embedding-3-small"),
});

// Add documents — extracts entities and relationships
await graphRag.addDocuments([
    MDocument.fromText("Alice works at Acme Corp. Bob is Alice's manager."),
]);

// Query — traverses the knowledge graph
const results = await graphRag.query("Who does Alice work with?");
```

`createGraphRAGTool` wraps GraphRAG as an agent tool:

```typescript
const graphTool = createGraphRAGTool({
    graphRag,
    description: "Query the knowledge graph for entity relationships",
});

const agent = new Agent({
    tools: { graph: graphTool },
    // ...
});
```

## Document Chunker Tool

Wraps document chunking as an agent tool:

```typescript
import { createDocumentChunkerTool } from "agent";

const chunkerTool = createDocumentChunkerTool({
    strategy: "markdown",
    size: 500,
    overlap: 50,
});

// Agent can process documents on demand
const agent = new Agent({
    tools: { chunker: chunkerTool },
    // ...
});
```

## End-to-End RAG Pattern

```typescript
// 1. Process documents into chunks
const doc = MDocument.fromMarkdown(rawMarkdown);
const chunks = await doc.chunk({ strategy: "markdown", size: 500 });

// 2. Embed and store chunks
const vs = vectorStore("main");
await vs.createIndex("docs", 1536, "cosine");

for (const chunk of chunks) {
    const { embedding } = await embed({
        model: embeddingModel("openai", "text-embedding-3-small"),
        value: chunk.getText(),
    });
    await vs.upsert("docs", [{
        id: crypto.randomUUID(),
        values: embedding,
        metadata: { text: chunk.getText() },
    }]);
}

// 3. Create agent with vector query tool
const queryTool = createVectorQueryTool({
    vectorStoreName: "main",
    indexName: "docs",
    model: embeddingModel("openai", "text-embedding-3-small"),
});

const agent = new Agent({
    name: "rag-bot",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Answer questions using the vector query tool to find relevant context.",
    tools: { search: queryTool },
});

const result = await agent.generate("What does the documentation say about deployment?");
```

# rag/ Fixtures

Tests the RAG (Retrieval-Augmented Generation) pipeline: document chunking strategies, MDocument parsing, rerank function availability, GraphRAG availability, and end-to-end vector query with embed+upsert+search.

## Fixtures

### chunk/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| markdown | no | none | `MDocument.fromMarkdown()` + `doc.chunk({ strategy: "markdown" })` splits by headers (#, ##, ###); produces multiple chunks with header metadata |
| text | no | none | `MDocument.fromText()` + `doc.chunk({ strategy: "recursive", maxSize: 200, overlap: 20 })` splits long text into multiple overlapping chunks |
| token | no | none | `MDocument.fromText()` + `doc.chunk({ strategy: "token", maxSize: 50, overlap: 10 })` splits by token count using js-tiktoken |

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| document-chunker-tool | no | none | `MDocument.fromText()` creates a document object; verifies it is non-null |
| graph-rag | no | none | Verifies `GraphRAG` constructor is available as a globalThis endowment from kit_runtime.js |
| mdocument-parsing | no | none | `MDocument.fromText()` creates document; `MDocument.fromHTML()` tested for availability (may not be supported) |

### rerank/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| basic | no | none | Checks availability of `rerank`, `createVectorQueryTool`, `GraphRAG`, and `MDocument` on the agent embed namespace |
| functional | no | none | Verifies `rerank` and `rerankWithScorer` are available as callable functions on globalThis |

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| vector-query-tool | yes | libsql-server | End-to-end RAG pipeline: creates LibSQLVector index, chunks document with recursive strategy, embeds chunks with `text-embedding-3-small`, upserts vectors, queries "What is brainkit?" and verifies relevant results returned |

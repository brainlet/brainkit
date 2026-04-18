# rag-pipeline

The full Mastra RAG flow shaped for brainkit: chunk → embed →
upsert → query-tool → agent. Asks a question whose answer only
lives in the corpus, then asks one that doesn't to show the
agent declines instead of hallucinating.

## Run

```sh
docker compose -f examples/rag-pipeline/docker-compose.yml up -d
export OPENAI_API_KEY=sk-...
export PGVECTOR_URL="postgres://brainkit:brainkit@127.0.0.1:5434/brainkit?sslmode=disable"
go run ./examples/rag-pipeline

# Optional: run the rerank path.
go run ./examples/rag-pipeline -rerank

docker compose -f examples/rag-pipeline/docker-compose.yml down -v
```

Expected tail:

```
[2/4] ingesting 6 seed documents (chunk → embed → upsert)
        inserted 6 chunks across 6 docs

[3/4] positive question (answer lives in the corpus)
        Q: What is the brew pressure of the Vintlo Mk-7, and what powers its pump?
        A: The Vintlo Mk-7 produces 9 bars of brew pressure at the group head, powered by a 165-watt rotary pump. [docId: flr-004]
        sources: [flr-004]
        ✓ answer cites the corpus facts
        ✓ cited flr-004

[4/4] negative question (not in corpus — agent should decline)
        Q: What flavor of ice cream does the Vintlo Mk-7 pair best with?
        A: I don't know based on the available documents.
        ✓ agent declined rather than hallucinating
```

Port 5434 coexists with `examples/storage-vectors` (5433), so
both compose stacks can run side-by-side.

## What it shows

| Primitive | Role |
|---|---|
| `MDocument.fromText(...).chunk({strategy: "recursive", maxSize, overlap})` | chunking long documents |
| `embeddingModel("openai", "text-embedding-3-small").doEmbed(...)` | vectorizing chunks |
| `vectorStore("docs")` + `store.upsert(...)` | persisting chunks |
| `createVectorQueryTool({vectorStore, indexName, model})` | retrieval tool attached to an Agent |
| `Agent.generate(question)` | the LLM reads the tool output under strict "only from context" instructions |
| `rerankWithScorer(...)` | optional — rerank an over-fetched candidate set |

Shape of the pipeline:

```
docs → MDocument.chunk ─► embeddingModel.doEmbed ─► vectorStore.upsert ─► index "docs"
                                                                              │
question ──┐                                                                  ▼
           └──► Agent (with createVectorQueryTool) ──► similarity search ─► top-K
                                                                              │
                                                                              ▼
                                                                    Agent reads context +
                                                                    cites docId or declines
```

## Chunking knobs

`MDocument.chunk(options)` strategies worth knowing:

| strategy | use case | notes |
|---|---|---|
| `recursive` (default here) | generic prose | splits on paragraphs, then sentences, then characters |
| `markdown` | docs with headings | preserves section boundaries |
| `token` | token-accurate bounds | imports `js-tiktoken`, already bundled |

Tune with `maxSize` (default 400 here — small enough that each
fictional fact is ~1 chunk) and `overlap` (40 here — avoids
splitting a fact mid-sentence).

## When to turn on `-rerank`

The `-rerank` flag fetches `topK=8` candidates, reranks via
`rerankWithScorer` with combined semantic / vector / position
weights, and hands the top 3 to the agent as an explicit
preamble. Useful when the first-pass vector similarity gives
near-identical scores for several candidates — rerank breaks
ties using the embedding distance to the original query.

Before / after shape (illustrative):

```
no rerank:    [flr-003 0.81, flr-004 0.80, flr-005 0.74, flr-002 0.68]
with rerank:  [flr-004 0.92, flr-003 0.86, flr-002 0.62]  ← pump fact pulled up
```

## Swap the vector store

Same `.ts`, different backend — just change `Config.Vectors`:

| Backend | Config | When |
|---|---|---|
| PgVector (this example) | `brainkit.PgVectorStore(url)` | Production-grade; works alongside a normal Postgres |
| LibSQLVector | `brainkit.SQLiteVector(path)` | Embedded, no services; libsql build must include vector extension |
| MongoDBVectorStore | `brainkit.MongoVectorStore(uri)` | Already using MongoDB elsewhere |

## See also

- `examples/storage-vectors/` — the raw upsert + query path
  without an Agent or the RAG toolbox.
- `examples/custom-scorer/` — plug a custom scorer into
  `rerankWithScorer` for domain-specific reranking.
- `docs/guides/vectors-and-rag.md` — the prose guide.

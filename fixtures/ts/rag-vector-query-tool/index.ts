// Test: createVectorQueryTool — end-to-end RAG pipeline
import { MDocument, LibSQLVector } from "agent";
import { embed } from "ai";
import { model, output } from "kit";

const url = globalThis.process?.env?.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set");

// 1. Create vector store + index
const vectorStore = new LibSQLVector({ id: "rag-test", url });
await vectorStore.createIndex({ indexName: "rag_docs", dimension: 1536 });

// 2. Chunk a document
const doc = MDocument.fromText(`
Brainlet is an Agent OS. It runs teams of AI agents that operate like virtual employees.
The core engine is called brainkit. It embeds Mastra and AI SDK inside QuickJS.
Agents communicate through a bus system with pub/sub and request/response patterns.
Workflows support suspend and resume for human-in-the-loop approval.
`.trim());

const chunks = await doc.chunk({ strategy: "recursive", maxSize: 100, overlap: 10 });

// 3. Embed and upsert chunks
const vectors = [];
const metadata = [];
const ids = [];
for (let i = 0; i < chunks.length; i++) {
  const embResult = await embed({
    model: model("openai", "text-embedding-3-small"),
    value: chunks[i].text,
  });
  vectors.push(embResult.embedding);
  metadata.push({ text: chunks[i].text, index: i });
  ids.push("chunk-" + i);
}

await (vectorStore as any).upsert({
  indexName: "rag_docs",
  vectors,
  metadata,
  ids,
});

// 4. Query — find chunks about "brainkit"
const queryEmb = await embed({
  model: model("openai", "text-embedding-3-small"),
  value: "What is brainkit?",
});

const results = await vectorStore.query({
  indexName: "rag_docs",
  queryVector: queryEmb.embedding,
  topK: 2,
});

output({
  chunkCount: chunks.length,
  resultCount: results.length,
  hasResults: results.length > 0,
  topResult: (results[0]?.metadata?.text as string)?.substring(0, 80),
});

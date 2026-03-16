// Test: Workspace vector + hybrid search
// Requires: real libsql-server (testcontainer) with vector extensions + OpenAI API key
// The LIBSQL_URL env var is set by the test to point to a real libsql-server with vector32() support.
import { Workspace, LocalFilesystem, LibSQLVector, ai, output } from "brainlet";

const basePath = globalThis.process?.env?.WORKSPACE_PATH;
const libsqlUrl = globalThis.process?.env?.LIBSQL_URL;
if (!basePath) throw new Error("WORKSPACE_PATH not set");
if (!libsqlUrl) throw new Error("LIBSQL_URL not set — need a real libsql-server with vector extensions");

const results = {};

try {
  const embedder = async (text) => {
    const r = await ai.embed({ model: "openai/text-embedding-3-small", value: text });
    return r.embedding;
  };

  const vectors = new LibSQLVector({ id: "ws-vectors", url: libsqlUrl });
  const ws = new Workspace({
    id: "vector-search-test",
    filesystem: new LocalFilesystem({ basePath }),
    bm25: true,
    vectorStore: vectors,
    embedder: embedder,
  });

  await ws.init();

  // Create vector index
  try {
    await vectors.createIndex({ indexName: "vector_search_test_search", dimension: 1536, metric: "cosine" });
  } catch(e) { /* index might exist */ }

  // Index documents
  await ws.index("doc1.txt", "Rust is a systems programming language focused on safety and performance.");
  await ws.index("doc2.txt", "Python is great for data science, machine learning, and scripting.");
  await ws.index("doc3.txt", "Go is a compiled language designed for concurrent programming and cloud services.");

  // 1. BM25 search
  const bm25Results = await ws.search("systems programming safety", { mode: "bm25", topK: 3 });
  results.bm25Count = bm25Results.length;

  // 2. Vector search — semantic match
  const vectorResults = await ws.search("language for building web servers", { mode: "vector", topK: 3 });
  results.vectorCount = vectorResults.length;
  results.vectorTopId = vectorResults[0]?.id || "none";

  // 3. Hybrid search — combined
  const hybridResults = await ws.search("fast compiled language", { mode: "hybrid", topK: 3 });
  results.hybridCount = hybridResults.length;
  results.hybridHasScoreDetails = hybridResults[0]?.scoreDetails ? "ok" : "no details";

  // 4. Auto-detect (should use hybrid since both configured)
  const autoResults = await ws.search("programming language", { topK: 3 });
  results.autoCount = autoResults.length;

  await ws.destroy();
  results.status = "ok";

} catch(e) {
  results.error = e.message;
  results.stack = (e.stack || "").substring(0, 200);
}

output(results);

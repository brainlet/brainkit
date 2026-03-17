// Test: Workspace BM25 search (keyword-based)
// Works with embedded SQLite bridge — no vector extensions needed.
import { Workspace, LocalFilesystem, output } from "kit";

const basePath = globalThis.process?.env?.WORKSPACE_PATH;
if (!basePath) throw new Error("WORKSPACE_PATH not set");

const results = {};

try {
  const ws = new Workspace({
    id: "bm25-test",
    filesystem: new LocalFilesystem({ basePath }),
    bm25: true,
  });

  await ws.init();

  // Index documents
  await ws.index("doc1.txt", "Rust is a systems programming language focused on safety and performance.");
  await ws.index("doc2.txt", "Python is great for data science, machine learning, and scripting.");
  await ws.index("doc3.txt", "Go is a compiled language designed for concurrent programming and cloud services.");

  // 1. BM25 search — keyword match
  const rustResults = await ws.search("systems programming safety", { mode: "bm25", topK: 3 });
  results.rustCount = rustResults.length;
  results.rustTopId = rustResults[0]?.id || "none";
  results.rustHasScore = typeof rustResults[0]?.score === "number" ? "ok" : "no score";

  // 2. Search for Go
  const goResults = await ws.search("concurrent cloud compiled", { mode: "bm25", topK: 3 });
  results.goCount = goResults.length;
  results.goTopId = goResults[0]?.id || "none";

  // 3. Search with no results
  const noResults = await ws.search("javascript react frontend", { mode: "bm25", topK: 3 });
  results.noResultsCount = noResults.length;

  // 4. Auto-detect mode (should use bm25 since only bm25 is configured)
  const autoResults = await ws.search("programming language", { topK: 3 });
  results.autoCount = autoResults.length;

  await ws.destroy();
  results.status = "ok";

} catch(e) {
  results.error = e.message;
}

output(results);

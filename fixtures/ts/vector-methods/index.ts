// Test: Vector store management methods — createIndex, listIndexes, describeIndex, deleteIndex
import { LibSQLVector } from "agent";
import { output } from "kit";

const url = globalThis.process?.env?.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set, process.env keys: " + Object.keys(globalThis.process?.env || {}).join(","));

const vector: any = new LibSQLVector({
  id: "test-vector-methods",
  url: url,
});

// 1. Create two indexes
await vector.createIndex({ indexName: "test_idx_a", dimension: 3 });
await vector.createIndex({ indexName: "test_idx_b", dimension: 5 });

// 2. List indexes
const indexes = await vector.listIndexes();
const hasA = indexes.includes("test_idx_a");
const hasB = indexes.includes("test_idx_b");

// 3. Describe an index
const info = await vector.describeIndex({ indexName: "test_idx_a" });

// 4. Upsert some vectors
await (vector as any).upsert({
  indexName: "test_idx_a",
  vectors: [[1.0, 0.0, 0.0], [0.0, 1.0, 0.0]],
  ids: ["v1", "v2"],
  metadata: [{ tag: "first" }, { tag: "second" }],
});

// 5. Query to verify data
const results = await vector.query({
  indexName: "test_idx_a",
  queryVector: [1.0, 0.0, 0.0],
  topK: 1,
});

// 6. Delete a single vector
await vector.deleteVectors({ indexName: "test_idx_a", ids: ["v2"] });

// 7. Query again — should only find v1
const afterDelete = await vector.query({
  indexName: "test_idx_a",
  queryVector: [0.0, 1.0, 0.0],
  topK: 2,
});

// 8. Delete an index
await vector.deleteIndex({ indexName: "test_idx_b" });

// 9. List again
const indexesAfter = await vector.listIndexes();
const bGone = !indexesAfter.includes("test_idx_b");

output({
  listIndexes: { hasA, hasB, count: indexes.length },
  describeIndex: info,
  query: { topId: results[0]?.id, count: results.length },
  afterDelete: { count: afterDelete.length, ids: afterDelete.map(r => r.id) },
  deleteIndex: { bGone },
  allPassed: hasA && hasB && results[0]?.id === "v1" && afterDelete.length === 1 && bGone,
});

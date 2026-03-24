// Test: PgVector — PostgreSQL vector store for semantic search
import { PgVector } from "agent";
import { output } from "kit";

const url = globalThis.process?.env?.POSTGRES_URL;
if (!url) throw new Error("POSTGRES_URL not set");

const vector = new PgVector({
  id: "test-pgvector",
  connectionString: url,
});

// Create an index
await vector.createIndex({
  indexName: "test_vectors",
  dimension: 3,
});

// Upsert vectors
await vector.upsert({
  indexName: "test_vectors",
  vectors: [
    [1.0, 0.0, 0.0],
    [0.0, 1.0, 0.0],
    [0.0, 0.0, 1.0],
    [0.7, 0.7, 0.0],
  ],
  ids: ["x-axis", "y-axis", "z-axis", "xy-diag"],
  metadata: [
    { label: "x" },
    { label: "y" },
    { label: "z" },
    { label: "xy" },
  ],
});

// Query — find vectors closest to [1, 0, 0] (should be x-axis)
const results = await vector.query({
  indexName: "test_vectors",
  queryVector: [0.9, 0.1, 0.0],
  topK: 2,
});

output({
  resultCount: results.length,
  topId: results[0]?.id,
  topLabel: results[0]?.metadata?.label,
  secondId: results[1]?.id,
});

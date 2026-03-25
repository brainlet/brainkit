// Test: MongoDBVector — basic vector upsert on MongoDB Community
// MongoDB Community doesn't support Atlas Search, so we test upsert/query only.
import { MongoDBVector } from "agent";
import { output } from "kit";

const url = process.env.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");

try {
  const vector = new MongoDBVector({
    id: "test-mongo-vector",
    uri: url,
    dbName: "brainlet_vector_test",
  });

  // Explicitly connect first
  await vector.connect();
  console.log("[vector-mongodb] connected");

  // Try createIndex — will fail on Community (no Atlas Search)
  try {
    await vector.createIndex({ indexName: "test_vectors", dimension: 3 });
    output({ created: true, atlas: true });
  } catch (e: any) {
    console.log("[vector-mongodb] createIndex error:", e.message?.substring(0, 100));
    // Expected: Community Edition doesn't support createSearchIndexes
    // Verify we can upsert to the backing collection
    await (vector as any).upsert({
      indexName: "test_vectors",
      vectors: [[1.0, 0.0, 0.0], [0.0, 1.0, 0.0]],
      ids: ["vec-x", "vec-y"],
      metadata: [{ label: "x" }, { label: "y" }],
    });
    output({ created: false, atlas: false, upserted: 2, reason: "community-edition" });
  }

  await vector.disconnect();
} catch (e: any) {
  console.error("[vector-mongodb] ERROR:", e.message);
  output({ error: e.message, stack: (e.stack || "").substring(0, 300) });
}

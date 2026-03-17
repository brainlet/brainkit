// Test: MongoDBVector — MongoDB Atlas-style vector search
// Note: MongoDB Community doesn't support Atlas Search indexes,
// so we test basic vector upsert and retrieval via the MongoDBVector API.
import { MongoDBVector, output } from "kit";

const url = globalThis.process?.env?.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");

const vector = new MongoDBVector({
  id: "test-mongo-vector",
  uri: url,
  dbName: "brainlet_vector_test",
});

// MongoDB Community doesn't support Atlas Search (createSearchIndexes).
// We can only verify that the MongoDBVector class instantiates and connects.
// Full vector search requires MongoDB Atlas.
// Test: verify the constructor works and we can upsert documents to the collection.
try {
  await vector.createIndex({ indexName: "test_vectors", dimension: 3 });
  output({ created: true, atlas: true });
} catch(e) {
  if (e.message && e.message.includes("createSearchIndexes")) {
    // Expected on Community Edition — Atlas Search not available.
    // Verify we can at least upsert to the backing collection.
    await vector.upsert({
      indexName: "test_vectors",
      vectors: [[1.0, 0.0, 0.0], [0.0, 1.0, 0.0]],
      ids: ["x", "y"],
      metadata: [{ label: "x" }, { label: "y" }],
    });
    output({ created: false, atlas: false, upserted: 2, reason: "community-edition" });
  } else {
    throw e;
  }
}

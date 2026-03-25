import { MongoDBVector } from "agent";
import { output } from "kit";
const url = process.env.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");
try {
  const vs = new MongoDBVector({ id: "mongo-crud", uri: url, dbName: "vector_test" });
  // MongoDB Community Edition doesn't support $vectorSearch (needs Atlas)
  // Test createIndex + upsert at minimum
  const indexName = "test_crud_" + Date.now();
  await vs.createIndex({ indexName, dimension: 3 });
  await vs.upsert({ indexName, vectors: [
    { id: "v1", vector: [1, 0, 0], metadata: { label: "x" } },
  ] });
  output({ upserted: true, backend: "mongodb" });
} catch(e: any) { output({ error: e.message.substring(0, 200) }); }

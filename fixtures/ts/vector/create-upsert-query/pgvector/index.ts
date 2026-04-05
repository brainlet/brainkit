import { PgVector } from "agent";
import { output } from "kit";
const url = process.env.POSTGRES_URL;
if (!url) throw new Error("POSTGRES_URL not set");
try {
  const vs = new PgVector({ id: "pg-crud", connectionString: url });
  const indexName = "test_crud_" + Date.now();
  await vs.createIndex({ indexName, dimension: 3 });
  await vs.upsert({ indexName, vectors: [
    { id: "v1", vector: [1, 0, 0], metadata: { label: "x" } },
    { id: "v2", vector: [0, 1, 0], metadata: { label: "y" } },
  ] });
  const results = await vs.query({ indexName, queryVector: [1, 0, 0], topK: 2 });
  await vs.deleteIndex(indexName);
  output({ resultCount: results.length, topId: results[0]?.id || "" });
} catch(e: any) { output({ error: e.message.substring(0, 200) }); }

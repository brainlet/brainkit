import { LibSQLVector } from "agent";
import { output } from "kit";
const url = process.env.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set");
try {
  const vs = new LibSQLVector({ id: "crud-test", connectionUrl: url });
  const indexName = "test_crud_" + Date.now();
  await vs.createIndex({ indexName, dimension: 3 });
  const indexes = await vs.listIndexes();
  await vs.upsert({ indexName, vectors: [
    { id: "v1", vector: [1, 0, 0], metadata: { label: "x" } },
    { id: "v2", vector: [0, 1, 0], metadata: { label: "y" } },
    { id: "v3", vector: [0, 0, 1], metadata: { label: "z" } },
  ] });
  const results = await vs.query({ indexName, queryVector: [1, 0, 0], topK: 2 });
  await vs.deleteIndex(indexName);
  output({ indexCreated: indexes.includes(indexName), resultCount: results.length, topId: results[0]?.id || "" });
} catch(e: any) { output({ error: e.message.substring(0, 200) }); }

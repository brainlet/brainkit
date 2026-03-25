// Test: agent memory with MongoDB storage (real database)
// The MONGODB_URL env var is set by the test (points to testcontainer)
import { Agent, Memory, MongoDBStore } from "agent";
import { model, output } from "kit";

const url = process.env.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");

try {
  const store = new MongoDBStore({
    id: "test-mongodb-store",
    url: url,
    dbName: "brainlet_test",
  });

  await store.init();

  const mem = new Memory({
    storage: store,
    options: { lastMessages: 10 },
  });

  const a = new Agent({
    name: "fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a helpful assistant. Remember what the user tells you.",
    memory: mem,
  });

  // First call
  await a.generate("My favorite color is purple.", {
    memory: { thread: { id: "mongodb-test-1" }, resource: "test-user" },
  });

  // Second call — should remember
  const result = await a.generate("What is my favorite color?", {
    memory: { thread: { id: "mongodb-test-1" }, resource: "test-user" },
  });

  output({
    text: result.text,
    remembers: result.text.toLowerCase().includes("purple"),
    store: "mongodb",
  });
} catch (e: any) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 500) });
}

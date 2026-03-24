// Test: agent memory with MongoDB storage (real database)
// The MONGODB_URL env var is set by the test (points to testcontainer)
import { Agent, MongoDBStore } from "agent";
import { model, output } from "kit";

const url = globalThis.process?.env?.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");

const store = new MongoDBStore({
  id: "test-mongodb-store",
  url: url,
  dbName: "brainlet_test",
});

// Explicitly init — Mastra may not auto-init before first use
await store.init();

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: store as any,
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
  url: url,
});

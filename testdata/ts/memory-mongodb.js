// Test: agent memory with MongoDB storage (real database)
// The MONGODB_URL env var is set by the test (points to testcontainer)
import { agent, MongoDBStore, output } from "kit";

const url = globalThis.process?.env?.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");

const store = new MongoDBStore({
  id: "test-mongodb-store",
  url: url,
  dbName: "brainlet_test",
});

// Explicitly init — Mastra may not auto-init before first use
await store.init();

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: {
    thread: "mongodb-test-1",
    resource: "test-user",
    storage: store,
  },
});

// First call
await a.generate("My favorite color is purple.");

// Second call — should remember
const result = await a.generate("What is my favorite color?");

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("purple"),
  store: "mongodb",
  url: url,
});

// Test: agent memory with MongoDB storage (real database)
// The MONGODB_URL env var is set by the test (points to testcontainer)
import { Agent, Memory, MongoDBStore } from "agent";
import { model, output } from "kit";

const url = process.env.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");

try {
  console.log("[mongo] creating store with url:", url.substring(0, 30));
  const store = new MongoDBStore({
    id: "test-mongodb-store",
    url: url,
    dbName: "brainlet_test",
  });
  console.log("[mongo] store created, calling init...");

  await store.init();
  console.log("[mongo] store initialized");

  const mem = new Memory({
    storage: store,
    options: { lastMessages: 10 },
  });
  console.log("[mongo] memory created");

  const a = new Agent({
    name: "fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a helpful assistant. Remember what the user tells you.",
    memory: mem,
  });
  console.log("[mongo] agent created, generating...");

  await a.generate("My favorite color is purple.", {
    memory: { thread: { id: "mongodb-test-1" }, resource: "test-user" },
  });
  console.log("[mongo] first generate done");

  const result = await a.generate("What is my favorite color?", {
    memory: { thread: { id: "mongodb-test-1" }, resource: "test-user" },
  });
  console.log("[mongo] second generate done");

  output({
    text: result.text,
    remembers: result.text.toLowerCase().includes("purple"),
    store: "mongodb",
  });
} catch (e: any) {
  console.error("[mongo] ERROR:", e.message);
  output({ error: e.message, stack: (e.stack || "").substring(0, 500) });
}

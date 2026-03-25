// Test: agent memory with PostgreSQL storage (real database)
// The POSTGRES_URL env var is set by the test (points to testcontainer)
import { Agent, Memory, PostgresStore } from "agent";
import { model, output } from "kit";

const url = process.env.POSTGRES_URL;
if (!url) throw new Error("POSTGRES_URL not set");

try {
  const store = new PostgresStore({
    id: "test-postgres-store",
    connectionString: url,
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
  await a.generate("My favorite animal is a cat.", {
    memory: { thread: { id: "postgres-test-1" }, resource: "test-user" },
  });

  // Second call — should remember
  const result = await a.generate("What is my favorite animal?", {
    memory: { thread: { id: "postgres-test-1" }, resource: "test-user" },
  });

  output({
    text: result.text,
    remembers: result.text.toLowerCase().includes("cat"),
    store: "postgres",
  });
} catch (e: any) {
  output({
    error: e.message,
    stack: (e.stack || "").substring(0, 500),
    store: "postgres",
  });
}

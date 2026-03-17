// Test: agent memory with PostgreSQL storage (real database)
// The POSTGRES_URL env var is set by the test (points to testcontainer)
import { agent, PostgresStore, output } from "kit";

const url = globalThis.process?.env?.POSTGRES_URL;
if (!url) throw new Error("POSTGRES_URL not set");

const store = new PostgresStore({
  id: "test-postgres-store",
  connectionString: url,
});

await store.init();

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: {
    thread: "postgres-test-1",
    resource: "test-user",
    storage: store,
  },
});

// First call
await a.generate("My favorite animal is a cat.");

// Second call — should remember
const result = await a.generate("What is my favorite animal?");

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("cat"),
  store: "postgres",
  url: url,
});

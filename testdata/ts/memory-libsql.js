// Test: agent memory with LibSQL storage (real database)
// The LIBSQL_URL env var is set by the test (points to testcontainer or in-memory)
import { agent, Memory, LibSQLStore, output } from "kit";

const url = globalThis.process?.env?.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set — need a LibSQL HTTP endpoint (testcontainer or Turso)");

const store = new LibSQLStore({
  id: "test-libsql-store",
  url: url,
});

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: {
    thread: "libsql-test-1",
    resource: "test-user",
    storage: store,
  },
});

// First call
await a.generate("My favorite programming language is Go.");

// Second call — should remember
const result = await a.generate("What is my favorite programming language?");

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("go"),
  store: "libsql",
  url: url,
});

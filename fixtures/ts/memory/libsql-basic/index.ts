// Test: agent memory with LibSQL storage (real database)
// The LIBSQL_URL env var is set by the test (points to testcontainer or in-memory)
import { Agent, Memory, LibSQLStore } from "agent";
import { model, output } from "kit";

const url = globalThis.process?.env?.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set — need a LibSQL HTTP endpoint (testcontainer or Turso)");

const store = new LibSQLStore({
  id: "test-libsql-store",
  url: url,
});

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
await a.generate("My favorite programming language is Go.", {
  memory: { thread: { id: "libsql-test-1" }, resource: "test-user" },
});

// Second call — should remember
const result = await a.generate("What is my favorite programming language?", {
  memory: { thread: { id: "libsql-test-1" }, resource: "test-user" },
});

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("go"),
  store: "libsql",
  url: url,
});

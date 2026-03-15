// Test: semantic recall — Mastra Memory with vector store
// Uses the same pattern as Mastra docs: create Memory, pass to agent
import { agent, Memory, LibSQLStore, LibSQLVector, output } from "brainlet";

const url = globalThis.process?.env?.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set");

// Mastra-style: create Memory with storage + vector + embedder
const memory = new Memory({
  storage: new LibSQLStore({ id: "sem-storage", url: url }),
  vector: new LibSQLVector({ id: "sem-vector", url: url }),
  embedder: "openai/text-embedding-3-small",
  options: {
    lastMessages: 5,
    semanticRecall: {
      topK: 3,
      messageRange: 2,
    },
  },
});

// Mastra-style: pass Memory instance to agent
const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what users tell you.",
  memory: memory,
});

// Teach the agent a fact
await a.generate("My favorite programming language is Rust.", {
  memory: { thread: { id: "sem-test-1" }, resource: "test-user" },
});

// Ask about it — semantic recall should find the earlier message
const result = await a.generate("What programming language do I like?", {
  memory: { thread: { id: "sem-test-1" }, resource: "test-user" },
});

output({
  text: result.text,
  remembersRust: result.text.toLowerCase().includes("rust"),
});

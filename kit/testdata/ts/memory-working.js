// Test: working memory — agent maintains a structured profile across calls
import { Agent, Memory, LibSQLStore, LibSQLVector } from "agent";
import { model, output } from "kit";

const url = globalThis.process?.env?.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set");

const memory = new Memory({
  storage: new LibSQLStore({ id: "wm-storage", url: url }),
  vector: new LibSQLVector({ id: "wm-vector", url: url }),
  embedder: model("openai", "text-embedding-3-small"),
  options: {
    lastMessages: 5,
    semanticRecall: true,
    workingMemory: {
      enabled: true,
      template: `# User Profile
- Name:
- Location:
- Favorite Language:
`,
    },
  },
});

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: `You are a helpful assistant with working memory.
When you learn facts about the user, update your working memory.
Always check working memory before answering questions about the user.`,
  memory: memory,
});

const memOpts = { memory: { thread: { id: "wm-test-1" }, resource: "wm-user" } };

// Teach facts across multiple calls
await a.generate("My name is Alice.", memOpts);
await a.generate("I live in Tokyo.", memOpts);
await a.generate("I love writing Rust.", memOpts);

// Ask a question that requires working memory
const result = await a.generate("What do you know about me?", memOpts);

output({
  text: result.text,
  knowsName: result.text.toLowerCase().includes("alice"),
  knowsLocation: result.text.toLowerCase().includes("tokyo"),
  knowsLanguage: result.text.toLowerCase().includes("rust"),
});

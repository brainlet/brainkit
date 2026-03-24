// Debug: test LibSQLStore directly — create, save thread, save messages, recall
import { Agent, Memory, LibSQLStore } from "agent";
import { model, output } from "kit";

const url = globalThis.process?.env?.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set");

const store = new LibSQLStore({
  id: "debug-store",
  url: url,
});

// Step 1: Create a memory instance with thread
const mem = new Memory({
  storage: store,
  options: {
    lastMessages: 10,
  },
});

// Step 2: First agent call
const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: mem,
});

const r1 = await a.generate("My favorite color is blue.", {
  memory: { thread: { id: "debug-thread-1" }, resource: "test-user" },
});
const r1text = r1.text;

// Step 3: Second agent call
const r2 = await a.generate("What is my favorite color?", {
  memory: { thread: { id: "debug-thread-1" }, resource: "test-user" },
});
const r2text = r2.text;

output({
  call1: r1text,
  call2: r2text,
  remembers: r2text.toLowerCase().includes("blue"),
});

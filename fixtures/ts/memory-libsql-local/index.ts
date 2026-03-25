// Test: agent memory with local SQLite via Kit's embedded bridge.
// Uses LIBSQL_URL from env (injected by test runner from embedded bridge).
import { Agent, Memory, LibSQLStore } from "agent";
import { model, output } from "kit";

const url = process.env.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set — embedded bridge URL should be injected by test");

const store = new LibSQLStore({ id: "local-store", url: url });
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

// First call — teach
await a.generate("My favorite color is blue and my dog's name is Rex.", {
  memory: { thread: { id: "local-sqlite-test-1" }, resource: "test-user" },
});

// Second call — recall
const result = await a.generate("What is my favorite color and what is my dog's name?", {
  memory: { thread: { id: "local-sqlite-test-1" }, resource: "test-user" },
});

const text = result.text.toLowerCase();
output({
  text: result.text,
  remembersColor: text.includes("blue"),
  remembersDog: text.includes("rex"),
  store: "local-sqlite",
});

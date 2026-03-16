// Test: agent memory with local SQLite via Kit's embedded bridge.
// No URL needed — LibSQLStore auto-connects to the Kit's embedded bridge.
import { agent, Memory, LibSQLStore, output } from "brainlet";

const store = new LibSQLStore({ id: "local-store" });

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: {
    thread: "local-sqlite-test-1",
    resource: "test-user",
    storage: store,
  },
});

// First call — teach
await a.generate("My favorite color is blue and my dog's name is Rex.");

// Second call — recall
const result = await a.generate("What is my favorite color and what is my dog's name?");

const text = result.text.toLowerCase();
output({
  text: result.text,
  remembersColor: text.includes("blue"),
  remembersDog: text.includes("rex"),
  store: "local-sqlite",
});

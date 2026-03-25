// Test: agent memory with InMemoryStore (no external database)
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

const memory = new Memory({
  storage: new InMemoryStore(),
  options: { lastMessages: 10 },
});

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: memory,
});

// First call — tell the agent something
await a.generate("My name is Alice and I work at Brainlet.", {
  memory: { thread: { id: "inmemory-test-1" }, resource: "test-user" },
});

// Second call — ask about it
const result = await a.generate("What is my name and where do I work?", {
  memory: { thread: { id: "inmemory-test-1" }, resource: "test-user" },
});

output({
  text: result.text,
  remembersName: result.text.toLowerCase().includes("alice"),
  remembersWork: result.text.toLowerCase().includes("brainlet"),
});

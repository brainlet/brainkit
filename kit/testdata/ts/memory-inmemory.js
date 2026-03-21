// Test: agent memory with InMemoryStore (no external database)
import { agent, Memory, InMemoryStore, output } from "kit";

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: {
    thread: "inmemory-test-1",
    resource: "test-user",
    storage: new InMemoryStore(),
  },
});

// First call — tell the agent something
await a.generate("My name is Alice and I work at Brainlet.");

// Second call — ask about it
const result = await a.generate("What is my name and where do I work?");

output({
  text: result.text,
  remembersName: result.text.toLowerCase().includes("alice"),
  remembersWork: result.text.toLowerCase().includes("brainlet"),
});

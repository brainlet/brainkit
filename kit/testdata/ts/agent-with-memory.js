// Test: agent with memory — remembers across calls
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
await a.generate("My favorite color is blue. Remember that.", {
  memory: { thread: { id: "test-conversation-1" }, resource: "test-user" },
});

// Second call — ask about it
const result = await a.generate("What is my favorite color?", {
  memory: { thread: { id: "test-conversation-1" }, resource: "test-user" },
});

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("blue"),
});

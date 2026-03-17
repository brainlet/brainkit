// Test: agent with memory — remembers across calls
import { agent, output } from "kit";

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: {
    thread: "test-conversation-1",
    resource: "test-user",
  },
});

// First call — tell the agent something
await a.generate("My favorite color is blue. Remember that.");

// Second call — ask about it
const result = await a.generate("What is my favorite color?");

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("blue"),
});

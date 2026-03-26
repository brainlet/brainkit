// Test: agent.generate with context messages prepended
import { Agent } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "context-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Answer questions based on the provided context.",
});

const result = await agent.generate("What is Alice's favorite color?", {
  context: [
    { role: "user" as const, content: "Alice likes blue and her dog is named Rex." },
    { role: "assistant" as const, content: "Got it, I'll remember that about Alice." },
  ],
});

output({
  hasText: result.text.length > 0,
  knowsBlue: result.text.toLowerCase().includes("blue"),
});

// Test: per-call instructions override
import { Agent } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "overridable",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a helpful assistant.",
});

const result = await agent.generate("Say hello", {
  instructions: "You are a pirate. Always respond in pirate speak. Keep it to one sentence.",
});

output({
  hasText: result.text.length > 0,
  // Pirate speak typically contains arrr, ahoy, matey, etc.
  text: result.text,
});

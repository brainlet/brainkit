// Test: agent creates and generates a response
import { agent, output } from "kit";

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Reply with exactly: FIXTURE_WORKS",
});

const result = await a.generate("Say the magic word");
output({
  text: result.text,
  hasUsage: !!result.usage,
  finishReason: result.finishReason,
});

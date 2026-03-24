// Test: agent creates and generates a response
import { Agent } from "agent";
import { model, output } from "kit";

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply with exactly: FIXTURE_WORKS",
});

const result = await a.generate("Say the magic word");
output({
  text: result.text,
  hasUsage: !!result.usage,
  finishReason: result.finishReason,
});

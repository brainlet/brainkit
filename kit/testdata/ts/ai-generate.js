// Test: direct AI call without an agent (LOCAL, no Mastra overhead)
import { ai, output } from "kit";

const result = await ai.generate({
  model: "openai/gpt-4o-mini",
  prompt: "Reply with exactly one word: DIRECT",
});

output({
  text: result.text,
  hasUsage: !!result.usage,
  finishReason: result.finishReason,
});

// Test: direct AI call without an agent (LOCAL, no Mastra overhead)
import { generateText } from "ai";
import { model, output } from "kit";

const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "Reply with exactly one word: DIRECT",
});

output({
  text: result.text,
  hasUsage: !!result.usage,
  finishReason: result.finishReason,
});

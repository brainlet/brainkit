import { generateText } from "ai";
import { model, output } from "kit";
const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  system: "You are a pirate. Always respond in pirate speak.",
  prompt: "Say hello in one sentence",
});
output({ hasText: result.text.length > 0, finishReason: result.finishReason, hasUsage: result.usage.totalTokens > 0 });

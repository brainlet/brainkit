// Test: generateText with maxTokens — limited output
import { generateText } from "ai";
import { model, output } from "kit";

const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "Write a very long and detailed essay about the complete history of computing from the abacus to modern AI. Include every major milestone, inventor, and breakthrough.",
  maxTokens: 10,
});

output({
  hasText: result.text.length > 0,
  isShort: result.text.split(" ").length < 30,
  finishReason: result.finishReason,
});

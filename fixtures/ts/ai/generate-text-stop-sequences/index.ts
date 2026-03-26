// Test: generateText with stopSequences
import { generateText } from "ai";
import { model, output } from "kit";

const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "Count from 1 to 10, each number on a new line: 1\n2\n3\n",
  stopSequences: ["5"],
});

output({
  hasText: result.text.length > 0,
  stoppedBefore5: !result.text.includes("6"),
  finishReason: result.finishReason,
});

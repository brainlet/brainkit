import { generateObject } from "ai";
import { model, output } from "kit";
const result = await generateObject({
  model: model("openai", "gpt-4o-mini"),
  output: "enum",
  enum: ["positive", "negative", "neutral"],
  prompt: "Classify the sentiment: 'I love this product!'",
});
output({ value: result.object, isString: typeof result.object === "string", validEnum: ["positive", "negative", "neutral"].includes(result.object as string) });

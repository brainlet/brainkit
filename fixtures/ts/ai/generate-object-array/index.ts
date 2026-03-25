import { generateObject, z } from "ai";
import { model, output } from "kit";
const result = await generateObject({
  model: model("openai", "gpt-4o-mini"),
  schema: z.object({ name: z.string(), color: z.string() }),
  output: "array",
  prompt: "List 3 fruits with their colors",
});
output({ count: Array.isArray(result.object) ? result.object.length : 0, isArray: Array.isArray(result.object), finishReason: result.finishReason });

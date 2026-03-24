// Test: generateObject — structured output with schema
import { generateObject, z } from "ai";
import { model, output } from "kit";

const result = await generateObject<{ name: string; age: number; hobbies: string[] }>({
  model: model("openai", "gpt-4o-mini"),
  schema: z.object({
    name: z.string().describe("A fictional person's name"),
    age: z.number().describe("Age between 20 and 80"),
    hobbies: z.array(z.string()).describe("List of 2-3 hobbies"),
  }),
  prompt: "Generate a fictional person profile.",
});

output({
  object: result.object,
  hasName: typeof result.object.name === "string" && result.object.name.length > 0,
  hasAge: typeof result.object.age === "number" && result.object.age >= 1,
  hasHobbies: Array.isArray(result.object.hobbies) && result.object.hobbies.length >= 1,
  hasUsage: result.usage && result.usage.totalTokens > 0,
  finishReason: result.finishReason,
});

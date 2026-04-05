import { streamText, z } from "ai";
import { createTool } from "agent";
import { model, kit, output } from "kit";
const multiplyTool = createTool({
  id: "multiply",
  description: "Multiply two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a * b }),
});
kit.register("tool", "multiply_ai", multiplyTool);
const result = streamText({
  model: model("openai", "gpt-4o-mini"),
  tools: { multiply: multiplyTool },
  maxSteps: 3,
  prompt: "What is 6 times 7? Use the multiply tool.",
});
let chunks = 0;
for await (const chunk of result.textStream) { chunks++; }
const text = await result.text;
const usage = await result.usage;
output({ chunks, hasText: text.length > 0, hasUsage: usage.totalTokens > 0 });

// Test: createTool with extended config fields (outputSchema passthrough)
import { agent, createTool, z, output } from "kit";

// Tool with outputSchema — Mastra validates the return value
const calculator = createTool({
  id: "calculator",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  outputSchema: z.object({ result: z.number() }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Use the calculator tool to add 17 and 25. Return ONLY the number.",
  tools: { calculator },
});

const result = await a.generate("What is 17 + 25?");

output({
  text: result.text,
  hasAnswer: result.text.includes("42"),
  toolCalls: result.toolCalls.length,
});

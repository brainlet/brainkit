// Test: agent uses a locally-defined tool with Zod schema
import { agent, createTool, z, output } from "kit";

const addTool = createTool({
  id: "add",
  description: "Adds two numbers together",
  inputSchema: z.object({
    a: z.number().describe("first number"),
    b: z.number().describe("second number"),
  }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Always use the add tool when asked to compute. Return just the number.",
  tools: { add: addTool },
});

const result = await a.generate("What is 15 + 27? Use the add tool.", { maxSteps: 3 });

output({
  text: result.text,
  toolCalls: result.toolCalls?.length || 0,
});

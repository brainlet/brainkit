// Test: activeTools option limits which tools are available per call
import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

const addTool = createTool({
  id: "add", description: "Add numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }: any) => ({ result: a + b }),
});

const multiplyTool = createTool({
  id: "multiply", description: "Multiply numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }: any) => ({ result: a * b }),
});

const agent = new Agent({
  name: "selective-tools",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Use available tools to answer math. Return only the number.",
  tools: { add: addTool, multiply: multiplyTool },
  maxSteps: 3,
});

// Only add is active — multiply should not be used
const result = await agent.generate("What is 3 + 4? Use the add tool.", {
  activeTools: ["add"],
});

output({
  hasText: result.text.length > 0,
  containsAnswer: result.text.includes("7"),
});

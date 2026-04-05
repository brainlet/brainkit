import { generateText, z } from "ai";
import { createTool } from "agent";
import { model, kit, output } from "kit";

const addTool = createTool({
  id: "add",
  description: "Add two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a + b }),
});
kit.register("tool", "add_ai", addTool);

const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  tools: { add: addTool },
  maxSteps: 3,
  prompt: "What is 17 + 25? Use the add tool.",
});

output({
  text: result.text,
  hasToolCalls: result.steps.some((s: any) => s.toolCalls && s.toolCalls.length > 0),
  stepsCount: result.steps.length,
  containsAnswer: result.text.includes("42"),
});

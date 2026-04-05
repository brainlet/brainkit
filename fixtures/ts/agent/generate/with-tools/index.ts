import { Agent, createTool, z } from "agent";
import { model, output } from "kit";
const addTool = createTool({
  id: "add",
  description: "Add two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a + b }),
});
const agent = new Agent({ name: "tool-agent", model: model("openai", "gpt-4o-mini"), instructions: "You have an add tool. Use it to answer math.", tools: { add: addTool } });
const result = await agent.generate("What is 17 + 25? Use the add tool.");
output({ text: result.text, hasToolCalls: result.toolCalls.length > 0 || result.steps.some((s: any) => s.toolCalls.length > 0) });

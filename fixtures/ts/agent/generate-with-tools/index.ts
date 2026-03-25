import { Agent, createTool, z } from "agent";
import { model, output } from "kit";
const calculator = createTool({ id: "calculator", description: "Perform arithmetic", inputSchema: z.object({ expression: z.string() }), execute: async ({ expression }) => { try { return { result: eval(expression) }; } catch { return { error: "invalid" }; } } });
const agent = new Agent({ name: "tool-agent", model: model("openai", "gpt-4o-mini"), instructions: "You have a calculator tool. Use it.", tools: { calculator } });
const result = await agent.generate("What is 123 * 456? Use the calculator tool.");
output({ text: result.text, hasToolCalls: result.toolCalls.length > 0 || result.steps.some((s: any) => s.toolCalls.length > 0) });

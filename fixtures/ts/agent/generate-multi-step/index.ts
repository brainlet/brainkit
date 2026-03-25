import { Agent, createTool, z } from "agent";
import { model, output } from "kit";
let callCount = 0;
const lookup = createTool({ id: "lookup", description: "Look up a fact", inputSchema: z.object({ query: z.string() }), execute: async ({ query }) => { callCount++; return { answer: "The capital of France is Paris" }; } });
const agent = new Agent({ name: "multi-step", model: model("openai", "gpt-4o-mini"), instructions: "Use the lookup tool to answer.", tools: { lookup }, maxSteps: 5 });
const result = await agent.generate("What is the capital of France? Use the lookup tool.");
output({ text: result.text, toolCallsMade: callCount, multiStep: result.steps.length > 1, containsAnswer: result.text.toLowerCase().includes("paris") });

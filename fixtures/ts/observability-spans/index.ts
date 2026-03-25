// Test: observability — verify agent with tools produces trace info
import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

const addTool = createTool({
  id: "add",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }: any) => ({ result: a + b }),
});

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Use the add tool to compute 10 + 32. Return ONLY the number.",
  tools: { add: addTool },
});

const result = await a.generate("What is 10 + 32?", { maxSteps: 3 });

output({
  text: result.text,
  hasAnswer: result.text.includes("42"),
  toolCalls: result.toolCalls?.length || 0,
  hasTraceId: typeof result.traceId === "string" && result.traceId.length > 0,
  traceId: result.traceId,
  hasRunId: typeof result.runId === "string" && result.runId.length > 0,
  hasUsage: (result.usage as any)?.totalTokens > 0,
  steps: result.steps?.length || 0,
});

// Test: end-to-end observability — Agent.generate() emits a trace with
// traceId + runId populated. Deeper span inspection (provider-specific)
// belongs in a Go suite test that wires a captured exporter.
import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

const echoTool = createTool({
  id: "echo",
  description: "Return the input unchanged.",
  inputSchema: z.object({ text: z.string() }),
  outputSchema: z.object({ text: z.string() }),
  execute: async ({ context }: any) => ({ text: context.text }),
});

const agent = new Agent({
  name: "obs-e2e",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Use the echo tool when asked.",
  tools: { echo: echoTool },
});

const result = await agent.generate("Call the echo tool with text='hi'.");

output({
  hasTraceId: typeof (result as any).traceId === "string" && (result as any).traceId.length > 0,
  hasRunId: typeof (result as any).runId === "string" && (result as any).runId.length > 0,
  hasUsage: typeof (result as any).usage === "object" && (result as any).usage !== null,
  hasSteps: Array.isArray((result as any).steps) && (result as any).steps.length > 0,
});

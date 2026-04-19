// Test: createToolCallAccuracyScorerLLM — judges tool-selection quality.
// Signature differs from other LLM scorers: { model, availableTools: Tool[] }.
import { createToolCallAccuracyScorerLLM, createTool, z } from "agent";
import { model, output } from "kit";

const sampleTool = createTool({
  id: "echo",
  description: "Echo input",
  inputSchema: z.object({ text: z.string() }),
  outputSchema: z.object({ text: z.string() }),
  execute: async ({ context }: any) => ({ text: context.text }),
});

const s = createToolCallAccuracyScorerLLM({
  model: model("openai", "gpt-4o-mini"),
  availableTools: [sampleTool],
});
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});

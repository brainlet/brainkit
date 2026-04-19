import { ToolSearchProcessor, createTool, z } from "agent";
import { output } from "kit";

const t = createTool({
  id: "sample",
  description: "Sample tool",
  inputSchema: z.object({}),
  outputSchema: z.object({ ok: z.boolean() }),
  execute: async () => ({ ok: true }),
});

const p = new ToolSearchProcessor({ tools: { sample: t }, search: { topK: 3 } });
output({ id: p.id, hasProcessInputStep: typeof (p as any).processInputStep === "function" });

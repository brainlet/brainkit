// Test: createTool with requireApproval: true — constructing
// the tool carries the flag through so the agent runtime
// knows to suspend on tool calls.
import { createTool, z } from "agent";
import { output } from "kit";

const guarded = createTool({
  id: "guarded-op",
  description: "A guarded operation",
  inputSchema: z.object({ action: z.string() }),
  outputSchema: z.object({ done: z.boolean() }),
  requireApproval: true,
  execute: async () => ({ done: true }),
});

output({
  id: guarded.id,
  requiresApproval: (guarded as any).requireApproval === true,
  hasExecute: typeof guarded.execute === "function",
});

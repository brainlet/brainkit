// Test: advanced tool execution receives an AbortSignal through the execution
// context. This locks the docs-supported cancellation shape without needing an
// agent/model call.
import { createTool, z } from "agent";
import { output } from "kit";

const cancellable = createTool({
  id: "cancellable",
  description: "Reports whether the caller already aborted",
  inputSchema: z.object({ value: z.string() }),
  outputSchema: z.object({ value: z.string(), aborted: z.boolean() }),
  execute: async (input: any, ctx: any) => ({
    value: input.value,
    aborted: ctx?.abortSignal?.aborted === true,
  }),
});

const controller = new AbortController();
controller.abort();

const result = await (cancellable as any).execute(
  { value: "payload" },
  { abortSignal: controller.signal },
);

output({
  value: result.value,
  aborted: result.aborted,
});

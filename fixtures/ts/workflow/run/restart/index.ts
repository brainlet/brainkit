// Test: run.restart — API surface.
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const step = createStep({
  id: "noop",
  inputSchema: z.any(),
  outputSchema: z.object({ ok: z.boolean() }),
  execute: async () => ({ ok: true }),
});

const wf = createWorkflow({
  id: "restart-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
}).then(step).commit();

const run = await wf.createRun();
const hasRestart = typeof (run as any).restart === "function";
await run.start({ inputData: {} });
output({ hasRestart });

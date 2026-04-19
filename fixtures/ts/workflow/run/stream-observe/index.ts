// Test: run.stream / run.observeStream API-surface presence.
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const step = createStep({
  id: "noop",
  inputSchema: z.any(),
  outputSchema: z.object({ ok: z.boolean() }),
  execute: async () => ({ ok: true }),
});

const wf = createWorkflow({
  id: "stream-observe-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
}).then(step).commit();

const run = await wf.createRun();
const hasStream = typeof (run as any).stream === "function";
const hasObserveStream = typeof (run as any).observeStream === "function";
await run.start({ inputData: {} });

output({ hasStream, hasObserveStream });

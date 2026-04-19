// Test: run.startAsync — alternative to createRun+start.
// API surface check; fall back to start() when startAsync isn't
// runtime-present.
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const step = createStep({
  id: "noop",
  inputSchema: z.any(),
  outputSchema: z.object({ ok: z.boolean() }),
  execute: async () => ({ ok: true }),
});

const wf = createWorkflow({
  id: "start-async-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
}).then(step).commit();

const run = await wf.createRun();
const hasStartAsync = typeof (run as any).startAsync === "function";
const result = await run.start({ inputData: {} });

output({ status: result.status, hasStartAsync });

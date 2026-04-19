// Test: run.resumeStream exists (presence check). Full round-trip
// requires a suspended run; covered by other suspend-resume fixtures.
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const step = createStep({
  id: "noop",
  inputSchema: z.any(),
  outputSchema: z.object({ ok: z.boolean() }),
  execute: async () => ({ ok: true }),
});

const wf = createWorkflow({
  id: "resume-stream-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
}).then(step).commit();

const run = await wf.createRun();
const hasResumeStream = typeof (run as any).resumeStream === "function";
await run.start({ inputData: {} });
output({ hasResumeStream });

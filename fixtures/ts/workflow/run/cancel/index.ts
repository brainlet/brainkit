// Test: run.cancel surface.
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const step = createStep({
  id: "noop",
  inputSchema: z.any(),
  outputSchema: z.object({ ok: z.boolean() }),
  execute: async () => ({ ok: true }),
});

const wf = createWorkflow({
  id: "cancel-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
}).then(step).commit();

const run = await wf.createRun();
const hasCancel = typeof run.cancel === "function";
await run.start({ inputData: {} });

// Safe to call after completion — should not throw.
let cancelThrew = false;
try { run.cancel(); } catch (_) { cancelThrew = true; }

output({ hasCancel, cancelSafeAfterCompletion: !cancelThrew });

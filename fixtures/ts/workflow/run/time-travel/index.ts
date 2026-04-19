// Test: run.timeTravel + timeTravelStream API-surface presence.
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const step = createStep({
  id: "noop",
  inputSchema: z.any(),
  outputSchema: z.object({ ok: z.boolean() }),
  execute: async () => ({ ok: true }),
});

const wf = createWorkflow({
  id: "time-travel-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
}).then(step).commit();

const run = await wf.createRun();
const hasTimeTravel = typeof (run as any).timeTravel === "function";
const hasTimeTravelStream = typeof (run as any).timeTravelStream === "function";
await run.start({ inputData: {} });
output({ hasTimeTravel, hasTimeTravelStream });

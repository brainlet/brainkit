// Test: .sleepUntil({date}) — pause until a specific date/time.
// Use a date 30ms in the future so the test doesn't wait long.
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const emitStep = createStep({
  id: "emit",
  inputSchema: z.any(),
  outputSchema: z.object({ marker: z.string() }),
  execute: async () => ({ marker: "after-sleep" }),
});

const target = new Date(Date.now() + 30);

const workflow = createWorkflow({
  id: "sleep-until-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
})
  .sleepUntil(target)
  .then(emitStep)
  .commit();

const start = Date.now();
const run = await workflow.createRun();
const result = await run.start({ inputData: {} });
const elapsed = Date.now() - start;

output({
  status: result.status,
  marker: result.result?.marker,
  waitedEnough: elapsed >= 20,
});

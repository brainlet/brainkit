// Test: .sleep() — pause workflow execution
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const beforeStep = createStep({
  id: "before",
  inputSchema: z.object({}),
  outputSchema: z.object({ time: z.number() }),
  execute: async () => ({ time: Date.now() }),
});

const afterStep = createStep({
  id: "after",
  inputSchema: z.object({ time: z.number() }),
  outputSchema: z.object({ elapsed: z.number() }),
  execute: async ({ inputData }) => ({
    elapsed: Date.now() - inputData.time,
  }),
});

const workflow = createWorkflow({
  id: "sleep-wf",
  inputSchema: z.object({}),
  outputSchema: z.object({ elapsed: z.number() }),
})
  .then(beforeStep)
  .sleep(100)
  .then(afterStep)
  .commit();

const run = await workflow.createRun();
const result = await run.start({ inputData: {} });

output({
  status: result.status,
  elapsed: result.result?.elapsed,
  sleptEnough: result.result?.elapsed >= 50,
});

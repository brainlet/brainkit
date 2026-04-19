// Test: .map() — remap inputs / outputs between steps.
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const produce = createStep({
  id: "produce",
  inputSchema: z.any(),
  outputSchema: z.object({ value: z.number() }),
  execute: async () => ({ value: 7 }),
});

// Consumer expects `n`, not `value`.
const consume = createStep({
  id: "consume",
  inputSchema: z.object({ n: z.number() }),
  outputSchema: z.object({ doubled: z.number() }),
  execute: async ({ inputData }: any) => ({ doubled: inputData.n * 2 }),
});

const workflow = createWorkflow({
  id: "map-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
})
  .then(produce)
  .map(({ inputData }: any) => ({ n: inputData.value }))
  .then(consume)
  .commit();

const run = await workflow.createRun();
const result = await run.start({ inputData: {} });

output({
  status: result.status,
  doubled: result.result?.doubled,
  correct: result.result?.doubled === 14,
});

// Test: .foreach() — iterate over array from previous step
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const produceStep = createStep({
  id: "produce",
  inputSchema: z.object({ count: z.number() }),
  outputSchema: z.array(z.object({ n: z.number() })),
  execute: async ({ inputData }) => {
    return Array.from({ length: inputData.count }, (_, i) => ({ n: i + 1 }));
  },
});

const processStep = createStep({
  id: "process",
  inputSchema: z.object({ n: z.number() }),
  outputSchema: z.object({ doubled: z.number() }),
  execute: async ({ inputData }) => ({ doubled: inputData.n * 2 }),
});

const workflow = createWorkflow({
  id: "foreach-wf",
  inputSchema: z.object({ count: z.number() }),
  outputSchema: z.any(),
})
  .then(produceStep)
  .forEach({ items: "items", step: processStep })
  .commit();

const run = await workflow.createRun();
const result = await run.start({ inputData: { count: 3 } });

output({
  status: result.status,
  result: result.result,
  isArray: Array.isArray(result.result),
});

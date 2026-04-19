// Test: .dowhile() — loop while condition is true (check BEFORE each step).
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const stateSchema = z.object({ counter: z.number().optional() });

const incrementStep = createStep({
  id: "increment",
  inputSchema: z.any(),
  outputSchema: z.object({ value: z.number() }),
  stateSchema,
  execute: async ({ state, setState }: any) => {
    const next = (state.counter || 0) + 1;
    await setState({ counter: next });
    return { value: next };
  },
});

const workflow = createWorkflow({
  id: "dowhile-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
  stateSchema,
})
  .dowhile(
    incrementStep,
    async ({ state }: any) => (state.counter || 0) < 3,
  )
  .commit();

const run = await workflow.createRun();
const result = await run.start({ inputData: {} });

output({
  status: result.status,
  loopedCorrectly: result.result?.value === 3,
});

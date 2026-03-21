// Test: .dountil() — loop until condition is met
import { createWorkflow, createStep, z, output } from "kit";

const stateSchema = z.object({ counter: z.number().optional() });

const incrementStep = createStep({
  id: "increment",
  inputSchema: z.any(),
  outputSchema: z.object({ value: z.number() }),
  stateSchema: stateSchema,
  execute: async ({ state, setState }) => {
    const next = (state.counter || 0) + 1;
    await setState({ counter: next });
    return { value: next };
  },
});

const workflow = createWorkflow({
  id: "loop-wf",
  inputSchema: z.object({}),
  outputSchema: z.any(),
  stateSchema: stateSchema,
})
  .dountil(
    incrementStep,
    async ({ state }) => (state.counter || 0) >= 5,
  )
  .commit();

const run = await workflow.createRun();
const result = await run.start({ inputData: {} });

output({
  status: result.status,
  result: result.result,
  loopedCorrectly: result.result?.value === 5,
});

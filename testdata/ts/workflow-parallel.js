// Test: .parallel() — run two steps concurrently
import { createWorkflow, createStep, z, output } from "brainlet";

const stepA = createStep({
  id: "step-a",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ doubled: z.number() }),
  execute: async ({ inputData }) => ({ doubled: inputData.value * 2 }),
});

const stepB = createStep({
  id: "step-b",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ tripled: z.number() }),
  execute: async ({ inputData }) => ({ tripled: inputData.value * 3 }),
});

const collectStep = createStep({
  id: "collect",
  inputSchema: z.object({}),
  outputSchema: z.object({ sum: z.number() }),
  execute: async ({ inputData, getStepResult }) => {
    const a = getStepResult("step-a");
    const b = getStepResult("step-b");
    return { sum: (a?.doubled || 0) + (b?.tripled || 0) };
  },
});

const workflow = createWorkflow({
  id: "parallel-wf",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ sum: z.number() }),
})
  .parallel([stepA, stepB])
  .then(collectStep)
  .commit();

const run = await workflow.createRun();
const result = await run.start({ inputData: { value: 5 } });

output({
  status: result.status,
  result: result.result,
  correct: result.status === "success" && result.result?.sum === 25,
});

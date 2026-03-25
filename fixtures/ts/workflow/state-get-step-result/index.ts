// Test: getStepResult() accesses previous step's output
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const step1 = createStep({
  id: "compute",
  inputSchema: z.object({ x: z.number() }),
  outputSchema: z.object({ doubled: z.number() }),
  execute: async ({ inputData }) => ({ doubled: inputData.x * 2 }),
});

const step2 = createStep({
  id: "verify",
  inputSchema: z.object({}),
  outputSchema: z.object({ fromStep1: z.number(), correct: z.boolean() }),
  execute: async ({ getStepResult }) => {
    const prev = getStepResult("compute") as any;
    return { fromStep1: prev.doubled, correct: prev.doubled === 42 };
  },
});

const wf = createWorkflow({
  id: "step-result-test",
  inputSchema: z.object({ x: z.number() }),
  outputSchema: z.object({ fromStep1: z.number(), correct: z.boolean() }),
}).then(step1).then(step2).commit();

const run = await wf.createRun();
const result = await run.start({ inputData: { x: 21 } });

output({
  status: result.status,
  fromStep1: result.steps?.verify?.output?.fromStep1,
  correct: result.steps?.verify?.output?.correct,
});

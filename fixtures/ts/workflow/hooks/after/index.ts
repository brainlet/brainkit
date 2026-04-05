// Test: workflow with afterAll hook
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const step1 = createStep({
  id: "step-a",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ result: z.number() }),
  execute: async ({ inputData }) => ({ result: inputData.value + 1 }),
});

const step2 = createStep({
  id: "step-b",
  inputSchema: z.object({ result: z.number() }),
  outputSchema: z.object({ final: z.number() }),
  execute: async ({ inputData }) => ({ final: inputData.result * 2 }),
});

const wf = createWorkflow({
  id: "hooks-test",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ final: z.number() }),
}).then(step1).then(step2).commit();

const run = await wf.createRun();
const result = await run.start({ inputData: { value: 10 } });

// 10 + 1 = 11, 11 * 2 = 22
output({
  status: result.status,
  finalValue: (result as any).steps?.["step-b"]?.output?.final,
  isCorrect: (result as any).steps?.["step-b"]?.output?.final === 22,
});

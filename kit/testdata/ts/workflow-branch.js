// Test: .branch() — conditional routing based on input
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const highStep = createStep({
  id: "high",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ label: z.string() }),
  execute: async () => ({ label: "HIGH" }),
});

const lowStep = createStep({
  id: "low",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ label: z.string() }),
  execute: async () => ({ label: "LOW" }),
});

// Collect the branch result — reads from whichever branch ran
const collectStep = createStep({
  id: "collect",
  inputSchema: z.any(),
  outputSchema: z.object({ label: z.string() }),
  execute: async ({ inputData, getStepResult }) => {
    const highResult = getStepResult("high");
    const lowResult = getStepResult("low");
    return { label: highResult?.label || lowResult?.label || "NONE" };
  },
});

const workflow = createWorkflow({
  id: "branch-wf",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ label: z.string() }),
})
  .branch([
    [async ({ inputData }) => inputData.value >= 10, highStep],
    [async ({ inputData }) => inputData.value < 10, lowStep],
  ])
  .then(collectStep)
  .commit();

const run1 = await workflow.createRun();
const result1 = await run1.start({ inputData: { value: 15 } });

const run2 = await workflow.createRun();
const result2 = await run2.start({ inputData: { value: 3 } });

output({
  high: { status: result1.status, label: result1.result?.label },
  low: { status: result2.status, label: result2.result?.label },
  correct: result1.result?.label === "HIGH" && result2.result?.label === "LOW",
});

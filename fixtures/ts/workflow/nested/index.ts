// Test: nested workflow — outer workflow calls inner workflow as a step
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

// Inner workflow: doubles a number
const doubleStep = createStep({
  id: "double",
  inputSchema: z.object({ x: z.number() }),
  outputSchema: z.object({ doubled: z.number() }),
  execute: async ({ inputData }) => ({ doubled: inputData.x * 2 }),
});

const innerWf = createWorkflow({
  id: "inner-doubler",
  inputSchema: z.object({ x: z.number() }),
  outputSchema: z.object({ doubled: z.number() }),
}).then(doubleStep).commit();

// Outer workflow: adds 10 then calls inner
const addStep = createStep({
  id: "add-ten",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ x: z.number() }),
  execute: async ({ inputData }) => ({ x: inputData.value + 10 }),
});

const callInnerStep = createStep({
  id: "call-inner",
  inputSchema: z.object({ x: z.number() }),
  outputSchema: z.object({ result: z.number() }),
  execute: async ({ inputData }) => {
    const run = await innerWf.createRun();
    const innerResult = await run.start({ inputData: { x: inputData.x } });
    const doubled = (innerResult as any).steps?.double?.output?.doubled || 0;
    return { result: doubled };
  },
});

const outerWf = createWorkflow({
  id: "outer-composite",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({ result: z.number() }),
}).then(addStep).then(callInnerStep).commit();

const run = await outerWf.createRun();
const result = await run.start({ inputData: { value: 11 } });

// 11 + 10 = 21, 21 * 2 = 42
output({
  status: result.status,
  isCompleted: result.status === "completed",
  finalResult: (result as any).steps?.["call-inner"]?.output?.result,
  is42: (result as any).steps?.["call-inner"]?.output?.result === 42,
});

// Test: workflow step throws error — workflow reports failed status
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const failStep = createStep({
  id: "will-fail",
  inputSchema: z.object({ shouldFail: z.boolean() }),
  outputSchema: z.object({ message: z.string() }),
  execute: async ({ inputData }) => {
    if (inputData.shouldFail) {
      throw new Error("intentional failure for testing");
    }
    return { message: "success" };
  },
});

const wf = createWorkflow({
  id: "error-handling-test",
  inputSchema: z.object({ shouldFail: z.boolean() }),
  outputSchema: z.object({ message: z.string() }),
}).then(failStep).commit();

// Test 1: step fails
const run1 = await wf.createRun();
const result1 = await run1.start({ inputData: { shouldFail: true } });

// Test 2: step succeeds
const run2 = await wf.createRun();
const result2 = await run2.start({ inputData: { shouldFail: false } });

output({
  failedStatus: result1.status,
  isFailed: result1.status === "failed",
  successStatus: result2.status,
  isSuccess: result2.status === "completed",
});

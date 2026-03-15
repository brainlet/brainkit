// Test: basic workflow with createWorkflow + createStep
import { createWorkflow, createStep, z, output } from "brainlet";

// Step 1: format a message
const formatStep = createStep({
  id: "format",
  inputSchema: z.object({ message: z.string() }),
  outputSchema: z.object({ formatted: z.string() }),
  execute: async ({ inputData }) => {
    return { formatted: inputData.message.toUpperCase() };
  },
});

// Step 2: add emphasis
const emphasizeStep = createStep({
  id: "emphasize",
  inputSchema: z.object({ formatted: z.string() }),
  outputSchema: z.object({ result: z.string() }),
  execute: async ({ inputData }) => {
    return { result: inputData.formatted + "!!!" };
  },
});

// Create and commit the workflow
const workflow = createWorkflow({
  id: "test-workflow",
  inputSchema: z.object({ message: z.string() }),
  outputSchema: z.object({ result: z.string() }),
})
  .then(formatStep)
  .then(emphasizeStep)
  .commit();

// Run it
const run = await workflow.createRun();
const result = await run.start({ inputData: { message: "hello world" } });

output({
  status: result.status,
  result: result.result,
  expected: "HELLO WORLD!!!",
  match: result.status === "success" && result.result?.result === "HELLO WORLD!!!",
});

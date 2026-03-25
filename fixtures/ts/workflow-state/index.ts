// Test: workflow shared state via setState/state across steps
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

// Shared state schema — accumulates data across steps
const stateSchema = z.object({
  items: z.array(z.string()).optional(),
  count: z.number().optional(),
});

// Step 1: initialize state
const initStep = createStep({
  id: "init",
  inputSchema: z.object({ prefix: z.string() }),
  outputSchema: z.object({}),
  stateSchema: stateSchema,
  execute: async ({ inputData, setState }) => {
    await setState({ items: [inputData.prefix + "-first"], count: 1 });
    return {};
  },
});

// Step 2: accumulate into state (reads previous state, adds to it)
const accumulateStep = createStep({
  id: "accumulate",
  inputSchema: z.object({}),
  outputSchema: z.object({}),
  stateSchema: stateSchema,
  execute: async ({ state, setState }) => {
    const items = state.items || [];
    await setState({
      items: [...items, "second", "third"],
      count: (state.count || 0) + 2,
    });
    return {};
  },
});

// Step 3: read final state and output it
const readStep = createStep({
  id: "read",
  inputSchema: z.object({}),
  outputSchema: z.object({ items: z.array(z.string()), count: z.number() }),
  stateSchema: stateSchema,
  execute: async ({ state }) => {
    return { items: state.items || [], count: state.count || 0 };
  },
});

const workflow = createWorkflow({
  id: "state-workflow",
  inputSchema: z.object({ prefix: z.string() }),
  outputSchema: z.object({ items: z.array(z.string()), count: z.number() }),
  stateSchema: stateSchema,
})
  .then(initStep)
  .then(accumulateStep)
  .then(readStep)
  .commit();

const run = await workflow.createRun();
const result = await run.start({
  inputData: { prefix: "test" },
});

output({
  status: result.status,
  result: result.result,
  hasItems: Array.isArray(result.result?.items) && result.result.items.length === 3,
  hasCount: result.result?.count === 3,
  items: result.result?.items,
  firstItem: result.result?.items?.[0] === "test-first",
});

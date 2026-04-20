// `cloneWorkflow(wf, { id })` and `cloneStep(step, { id })` let you
// reuse a workflow / step under a fresh ID — independent runs and
// clean separation in logs and observability tools. Useful when you
// want to drive the same pipeline with different configs or tracing
// contexts without rebuilding it.
import { createWorkflow, createStep, cloneWorkflow, cloneStep, z } from "agent";
import { output } from "kit";

const add = createStep({
  id: "add",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  outputSchema: z.object({ sum: z.number() }),
  execute: async ({ inputData }) => ({ sum: inputData.a + inputData.b }),
});

const addClone = cloneStep(add, { id: "add-v2" });

const original = createWorkflow({
  id: "addition",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  outputSchema: z.object({ sum: z.number() }),
})
  .then(add)
  .commit();

const cloned = cloneWorkflow(original, { id: "addition-v2" });

output({
  originalId: original.id,
  clonedId: cloned.id,
  idsDiffer: original.id !== cloned.id,
  bothAreWorkflows:
    typeof original.createRun === "function" &&
    typeof cloned.createRun === "function",
  stepOriginalId: (add as any).id,
  stepClonedId: (addClone as any).id,
  stepIdsDiffer: (add as any).id !== (addClone as any).id,
});

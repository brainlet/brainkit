// Test: tool with outputSchema — Mastra validates the result
// shape at call time. Exercises the WorkflowToolExecutionContext
// slice via a type probe, proving the canonical
//   ToolExecutionContext { workflow?: WorkflowToolExecutionContext }
// wiring is importable end-to-end.
import { createTool, z } from "agent";
import type { WorkflowToolExecutionContext } from "agent";
import { output } from "kit";

const profile = createTool<
  "profile",
  { id: string },
  { id: string; name: string; age: number }
>({
  id: "profile",
  description: "Return a user profile",
  inputSchema: z.object({ id: z.string() }),
  outputSchema: z.object({
    id: z.string(),
    name: z.string(),
    age: z.number(),
  }),
  // ↓ No parameter type annotation — inferred from the generic
  //   slot as `{ id: string }`.
  execute: async ({ id }) => ({ id, name: "Bob", age: 30 }),
});

// Compile-time probe: workflow slice of ToolExecutionContext
// carries the canonical runId / workflowId / state / setState
// / suspend / resumeData fields.
const _workflowSlice: WorkflowToolExecutionContext<unknown, unknown> | undefined = undefined;
void _workflowSlice;

const res = await profile.execute!({ id: "u-1" });

output({
  id: (res as { id: string }).id,
  name: (res as { name: string }).name,
  age: (res as { age: number }).age,
  schemaCarried: !!profile.outputSchema,
});

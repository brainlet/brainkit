// Test: workflow steps receive per-run context through Mastra's current
// RequestContext plumbing, while Brainkit exposes the docs-facing
// RuntimeContext alias for user code.
import { RuntimeContext, RequestContext, createStep, createWorkflow, z } from "agent";
import { output } from "kit";

type UserContext = {
  "user-tier": "enterprise" | "pro";
};

const step = createStep({
  id: "read-context",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.object({
    tier: z.string(),
    doubled: z.number(),
    hasContext: z.boolean(),
  }),
  execute: async ({ inputData, requestContext, runtimeContext }: any) => {
    const ctx = runtimeContext || requestContext;
    return {
      tier: ctx?.get?.("user-tier") || "missing",
      doubled: inputData.value * 2,
      hasContext: !!ctx,
    };
  },
});

const workflow = createWorkflow({
  id: "runtime-context-workflow",
  inputSchema: z.object({ value: z.number() }),
  outputSchema: z.any(),
}).then(step).commit();

const ctx = new RuntimeContext<UserContext>();
ctx.set("user-tier", "pro");

const run = await workflow.createRun();
const result = await run.start({
  inputData: { value: 21 },
  requestContext: ctx,
});

output({
  status: result.status,
  tier: result.result?.tier,
  doubled: result.result?.doubled,
  hasContext: result.result?.hasContext,
  runtimeAliasMatchesRequestContext: RuntimeContext === RequestContext,
});

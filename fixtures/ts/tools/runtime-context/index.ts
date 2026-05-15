// Test: Mastra docs' RuntimeContext name is accepted at the Brainkit TS
// surface, and createTool can read per-call context data on direct execution.
import { RuntimeContext, RequestContext, createTool, z } from "agent";
import { output } from "kit";

type UserContext = {
  "user-tier": "enterprise" | "pro";
};

const runtimeContext = new RuntimeContext<UserContext>();
runtimeContext.set("user-tier", "enterprise");

const tool = createTool({
  id: "context-aware-tool",
  description: "Reads a request-scoped value",
  inputSchema: z.object({ value: z.string() }),
  outputSchema: z.object({ tier: z.string(), value: z.string(), hasContext: z.boolean() }),
  execute: async (input: any, execContext: any) => {
    const ctx = execContext?.runtimeContext || execContext?.requestContext;
    return {
      tier: ctx?.get?.("user-tier") || "missing",
      value: input.value || "missing",
      hasContext: !!ctx,
    };
  },
});

const result = await (tool as any).execute(
  { value: "payload" },
  {
  runtimeContext,
  },
);

output({
  tier: result.tier,
  value: result.value,
  hasContext: result.hasContext,
  runtimeAliasMatchesRequestContext: RuntimeContext === RequestContext,
});

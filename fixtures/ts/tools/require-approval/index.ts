// Test: createTool with requireApproval — suspend + approve
// round-trip exercised directly through the tool's execute
// function. The first call runs without resumeData → the tool
// invokes context.agent.suspend(payload) → the Tool class
// records the suspend data. The second call passes resumeData
// through the agent context → execute returns the approved
// output. Proves the canonical
//   ToolExecutionContext → AgentToolExecutionContext { suspend, resumeData }
// shape wires end-to-end at the type + runtime level — no AI
// round-trip required.
import { createTool, z } from "agent";
import type { AgentToolExecutionContext, ToolExecutionContext } from "agent";
import { output } from "kit";

const guarded = createTool<
  "guarded-op",
  { action: string },
  { done: boolean; action: string },
  { reason: string },
  { approved: boolean }
>({
  id: "guarded-op",
  description: "A guarded operation",
  inputSchema: z.object({ action: z.string() }),
  outputSchema: z.object({ done: z.boolean(), action: z.string() }),
  suspendSchema: z.object({ reason: z.string() }),
  resumeSchema: z.object({ approved: z.boolean() }),
  requireApproval: true,
  execute: async ({ action }, ctx) => {
    const resume = ctx?.agent?.resumeData;
    if (!resume) {
      // Phase 1: no resumeData — emit a suspend payload so the
      // surrounding runtime can route it to an approver.
      await ctx?.agent?.suspend({ reason: `approval needed for: ${action}` });
      return { done: false, action };
    }
    // Phase 2: resumeData carries the approval decision.
    return { done: resume.approved, action };
  },
});

// ── Phase 1: execute WITHOUT resumeData → suspend fires. ───────
let suspendPayload: { reason: string } | null = null;
const phase1Ctx: ToolExecutionContext<{ reason: string }, { approved: boolean }> = {
  agent: {
    toolCallId: "tc-1",
    messages: [],
    suspend: async (payload) => {
      suspendPayload = payload;
    },
  } satisfies AgentToolExecutionContext<{ reason: string }, { approved: boolean }>,
};
const phase1 = await guarded.execute!({ action: "delete-record" }, phase1Ctx);

// ── Phase 2: execute WITH resumeData → approved path runs. ─────
const phase2Ctx: ToolExecutionContext<{ reason: string }, { approved: boolean }> = {
  agent: {
    toolCallId: "tc-1",
    messages: [],
    suspend: async () => {},
    resumeData: { approved: true },
  },
};
const phase2 = await guarded.execute!({ action: "delete-record" }, phase2Ctx);

output({
  id: guarded.id,
  requiresApproval: guarded.requireApproval === true,
  hasExecute: typeof guarded.execute === "function",
  suspended: suspendPayload !== null,
  suspendReason: (suspendPayload as { reason: string } | null)?.reason ?? "",
  phase1Done: (phase1 as { done: boolean }).done,
  approved: (phase2 as { done: boolean }).done === true,
  phase2Action: (phase2 as { action: string }).action,
});

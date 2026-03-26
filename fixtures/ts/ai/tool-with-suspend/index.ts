// Test: Agent HITL — tool with requireApproval suspends, then approve/decline resumes
// Mastra pattern: createTool({ requireApproval: true }) → agent.generate returns
// finishReason:"suspended" with suspendPayload → agent.approveToolCallGenerate resumes
import { Agent, createTool, z, InMemoryStore, Memory } from "agent";
import { model, output } from "kit";

const deleteTool = createTool({
  id: "delete-record",
  description: "Delete a record by ID — requires human approval",
  inputSchema: z.object({ id: z.string() }),
  outputSchema: z.object({ deleted: z.boolean() }),
  requireApproval: true,
  execute: async ({ id }: any) => {
    return { deleted: true };
  },
});

// HITL requires a storage provider for snapshot persistence
const store = new InMemoryStore();
const mem = new Memory({ storage: store });

const agent = new Agent({
  name: "hitl-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "When asked to delete, use the delete-record tool.",
  tools: { "delete-record": deleteTool },
  memory: mem,
  maxSteps: 3,
});

try {
  // Phase 1: generate — should suspend because tool requires approval
  const suspended = await agent.generate("Delete record abc-123", {
    requireToolApproval: true,
  });

  const isSuspended = suspended.finishReason === "suspended";
  const hasSuspendPayload = suspended.suspendPayload !== undefined && suspended.suspendPayload !== null;
  const hasRunId = typeof suspended.runId === "string" && suspended.runId.length > 0;

  if (isSuspended && hasRunId) {
    // Phase 2: approve the tool call
    try {
      const payload = suspended.suspendPayload as { toolCallId: string };
      const approved = await agent.approveToolCallGenerate({
        runId: suspended.runId!,
        toolCallId: payload.toolCallId,
      });

      output({
        suspended: true,
        hasSuspendPayload,
        hasRunId,
        approved: true,
        finalText: approved?.text?.substring(0, 100) || "",
      });
    } catch (e: any) {
      // Approve might fail if snapshot storage isn't fully wired
      output({
        suspended: true,
        hasSuspendPayload,
        hasRunId,
        approveError: e.message.substring(0, 200),
      });
    }
  } else {
    // Model didn't suspend — might not have called the tool
    output({
      suspended: false,
      finishReason: suspended.finishReason,
      hasText: suspended.text.length > 0,
      text: suspended.text.substring(0, 100),
    });
  }
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}

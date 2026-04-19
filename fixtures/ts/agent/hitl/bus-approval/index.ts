// Test: bus-based HITL — generateWithApproval routes tool approval through the bus.
// The Go test runner subscribes to "test.approvals" and auto-approves.
//
// approveToolCallGenerate requires workflow-snapshot persistence to resume
// from the exact suspend point. A bare `new Agent({...})` has no attached
// Mastra instance, so its #mastra is undefined and the snapshot lookup
// short-circuits — the resume silently returns the same suspended state.
//
// The production shape is `new Mastra({ agents: { ... }, storage: ... })`;
// once the agent is reached through a Mastra instance the workflow store
// resolves and the approve/decline path actually advances the run.
import { Agent, Mastra, createTool, z, InMemoryStore } from "agent";
import { model, generateWithApproval, output } from "kit";

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

const hitlAgent = new Agent({
  name: "hitl-bus-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions:
    "ALWAYS use the delete-record tool when asked to delete anything. " +
    "Never ask for confirmation — just call the tool immediately.",
  tools: { "delete-record": deleteTool },
  maxSteps: 3,
});

// Wrap the agent in a Mastra instance with InMemoryStore so the agentic-loop
// workflow can save + load the resume snapshot used by approveToolCallGenerate.
const mastra = new Mastra({
  agents: { "hitl-bus-agent": hitlAgent },
  storage: new InMemoryStore(),
});

const agent = mastra.getAgent("hitl-bus-agent");

try {
  const result = await generateWithApproval(agent, "Delete record xyz-789", {
    approvalTopic: "test.approvals",
    timeout: 10000,
  });

  const toolResult = (result.steps || [])
    .flatMap((s: any) => s.toolResults || [])
    .find((tr: any) => tr.toolName === "delete-record" || tr.name === "delete-record");

  output({
    approved: true,
    finishReason: result.finishReason,
    hasText: (result.text || "").length > 0,
    toolExecuted: !!toolResult,
    toolDeleted:
      toolResult?.result?.deleted === true || toolResult?.output?.deleted === true,
    stepCount: (result.steps || []).length,
  });
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}

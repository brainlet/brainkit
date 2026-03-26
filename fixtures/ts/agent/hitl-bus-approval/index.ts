// Test: bus-based HITL — generateWithApproval routes tool approval through the bus
// The Go test runner subscribes to "test.approvals" and auto-approves.
import { Agent, createTool, z, InMemoryStore, Memory } from "agent";
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

const store = new InMemoryStore();
const mem = new Memory({ storage: store });

const agent = new Agent({
  name: "hitl-bus-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "ALWAYS use the delete-record tool when asked to delete anything. Never ask for confirmation — just call the tool immediately.",
  tools: { "delete-record": deleteTool },
  memory: mem,
  maxSteps: 3,
});

try {
  // generateWithApproval handles the full suspend → bus → approve loop.
  // The Go test runner subscribes to "test.approvals" and auto-approves.
  const result = await generateWithApproval(agent, "Delete record xyz-789", {
    approvalTopic: "test.approvals",
    timeout: 10000,
  });

  output({
    text: result.text,
    hasText: result.text.length > 0,
    finishReason: result.finishReason,
    approved: true, // If we get here, approval succeeded
  });
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}

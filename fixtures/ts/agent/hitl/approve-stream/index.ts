// Test: Agent.approveToolCall (stream variant) API surface.
import { Agent } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "hitl-approve-stream",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply briefly.",
});

output({
  hasApproveToolCall: typeof (agent as any).approveToolCall === "function",
});

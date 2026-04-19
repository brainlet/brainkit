// Test: Agent.approveNetworkToolCall / declineNetworkToolCall surface.
import { Agent } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "hitl-net-approval",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply briefly.",
});

output({
  hasApproveNetworkToolCall: typeof (agent as any).approveNetworkToolCall === "function",
  hasDeclineNetworkToolCall: typeof (agent as any).declineNetworkToolCall === "function",
});

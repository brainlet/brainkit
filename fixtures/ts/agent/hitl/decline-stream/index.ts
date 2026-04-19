// Test: Agent.declineToolCall (stream variant) API surface.
import { Agent } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "hitl-decline-stream",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply briefly.",
});

output({
  hasDeclineToolCall: typeof (agent as any).declineToolCall === "function",
});

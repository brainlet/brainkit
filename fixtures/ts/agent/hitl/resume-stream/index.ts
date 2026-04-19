// Test: Agent.resumeStream + streaming HITL method surface.
import { Agent } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "hitl-resume-stream",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply briefly.",
});

output({
  hasResumeStream: typeof (agent as any).resumeStream === "function",
  hasApproveToolCall: typeof (agent as any).approveToolCall === "function",
  hasDeclineToolCall: typeof (agent as any).declineToolCall === "function",
});

// Test: Agent.resumeGenerate API surface — covers the
// declaration even when no suspended run is available to
// resume (real round-trip is in bus-approval).
import { Agent } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "hitl-resume-gen",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply briefly.",
});

output({
  hasResumeGenerate: typeof (agent as any).resumeGenerate === "function",
  hasResumeStream: typeof (agent as any).resumeStream === "function",
  hasApproveToolCallGenerate: typeof (agent as any).approveToolCallGenerate === "function",
  hasDeclineToolCallGenerate: typeof (agent as any).declineToolCallGenerate === "function",
});

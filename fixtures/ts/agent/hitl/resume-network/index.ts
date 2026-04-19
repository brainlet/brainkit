// Test: Agent.resumeNetwork surface.
import { Agent } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "hitl-resume-net",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply briefly.",
});

output({
  hasResumeNetwork: typeof (agent as any).resumeNetwork === "function",
});

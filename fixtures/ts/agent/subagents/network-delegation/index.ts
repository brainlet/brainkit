// Test: agent.network() — supervisor delegates to sub-agents
import { Agent } from "agent";
import { model, output } from "kit";

const mathAgent = new Agent({
  name: "math-expert",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a math expert. Answer math questions concisely with just the number.",
});

const supervisor = new Agent({
  name: "supervisor",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a supervisor. Delegate math questions to math-expert. Return the result.",
  agents: { "math-expert": mathAgent },
  maxSteps: 5,
});

try {
  const result = await supervisor.generate("What is 6 times 7?");
  output({
    hasText: result.text.length > 0,
    steps: result.steps?.length || 0,
    multiStep: (result.steps?.length || 0) > 1,
  });
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}

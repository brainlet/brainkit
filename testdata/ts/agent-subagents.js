// Test: Agent networks / subagent delegation
// Verifies: agents config, sub-agents become tools, supervisor delegates to sub-agents
import { agent, createTool, z, output } from "brainlet";

const results = {};

try {
  // Create a math sub-agent with a calculator tool
  const calcTool = createTool({
    id: "calculate",
    description: "Evaluates a math expression and returns the result",
    inputSchema: z.object({ expression: z.string() }),
    execute: async ({ expression }) => {
      // Simple eval for basic math
      try {
        const result = new Function("return " + expression)();
        return { result: String(result) };
      } catch(e) {
        return { error: e.message };
      }
    },
  });

  const mathAgent = agent({
    model: "openai/gpt-4o-mini",
    instructions: "You are a math specialist. Use the calculate tool to solve math problems. Always use the tool, don't compute in your head.",
    tools: { calculate: calcTool },
    maxSteps: 3,
  });

  // Create a writing sub-agent (no tools, just writes)
  const writerAgent = agent({
    model: "openai/gpt-4o-mini",
    instructions: "You are a concise writer. Summarize information in one short sentence.",
  });

  // Create a supervisor that delegates to sub-agents
  const supervisor = agent({
    model: "openai/gpt-4o-mini",
    instructions: "You are a supervisor. When asked a math question, delegate to the math agent. When asked to summarize, delegate to the writer agent. Always delegate, never answer yourself.",
    agents: { math: mathAgent, writer: writerAgent },
    maxSteps: 5,
  });

  // Test 1: Verify the supervisor can generate (sub-agents become tools)
  const r = await supervisor.generate("What is 15 * 7? Use the math agent to calculate this.");
  results.supervisorText = r.text.substring(0, 100);
  results.hasAnswer = r.text.includes("105") ? "ok" : "no 105 in: " + r.text.substring(0, 50);

  // Test 2: Verify toolCalls show agent delegation
  results.hasToolCalls = (r.toolCalls && r.toolCalls.length > 0) ? "ok" : "no tool calls";
  if (r.toolCalls && r.toolCalls.length > 0) {
    results.firstToolName = r.toolCalls[0].toolName || "unknown";
  }

  // Test 3: Check steps for delegation evidence
  results.stepCount = r.steps ? r.steps.length : 0;

  results.status = "ok";
} catch(e) {
  results.error = e.message;
  results.stack = (e.stack || "").substring(0, 300);
}

output(results);

// Test: agent uses a platform-registered tool (e.g., from a plugin)
// The "multiply" tool is registered in Go before this runs.
import { Agent } from "agent";
import { model, tool, output } from "kit";

const multiplyTool = tool("multiply");

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Always use the multiply tool. Return only the number.",
  tools: { multiply: multiplyTool },
});

const result = await a.generate("What is 6 times 7? Use the multiply tool.", { maxSteps: 3 });

output({
  text: result.text,
  toolCalls: result.toolCalls?.length || 0,
});

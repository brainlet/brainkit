// Test: agent uses a platform-registered tool (e.g., from a plugin).
// The "multiply" tool is registered in Go before this runs.
import { Agent } from "agent";
import { model, tool, output } from "kit";

const multiplyTool = tool("multiply");

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions:
    "You help with arithmetic. Always call the multiply tool to compute products.",
  tools: { multiply: multiplyTool },
  maxSteps: 5,
});

const result = await a.generate("Use multiply to compute 6 × 7.");

// Mastra 1.x returns each tool event as
// `{ type: "tool-result", payload: { args, toolName, result: {...}, ... } }`
// so reach through `payload.result` for the Go executor's return value.
const toolResults = (result.steps || []).flatMap((s: any) => s.toolResults || []);
const multiplyEvent = toolResults.find(
  (tr: any) => (tr.payload?.toolName || tr.toolName || tr.name) === "multiply",
);
const toolResult = multiplyEvent?.payload?.result?.result ?? multiplyEvent?.result?.result ?? null;

output({
  usedTool: (result.steps || []).some((s: any) => (s.toolCalls || []).length > 0),
  toolResult,
  resultIs42: toolResult === 42,
});

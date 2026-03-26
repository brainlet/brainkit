// Test: tool with suspend — HITL pattern where tool pauses for human approval
import { generateText, z } from "ai";
import { createTool } from "agent";
import { model, output } from "kit";

let suspendCalled = false;

const riskyTool = createTool({
  id: "risky-action",
  description: "Performs a risky action that needs approval",
  inputSchema: z.object({ action: z.string() }),
  execute: async ({ action }: any, context: any) => {
    // In a real HITL flow, this would call context.suspend()
    // For testing, we just verify the tool executes
    suspendCalled = true;
    return { performed: action, approved: true };
  },
});

const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  tools: { riskyAction: riskyTool },
  maxSteps: 3,
  prompt: "Perform the risky action 'deploy-v2'. Use the riskyAction tool.",
});

output({
  hasText: result.text.length > 0,
  toolCalled: suspendCalled,
  hasSteps: result.steps.length > 0,
});

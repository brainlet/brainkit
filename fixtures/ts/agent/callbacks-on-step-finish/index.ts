import { Agent, createTool, z } from "agent";
import { model, output } from "kit";
const steps: any[] = [];
const echo = createTool({ id: "echo", description: "Echo input", inputSchema: z.object({ msg: z.string() }), execute: async ({ msg }) => ({ echoed: msg }) });
const agent = new Agent({ name: "callback-agent", model: model("openai", "gpt-4o-mini"), instructions: "Use the echo tool to echo what the user says.", tools: { echo }, maxSteps: 3 });
const result = await agent.generate("Echo the word 'hello'", { onStepFinish: (step: any) => { steps.push({ type: step.stepType }); } });
output({ text: result.text, callbackFired: steps.length > 0, stepCount: steps.length });

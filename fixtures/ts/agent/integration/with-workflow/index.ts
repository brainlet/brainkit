// Test: agent with workflows field
import { Agent, createWorkflow, createStep, z } from "agent";
import { model, output } from "kit";

const processStep = createStep({
  id: "process",
  inputSchema: z.object({ text: z.string() }),
  outputSchema: z.object({ processed: z.string() }),
  execute: async ({ inputData }) => ({ processed: inputData.text.toUpperCase() }),
});

const wf = createWorkflow({
  id: "text-processor",
  inputSchema: z.object({ text: z.string() }),
  outputSchema: z.object({ processed: z.string() }),
}).then(processStep).commit();

const agent = new Agent({
  name: "wf-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You have a text processing workflow. Use it when asked to process text.",
  workflows: { "text-processor": wf },
  maxSteps: 5,
});

const result = await agent.generate("Process the text 'hello world'");

output({
  hasText: result.text.length > 0,
  text: result.text.substring(0, 200),
});

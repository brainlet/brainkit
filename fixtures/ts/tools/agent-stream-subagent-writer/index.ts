import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

const subAgentTool = createTool({
  id: "subAgentTool",
  description: "Use this tool for the sub-agent analysis task.",
  inputSchema: z.object({ task: z.string() }),
  execute: async ({ task }: any, context: any) => {
    await context?.writer?.custom({
      type: "data-sub-agent-progress",
      data: { step: "initializing", task },
    });
    await context?.writer?.custom({
      type: "data-sub-agent-progress",
      data: { step: "processing", task, progress: 75 },
    });
    return { completed: true, task };
  },
});

const subAgent = new Agent({
  id: "subAgent",
  name: "Sub Agent",
  instructions:
    'When asked to analyze data, call subAgentTool exactly once with task "analyze data". After the tool result, answer with only SUB_DONE.',
  model: model("openai", "gpt-4o-mini"),
  tools: { subAgentTool },
  maxSteps: 3,
});

const parentAgent = new Agent({
  id: "parentAgent",
  name: "Parent Agent",
  instructions:
    "Delegate the user's request to subAgent. Do not answer directly. After subAgent finishes, answer with only DONE.",
  model: model("openai", "gpt-4o-mini"),
  agents: { subAgent },
  maxSteps: 3,
});

const stream = await parentAgent.stream("Ask subAgent to analyze data.", {
  maxSteps: 3,
  modelSettings: { temperature: 0 },
});
const chunks: any[] = [];
for await (const chunk of stream.fullStream) {
  chunks.push(chunk);
}

const customChunks = chunks.filter((chunk) => chunk.type === "data-sub-agent-progress");
const steps = customChunks.map((chunk) => chunk.data?.step).join(",");

output({
  chunkTypes: chunks.map((chunk) => chunk.type).join(","),
  customDataCount: customChunks.length,
  hasSubAgentCustomData: customChunks.length >= 2,
  hasInitializing: steps.includes("initializing"),
  hasProcessing: steps.includes("processing"),
  steps,
});

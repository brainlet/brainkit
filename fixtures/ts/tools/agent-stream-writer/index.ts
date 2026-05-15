import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

const mixedTool = createTool({
  id: "mixedTool",
  description: "Use this tool when asked to process the exact value alpha.",
  inputSchema: z.object({ value: z.string() }),
  execute: async ({ value }: any, context: any) => {
    await context?.writer?.write({
      type: "status-update",
      message: "Starting processing",
    });
    await context?.writer?.custom({
      type: "data-processing-metrics",
      data: {
        value,
        phase: "middle",
      },
    });
    await context?.writer?.write({
      type: "status-update",
      message: "Processing complete",
    });
    return { processed: value };
  },
});

const agent = new Agent({
  id: "agent-stream-writer",
  name: "Agent Stream Writer",
  instructions:
    'You must call mixedTool exactly once with value "alpha" before answering. After the tool result, answer with only DONE.',
  model: model("openai", "gpt-4o-mini"),
  tools: { mixedTool },
  maxSteps: 3,
});

const stream = await agent.stream('Call mixedTool with value "alpha", then answer DONE.', {
  maxSteps: 3,
  modelSettings: { temperature: 0 },
});
const chunks: any[] = [];
for await (const chunk of stream.fullStream) {
  chunks.push(chunk);
}

const toolOutputChunks = chunks.filter((chunk) => chunk.type === "tool-output");
const customDataChunks = chunks.filter((chunk) => chunk.type === "data-processing-metrics");
const hasRegularWrite = toolOutputChunks.some((chunk) => chunk.payload?.output?.type === "status-update");
const customPayload = customDataChunks[0]?.data;

output({
  chunkTypes: chunks.map((chunk) => chunk.type).join(","),
  toolOutputCount: toolOutputChunks.length,
  customDataCount: customDataChunks.length,
  hasRegularWrite,
  hasCustomData: customDataChunks.length > 0,
  customValue: customPayload?.value ?? "",
  customPhase: customPayload?.phase ?? "",
});

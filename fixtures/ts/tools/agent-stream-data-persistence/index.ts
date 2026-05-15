import { Agent, createTool, InMemoryStore, Memory, z } from "agent";
import { model, output } from "kit";

const memory = new Memory({
  storage: new InMemoryStore(),
  options: { lastMessages: 10 },
});

const progressTool = createTool({
  id: "progressTool",
  description: "Use this tool when asked to report progress for the exact test-task.",
  inputSchema: z.object({ taskName: z.string() }),
  execute: async ({ taskName }: any, context: any) => {
    await context?.writer?.custom({
      type: "data-progress",
      data: {
        taskName,
        progress: 50,
        status: "in-progress",
      },
    });
    await context?.writer?.custom({
      type: "data-progress",
      data: {
        taskName,
        progress: 100,
        status: "complete",
      },
    });
    return { success: true, taskName };
  },
});

const agent = new Agent({
  id: "agent-stream-data-persistence",
  name: "Agent Stream Data Persistence",
  instructions:
    'You must call progressTool exactly once with taskName "test-task". After the tool result, answer with only DONE.',
  model: model("openai", "gpt-4o-mini"),
  tools: { progressTool },
  memory,
  maxSteps: 3,
});

const threadId = "agent-stream-data-persistence-thread";
const resourceId = "agent-stream-data-persistence-user";

const stream = await agent.stream('Run progressTool for taskName "test-task", then answer DONE.', {
  maxSteps: 3,
  modelSettings: { temperature: 0 },
  memory: {
    thread: { id: threadId },
    resource: resourceId,
  },
});

const chunks: any[] = [];
for await (const chunk of stream.fullStream) {
  chunks.push(chunk);
}

async function sleep(ms: number) {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

function dataParts(messages: any[]) {
  return messages.flatMap((message) => {
    const content = message.content;
    if (content && typeof content === "object" && Array.isArray(content.parts)) {
      return content.parts.filter((part: any) => typeof part.type === "string" && part.type.startsWith("data-"));
    }
    return [];
  });
}

let recalledMessages: any[] = [];
let recalledDataParts: any[] = [];
for (let i = 0; i < 30; i++) {
  const recalled = await memory.recall({ threadId, resourceId });
  recalledMessages = recalled.messages;
  recalledDataParts = dataParts(recalledMessages);
  if (recalledDataParts.some((part) => part.type === "data-progress")) {
    break;
  }
  await sleep(250);
}

const streamedDataChunks = chunks.filter((chunk) => chunk.type === "data-progress");
const persistedProgressParts = recalledDataParts.filter((part) => part.type === "data-progress");
const persistedStatuses = persistedProgressParts.map((part) => part.data?.status).join(",");

output({
  chunkTypes: chunks.map((chunk) => chunk.type).join(","),
  streamedDataCount: streamedDataChunks.length,
  persistedDataCount: persistedProgressParts.length,
  hasStreamedProgress: streamedDataChunks.length >= 2,
  hasPersistedProgress: persistedProgressParts.length >= 2,
  persistedStatuses,
  assistantMessageCount: recalledMessages.filter((message) => message.role === "assistant").length,
});

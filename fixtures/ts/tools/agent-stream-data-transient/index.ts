import { Agent, createTool, InMemoryStore, Memory, z } from "agent";
import { model, output } from "kit";

const memory = new Memory({
  storage: new InMemoryStore(),
  options: { lastMessages: 10 },
});

const sandboxTool = createTool({
  id: "sandboxTool",
  description: "Use this tool when asked to run the sandbox task.",
  inputSchema: z.object({ taskName: z.string() }),
  execute: async ({ taskName }: any, context: any) => {
    await context?.writer?.custom({
      type: "data-sandbox-stdout",
      data: { output: "streaming output line 1\n", taskName },
      transient: true,
    });
    await context?.writer?.custom({
      type: "data-sandbox-stderr",
      data: { output: "error output\n", taskName },
      transient: true,
    });
    await context?.writer?.custom({
      type: "data-sandbox-exit",
      data: {
        exitCode: 0,
        success: true,
        executionTimeMs: 123,
        taskName,
      },
    });
    return { success: true, taskName };
  },
});

const agent = new Agent({
  id: "agent-stream-data-transient",
  name: "Agent Stream Data Transient",
  instructions:
    'You must call sandboxTool exactly once with taskName "test-task". After the tool result, answer with only DONE.',
  model: model("openai", "gpt-4o-mini"),
  tools: { sandboxTool },
  memory,
  maxSteps: 3,
});

const threadId = "agent-stream-data-transient-thread";
const resourceId = "agent-stream-data-transient-user";

const stream = await agent.stream('Run sandboxTool for taskName "test-task", then answer DONE.', {
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
  if (recalledDataParts.some((part) => part.type === "data-sandbox-exit")) {
    break;
  }
  await sleep(250);
}

const stdoutChunks = chunks.filter((chunk) => chunk.type === "data-sandbox-stdout");
const stderrChunks = chunks.filter((chunk) => chunk.type === "data-sandbox-stderr");
const exitChunks = chunks.filter((chunk) => chunk.type === "data-sandbox-exit");
const persistedStdoutParts = recalledDataParts.filter((part) => part.type === "data-sandbox-stdout");
const persistedStderrParts = recalledDataParts.filter((part) => part.type === "data-sandbox-stderr");
const persistedExitParts = recalledDataParts.filter((part) => part.type === "data-sandbox-exit");

output({
  chunkTypes: chunks.map((chunk) => chunk.type).join(","),
  streamedStdoutCount: stdoutChunks.length,
  streamedStderrCount: stderrChunks.length,
  streamedExitCount: exitChunks.length,
  persistedStdoutCount: persistedStdoutParts.length,
  persistedStderrCount: persistedStderrParts.length,
  persistedExitCount: persistedExitParts.length,
  streamsTransientChunks: stdoutChunks.length === 1 && stderrChunks.length === 1,
  persistsNonTransientExit: persistedExitParts.length === 1,
  omitsTransientFromMemory: persistedStdoutParts.length === 0 && persistedStderrParts.length === 0,
  assistantMessageCount: recalledMessages.filter((message) => message.role === "assistant").length,
});

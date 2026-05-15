import { Agent, createTool, InMemoryStore, Mastra, Memory, z } from "agent";
import { model, output } from "kit";

async function sleep(ms: number) {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

async function withTimeout<T>(label: string, ms: number, fn: () => Promise<T>): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      fn(),
      new Promise<T>((_, reject) => {
        timer = setTimeout(() => reject(new Error(`timeout:${label}`)), ms);
      }),
    ]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const researchTool = createTool({
  id: "research",
  description: 'Research a topic. Use this tool whenever the user asks to research the exact topic "quantum computing".',
  inputSchema: z.object({
    topic: z.string().describe("The topic to research"),
  }),
  outputSchema: z.object({
    summary: z.string(),
  }),
  background: {
    enabled: true,
    timeoutMs: 30_000,
  },
  execute: async ({ topic }: any) => {
    await sleep(250);
    return {
      summary: `Research complete on "${topic}": This is a comprehensive summary.`,
    };
  },
});

const agent = new Agent({
  id: "stream-until-idle-agent",
  name: "Stream Until Idle Agent",
  instructions:
    'You must call the research tool when the user asks to research "quantum computing". ' +
    "After the background task result is available, briefly mention the completed topic.",
  model: model("openai", "gpt-4o-mini"),
  tools: { research: researchTool },
  memory,
  backgroundTasks: {
    tools: {
      research: true,
    },
  },
  maxSteps: 3,
});

const mastra = new Mastra({
  agents: { "stream-until-idle-agent": agent },
  storage,
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 5,
    perAgentConcurrency: 3,
    defaultTimeoutMs: 30_000,
  },
});

let workersStarted = false;

await withTimeout("startWorkers", 10_000, async () => {
  await mastra.startWorkers("backgroundTasks");
  workersStarted = true;
});

const threadId = "stream-until-idle-thread";
const resourceId = "stream-until-idle-user";

try {
  const liveAgent = mastra.getAgent("stream-until-idle-agent");
  const result = await withTimeout("streamUntilIdle", 120_000, () =>
    liveAgent.streamUntilIdle('Please research "quantum computing" for me.', {
      memory: {
        thread: { id: threadId },
        resource: resourceId,
      },
      maxSteps: 3,
      maxIdleMs: 30_000,
      modelSettings: { temperature: 0 },
    }),
  );

  const chunks: any[] = [];
  await withTimeout("fullStream", 120_000, async () => {
    for await (const chunk of result.fullStream) {
      chunks.push(chunk);
    }
  });

  const bgStarted = chunks.find((chunk) => chunk.type === "background-task-started");
  const bgRunning = chunks.find((chunk) => chunk.type === "background-task-running");
  const bgCompleted = chunks.find((chunk) => chunk.type === "background-task-completed");
  const finishes = chunks.filter((chunk) => chunk.type === "finish");
  const text = chunks
    .filter((chunk) => chunk.type === "text-delta")
    .map((chunk) => chunk.payload?.text ?? chunk.delta ?? "")
    .join("")
    .toLowerCase();

  const manager = mastra.backgroundTaskManager;
  const completedTasks = (await manager?.listTasks({ toolName: "research", status: "completed" })) ?? {
    tasks: [],
    total: 0,
  };
  const taskResult = (completedTasks.tasks[0]?.result as any)?.summary ?? "";

  output({
    chunkTypes: chunks.map((chunk) => chunk.type).join(","),
    hasBackgroundStarted: !!bgStarted,
    hasBackgroundRunning: !!bgRunning,
    hasBackgroundCompleted: !!bgCompleted,
    hasMatchingTaskId: !!bgStarted && !!bgCompleted && bgStarted.payload?.taskId === bgCompleted.payload?.taskId,
    hasContinuationFinish: finishes.length >= 2,
    finishCount: finishes.length,
    completedTaskCount: completedTasks.total,
    taskResultIncludesTopic: taskResult.toLowerCase().includes("quantum computing"),
    textIncludesTopic: text.includes("quantum computing"),
  });
} finally {
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

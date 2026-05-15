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

async function waitForTaskStatus(manager: any, taskId: string, status: string): Promise<any> {
  return withTimeout(`task:${status}`, 30_000, async () => {
    while (true) {
      const task = await manager.getTask(taskId);
      if (task?.status === status) return task;
      await sleep(50);
    }
  });
}

const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

let attemptCount = 0;

const retryTool = createTool({
  id: "retry-research",
  description:
    'Run research that fails once with a transient error and then succeeds. Use this tool whenever the user asks for retry research on "venus".',
  inputSchema: z.object({
    topic: z.string().describe("The retry research topic"),
  }),
  outputSchema: z.object({
    summary: z.string(),
    attempts: z.number(),
  }),
  background: {
    enabled: true,
    timeoutMs: 30_000,
    maxRetries: 1,
  },
  execute: async ({ topic }: any) => {
    attemptCount += 1;
    if (attemptCount === 1) {
      throw new Error(`transient retry error for ${topic}`);
    }
    return {
      summary: `Retry research complete on "${topic}".`,
      attempts: attemptCount,
    };
  },
});

const agent = new Agent({
  id: "background-retry-agent",
  name: "Background Retry Agent",
  instructions:
    'You must call the retry-research tool when the user asks for retry research on "venus". ' +
    'When you call it, include "_background": { "enabled": true, "maxRetries": 1, "timeoutMs": 30000 } in the tool arguments. ' +
    "After the background task result is available, briefly mention venus.",
  model: model("openai", "gpt-4o-mini"),
  tools: { retryResearch: retryTool },
  memory,
  backgroundTasks: {
    tools: {
      retryResearch: { enabled: true, timeoutMs: 30_000 },
    },
  },
  maxSteps: 3,
});

const mastra = new Mastra({
  agents: { "background-retry-agent": agent },
  storage,
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 5,
    perAgentConcurrency: 3,
    defaultTimeoutMs: 30_000,
    defaultRetries: { maxRetries: 1 },
  },
});

let workersStarted = false;

await withTimeout("startWorkers", 10_000, async () => {
  await mastra.startWorkers("backgroundTasks");
  workersStarted = true;
});

try {
  const liveAgent = mastra.getAgent("background-retry-agent");
  const result = await withTimeout("streamUntilIdle", 120_000, () =>
    liveAgent.streamUntilIdle('Please run retry research on "venus".', {
      memory: {
        thread: { id: "background-retry-thread" },
        resource: "background-retry-user",
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
  const bgCompleted = chunks.find((chunk) => chunk.type === "background-task-completed");
  const taskId = bgStarted?.payload?.taskId ?? bgCompleted?.payload?.taskId;
  const completedTask = taskId ? await waitForTaskStatus(mastra.backgroundTaskManager, taskId, "completed") : undefined;
  const completedTasks = (await mastra.backgroundTaskManager?.listTasks({
    toolName: "retryResearch",
    status: "completed",
  })) ?? { tasks: [], total: 0 };
  const resultSummary = String((completedTask?.result as any)?.summary ?? "");
  const resultAttempts = Number((completedTask?.result as any)?.attempts ?? 0);
  const text = chunks
    .filter((chunk) => chunk.type === "text-delta")
    .map((chunk) => chunk.payload?.text ?? chunk.delta ?? "")
    .join("")
    .toLowerCase();

  output({
    chunkTypes: chunks.map((chunk) => chunk.type).join(","),
    hasBackgroundStarted: !!bgStarted,
    hasBackgroundCompleted: !!bgCompleted,
    hasNoBackgroundFailed: !chunks.some((chunk) => chunk.type === "background-task-failed"),
    hasMatchingTaskId: !!bgStarted && !!bgCompleted && bgStarted.payload?.taskId === bgCompleted.payload?.taskId,
    completedStatus: completedTask?.status ?? "",
    retryCount: completedTask?.retryCount ?? -1,
    attemptCount,
    resultAttempts,
    completedTaskCount: completedTasks.total,
    resultIncludesTopic: resultSummary.toLowerCase().includes("venus"),
    textIncludesTopic: text.includes("venus"),
  });
} finally {
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

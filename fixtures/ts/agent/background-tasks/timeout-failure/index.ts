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

async function waitForAbort(signal: AbortSignal | undefined, fallbackMs: number): Promise<void> {
  if (!signal) throw new Error("missing background abortSignal");
  if (signal.aborted) throw signal.reason ?? Object.assign(new Error("Aborted"), { name: "AbortError" });

  await new Promise<void>((_, reject) => {
    const fallback = setTimeout(() => reject(new Error("background abortSignal did not abort")), fallbackMs);
    const onAbort = () => {
      clearTimeout(fallback);
      signal.removeEventListener("abort", onAbort);
      reject(signal.reason ?? Object.assign(new Error("Aborted"), { name: "AbortError" }));
    };
    signal.addEventListener("abort", onAbort);
  });
}

const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const timeoutTool = createTool({
  id: "timeout-research",
  description:
    'Run a deliberately slow research job. Use this tool whenever the user asks for timeout research on "neptune".',
  inputSchema: z.object({
    topic: z.string().describe("The timeout research topic"),
  }),
  outputSchema: z.object({
    summary: z.string(),
  }),
  background: {
    enabled: true,
    timeoutMs: 300,
    maxRetries: 0,
  },
  execute: async ({ topic }: any, context: any) => {
    await waitForAbort(context?.abortSignal ?? context?.agent?.abortSignal, 10_000);
    return {
      summary: `Unexpected timeout research completion for "${topic}".`,
    };
  },
});

const agent = new Agent({
  id: "background-timeout-agent",
  name: "Background Timeout Agent",
  instructions:
    'You must call the timeout-research tool when the user asks for timeout research on "neptune". ' +
    'When you call it, include "_background": { "enabled": true, "timeoutMs": 300, "maxRetries": 0 } in the tool arguments. ' +
    "Do not answer from memory; the tool call is required.",
  model: model("openai", "gpt-4o-mini"),
  tools: { timeoutResearch: timeoutTool },
  memory,
  backgroundTasks: {
    tools: {
      timeoutResearch: { enabled: true, timeoutMs: 300 },
    },
  },
  maxSteps: 3,
});

const mastra = new Mastra({
  agents: { "background-timeout-agent": agent },
  storage,
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 5,
    perAgentConcurrency: 3,
    defaultTimeoutMs: 300,
  },
});

let workersStarted = false;

await withTimeout("startWorkers", 10_000, async () => {
  await mastra.startWorkers("backgroundTasks");
  workersStarted = true;
});

try {
  const liveAgent = mastra.getAgent("background-timeout-agent");
  const result = await withTimeout("streamUntilIdle", 120_000, () =>
    liveAgent.streamUntilIdle(
      'Please run timeout research on "neptune" using a 300 ms background timeout override.',
      {
        memory: {
          thread: { id: "background-timeout-thread" },
          resource: "background-timeout-user",
        },
        maxSteps: 1,
        maxIdleMs: 5_000,
        modelSettings: { temperature: 0 },
      },
    ),
  );

  const chunks: any[] = [];
  await withTimeout("fullStream", 120_000, async () => {
    for await (const chunk of result.fullStream) {
      chunks.push(chunk);
    }
  });

  const bgStarted = chunks.find((chunk) => chunk.type === "background-task-started");
  const bgRunning = chunks.find((chunk) => chunk.type === "background-task-running");
  const bgFailed = chunks.find((chunk) => chunk.type === "background-task-failed");
  const taskId = bgStarted?.payload?.taskId ?? bgFailed?.payload?.taskId;
  const observedTask = taskId ? await waitForTaskStatus(mastra.backgroundTaskManager, taskId, "timed_out") : undefined;
  const timedOutTasks = (await mastra.backgroundTaskManager?.listTasks({
    toolName: "timeoutResearch",
    status: "timed_out",
  })) ?? { tasks: [], total: 0 };
  const allTasks = (await mastra.backgroundTaskManager?.listTasks({})) ?? { tasks: [], total: 0 };
  const errorMessage = String(bgFailed?.payload?.error?.message ?? observedTask?.error?.message ?? "");

  output({
    chunkTypes: chunks.map((chunk) => chunk.type).join(","),
    allTaskStatuses: allTasks.tasks.map((task: any) => `${task.toolName}:${task.status}`).join(","),
    allTaskTimeouts: allTasks.tasks.map((task: any) => `${task.toolName}:${task.timeoutMs}`).join(","),
    observedError: errorMessage,
    hasBackgroundStarted: !!bgStarted,
    hasBackgroundRunning: !!bgRunning,
    hasBackgroundFailed: !!bgFailed,
    hasNoBackgroundCompleted: !chunks.some((chunk) => chunk.type === "background-task-completed"),
    hasMatchingTaskId: !!bgStarted && !!bgFailed && bgStarted.payload?.taskId === bgFailed.payload?.taskId,
    failedErrorIncludesTimeout: errorMessage.toLowerCase().includes("timed out"),
    timedOutStatus: observedTask?.status ?? "",
    timedOutTaskCount: timedOutTasks.total,
  });
} finally {
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

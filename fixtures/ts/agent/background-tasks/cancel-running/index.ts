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

async function waitForFlag(label: string, read: () => boolean): Promise<boolean> {
  return withTimeout(label, 30_000, async () => {
    while (!read()) {
      await sleep(25);
    }
    return true;
  });
}

async function readUntilEvent(reader: any, type: string): Promise<any> {
  return withTimeout(`event:${type}`, 30_000, async () => {
    while (true) {
      const { done, value } = await reader.read();
      if (done) return undefined;
      if (value?.type === type) return value;
    }
  });
}

let toolEntered = false;
let toolAbortObserved = false;

async function waitForCancellation(signal: AbortSignal | undefined): Promise<void> {
  if (!signal) throw new Error("missing background abortSignal");
  if (signal.aborted) {
    toolAbortObserved = true;
    throw signal.reason ?? new Error("Task cancelled");
  }

  await new Promise<void>((_, reject) => {
    const fallback = setTimeout(() => reject(new Error("background cancel signal did not abort")), 60_000);
    const onAbort = () => {
      toolAbortObserved = true;
      clearTimeout(fallback);
      signal.removeEventListener("abort", onAbort);
      reject(signal.reason ?? new Error("Task cancelled"));
    };
    signal.addEventListener("abort", onAbort);
  });
}

const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const cancelTool = createTool({
  id: "cancel-research",
  description:
    'Run a long research job that will be cancelled. Use this tool whenever the user asks for cancel research on "saturn".',
  inputSchema: z.object({
    topic: z.string().describe("The cancel research topic"),
  }),
  outputSchema: z.object({
    summary: z.string(),
  }),
  background: {
    enabled: true,
    timeoutMs: 30_000,
    maxRetries: 0,
  },
  execute: async ({ topic }: any, context: any) => {
    toolEntered = true;
    await waitForCancellation(context?.abortSignal ?? context?.agent?.abortSignal);
    return {
      summary: `Unexpected cancel research completion for "${topic}".`,
    };
  },
});

const agent = new Agent({
  id: "background-cancel-agent",
  name: "Background Cancel Agent",
  instructions:
    'You must call the cancel-research tool when the user asks for cancel research on "saturn". ' +
    "Do not answer from memory; the tool call is required.",
  model: model("openai", "gpt-4o-mini"),
  tools: { cancelResearch: cancelTool },
  memory,
  backgroundTasks: {
    tools: {
      cancelResearch: true,
    },
  },
  maxSteps: 2,
});

const mastra = new Mastra({
  agents: { "background-cancel-agent": agent },
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

try {
  const liveAgent = mastra.getAgent("background-cancel-agent");
  const result = await withTimeout("stream", 120_000, () =>
    liveAgent.stream('Please run cancel research on "saturn".', {
      memory: {
        thread: { id: "background-cancel-thread" },
        resource: "background-cancel-user",
      },
      maxSteps: 2,
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
  const taskId = bgStarted?.payload?.taskId;

  let startedTask: any;
  let cancelledTask: any;
  let cancelEvent: any;
  let cancelledTasks = { tasks: [], total: 0 } as any;

  if (taskId) {
    await waitForFlag("tool-entered", () => toolEntered);
    startedTask = await waitForTaskStatus(mastra.backgroundTaskManager, taskId, "running");

    const abortController = new AbortController();
    const managerStream = mastra.backgroundTaskManager.stream({
      taskId,
      abortSignal: abortController.signal,
    });
    const reader = managerStream.getReader();

    try {
      const cancelEventPromise = readUntilEvent(reader, "background-task-cancelled");
      await mastra.backgroundTaskManager.cancel(taskId);
      cancelEvent = await cancelEventPromise;
      cancelledTask = await waitForTaskStatus(mastra.backgroundTaskManager, taskId, "cancelled");
      await waitForFlag("tool-abort-observed", () => toolAbortObserved);
    } finally {
      abortController.abort();
    }

    cancelledTasks = await mastra.backgroundTaskManager.listTasks({
      toolName: "cancelResearch",
      status: "cancelled",
    });
  }

  output({
    chunkTypes: chunks.map((chunk) => chunk.type).join(","),
    hasBackgroundStarted: !!bgStarted,
    hasNoBackgroundCompleted: !chunks.some((chunk) => chunk.type === "background-task-completed"),
    startedTaskStatus: startedTask?.status ?? "",
    hasCancelEvent: cancelEvent?.type === "background-task-cancelled",
    cancelEventTaskMatches: !!taskId && cancelEvent?.payload?.taskId === taskId,
    cancelledStatus: cancelledTask?.status ?? "",
    cancelledTaskCount: cancelledTasks.total,
    toolAbortObserved,
  });
} finally {
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

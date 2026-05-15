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

function progressLabel(chunk: any): string {
  const payload = chunk?.payload?.payload ?? chunk?.payload ?? {};
  const nested = payload?.payload ?? payload?.data ?? payload;
  return String(nested?.label ?? nested?.output?.label ?? nested?.output ?? "");
}

const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const progressTool = createTool({
  id: "progress-report",
  description: 'Create a progress report. Use this tool whenever the user asks for the exact report "alpha".',
  inputSchema: z.object({
    report: z.string().describe("The report name"),
  }),
  outputSchema: z.object({
    summary: z.string(),
  }),
  background: {
    enabled: true,
    timeoutMs: 30_000,
  },
  execute: async ({ report }: any, context: any) => {
    await context?.writer?.write({
      type: "progress",
      label: "queued",
      report,
    });
    await sleep(50);
    await context?.writer?.write({
      type: "progress",
      label: "halfway",
      report,
    });
    await sleep(50);
    return {
      summary: `Progress report "${report}" finished.`,
    };
  },
});

const agent = new Agent({
  id: "background-progress-agent",
  name: "Background Progress Agent",
  instructions:
    'You must call the progress-report tool when the user asks for report "alpha". ' +
    "After the background task result is available, briefly mention that alpha finished.",
  model: model("openai", "gpt-4o-mini"),
  tools: { progressReport: progressTool },
  memory,
  backgroundTasks: {
    tools: {
      progressReport: true,
    },
  },
  maxSteps: 3,
});

const mastra = new Mastra({
  agents: { "background-progress-agent": agent },
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
  const liveAgent = mastra.getAgent("background-progress-agent");
  const result = await withTimeout("streamUntilIdle", 120_000, () =>
    liveAgent.streamUntilIdle('Please create report "alpha".', {
      memory: {
        thread: { id: "background-progress-thread" },
        resource: "background-progress-user",
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
  const bgOutputs = chunks.filter((chunk) => chunk.type === "background-task-output");
  const labels = bgOutputs.map(progressLabel).filter(Boolean);
  const completedTasks = (await mastra.backgroundTaskManager?.listTasks({
    toolName: "progressReport",
    status: "completed",
  })) ?? { tasks: [], total: 0 };
  const taskResult = (completedTasks.tasks[0]?.result as any)?.summary ?? "";

  output({
    chunkTypes: chunks.map((chunk) => chunk.type).join(","),
    outputLabels: labels.join(","),
    hasBackgroundStarted: !!bgStarted,
    hasBackgroundCompleted: !!bgCompleted,
    hasMatchingTaskId: !!bgStarted && !!bgCompleted && bgStarted.payload?.taskId === bgCompleted.payload?.taskId,
    backgroundOutputCount: bgOutputs.length,
    hasQueuedOutput: labels.includes("queued"),
    hasHalfwayOutput: labels.includes("halfway"),
    completedTaskCount: completedTasks.total,
    taskResultIncludesReport: taskResult.toLowerCase().includes("alpha"),
  });
} finally {
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

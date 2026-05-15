import { createBackgroundTask, InMemoryStore, Mastra } from "agent";
import { output } from "kit";

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
      await sleep(25);
    }
  });
}

const storage = new InMemoryStore();
const mastra = new Mastra({
  storage,
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 2,
    perAgentConcurrency: 2,
    defaultTimeoutMs: 30_000,
  },
});

const workerNames = Array.from((mastra as any).workers ?? []).map((worker: any) => worker.name).join(",");

let workersStarted = false;

await withTimeout("startWorkers-all", 10_000, async () => {
  await mastra.startWorkers();
  workersStarted = true;
});

const manager = mastra.backgroundTaskManager;
if (!manager) throw new Error("missing backgroundTaskManager");

try {
  const task = createBackgroundTask(manager, {
    toolName: "all-workers-task",
    toolCallId: "call-all-workers",
    args: { check: "all-workers" },
    agentId: "all-workers-agent",
    runId: "all-workers-run",
    context: {
      executor: {
        execute: async ({ check }: any) => {
          await sleep(50);
          return { ok: true, check };
        },
      },
    },
  });

  await task.dispatch();
  const completedTask = await waitForTaskStatus(manager, task.task.id, "completed");

  output({
    workerNames,
    workersStarted,
    hasOrchestrationWorker: workerNames.includes("orchestration"),
    hasBackgroundTaskWorker: workerNames.includes("backgroundTasks"),
    completedStatus: completedTask?.status ?? "",
    completedCheck: String(completedTask?.result?.check ?? ""),
  });
} finally {
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

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

async function waitForCallbackCount(read: () => number, expected: number): Promise<void> {
  await withTimeout(`callback:${expected}`, 30_000, async () => {
    while (read() !== expected) {
      await sleep(25);
    }
  });
}

const events: string[] = [];
const managerCompleteStatuses: string[] = [];
const managerFailedStatuses: string[] = [];

const storage = new InMemoryStore();
const mastra = new Mastra({
  storage,
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 4,
    perAgentConcurrency: 4,
    defaultTimeoutMs: 30_000,
    onTaskComplete: async (task: any) => {
      managerCompleteStatuses.push(task.status);
      events.push(`manager-complete:${task.toolName}:${task.status}`);
    },
    onTaskFailed: async (task: any) => {
      managerFailedStatuses.push(task.status);
      events.push(`manager-failed:${task.toolName}:${task.status}`);
    },
  },
});

let workersStarted = false;

await withTimeout("startWorkers", 10_000, async () => {
  await mastra.startWorkers("backgroundTasks");
  workersStarted = true;
});

const manager = mastra.backgroundTaskManager;
if (!manager) throw new Error("missing backgroundTaskManager");

function makeTask(label: "complete" | "failed") {
  return createBackgroundTask(manager, {
    toolName: `callback-${label}`,
    toolCallId: `call-${label}`,
    args: { label },
    agentId: "callback-agent",
    runId: "callback-run",
    context: {
      executor: {
        execute: async () => {
          events.push(`execute:${label}`);
          if (label === "failed") {
            throw new Error("planned callback failure");
          }
          return { label, ok: true };
        },
      },
      onExecution: async (task: any) => {
        events.push(`execution:${label}:${task.toolName}`);
      },
      onChunk: (chunk: any) => {
        events.push(`chunk:${label}:${chunk.type}`);
      },
      onResult: async (result: any) => {
        events.push(`result:${label}:${result.status}`);
      },
      onComplete: async (task: any) => {
        events.push(`task-complete:${label}:${task.status}`);
      },
      onFailed: async (task: any) => {
        events.push(`task-failed:${label}:${task.status}`);
      },
    },
  });
}

try {
  const completeHandle = makeTask("complete");
  const failedHandle = makeTask("failed");

  await completeHandle.dispatch();
  await failedHandle.dispatch();

  const completedTask = await waitForTaskStatus(manager, completeHandle.task.id, "completed");
  const failedTask = await waitForTaskStatus(manager, failedHandle.task.id, "failed");
  await waitForCallbackCount(() => managerCompleteStatuses.length, 1);
  await waitForCallbackCount(() => managerFailedStatuses.length, 1);

  output({
    events: events.join(","),
    completedStatus: completedTask?.status ?? "",
    failedStatus: failedTask?.status ?? "",
    completedResultLabel: String(completedTask?.result?.label ?? ""),
    failedErrorIncludesPlanned: String(failedTask?.error?.message ?? "").includes("planned callback failure"),
    managerCompleteCount: managerCompleteStatuses.length,
    managerFailedCount: managerFailedStatuses.length,
    perTaskExecutionCount: events.filter((event) => event.startsWith("execution:")).length,
    hasCompleteChunk: events.includes("chunk:complete:background-task-completed"),
    hasFailedChunk: events.includes("chunk:failed:background-task-failed"),
    hasCompleteResult: events.includes("result:complete:completed"),
    hasFailedResult: events.includes("result:failed:failed"),
    hasPerTaskComplete: events.includes("task-complete:complete:completed"),
    hasPerTaskFailed: events.includes("task-failed:failed:failed"),
  });
} finally {
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

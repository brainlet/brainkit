import { __brainkitMastraBackgroundTaskDebug, createBackgroundTask, InMemoryStore, Mastra } from "agent";
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

function deferred() {
  let resolve!: () => void;
  const promise = new Promise<void>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

function debug() {
  const snapshot = __brainkitMastraBackgroundTaskDebug?.();
  if (!snapshot) throw new Error("missing Brainkit background task debug hook");
  return snapshot;
}

async function readUntil(reader: any, predicate: (chunk: any) => boolean, label: string): Promise<any> {
  return withTimeout(label, 30_000, async () => {
    while (true) {
      const { value, done } = await reader.read();
      if (done) throw new Error(`stream closed before ${label}`);
      if (predicate(value)) return value;
    }
  });
}

async function waitForStatus(manager: any, taskId: string, status: string): Promise<any> {
  return withTimeout(`task:${status}`, 30_000, async () => {
    while (true) {
      const task = await manager.getTask(taskId);
      if (task?.status === status) return task;
      await sleep(25);
    }
  });
}

async function readUntilClosed(reader: any, label: string): Promise<boolean> {
  return withTimeout(label, 5_000, async () => {
    while (true) {
      const { done } = await reader.read();
      if (done) return true;
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

let workersStarted = false;
const gate = deferred();

await withTimeout("startWorkers", 10_000, async () => {
  await mastra.startWorkers();
  workersStarted = true;
});

const manager = mastra.backgroundTaskManager;
if (!manager) throw new Error("missing backgroundTaskManager");

try {
  const backgroundTask = createBackgroundTask(manager, {
    toolName: "stream-worker-teardown-task",
    toolCallId: "call-stream-worker-teardown",
    args: { value: "teardown" },
    agentId: "stream-worker-teardown-agent",
    runId: "stream-worker-teardown-run",
    context: {
      executor: {
        execute: async ({ value }: any) => {
          await gate.promise;
          return { ok: true, value };
        },
      },
    },
  });

  await backgroundTask.dispatch();
  await waitForStatus(manager, backgroundTask.task.id, "running");

  const callerAbort = new AbortController();
  const callerReader = manager
    .stream({
      taskId: backgroundTask.task.id,
      abortSignal: callerAbort.signal,
    })
    .getReader();
  const callerSnapshot = await readUntil(
    callerReader,
    (chunk) => chunk.type === "background-task-running",
    "caller-abort-snapshot",
  );
  const activeAfterCallerOpen = debug().activeStreams;
  callerAbort.abort();
  const callerClosed = await withTimeout("caller-abort-close", 5_000, async () => {
    const { done } = await callerReader.read();
    return done;
  });
  const debugAfterCallerAbort = debug();

  const shutdownReader = manager
    .stream({
      taskId: backgroundTask.task.id,
    })
    .getReader();
  const shutdownSnapshot = await readUntil(
    shutdownReader,
    (chunk) => chunk.type === "background-task-running",
    "shutdown-snapshot",
  );
  const activeAfterShutdownOpen = debug().activeStreams;

  gate.resolve();
  await waitForStatus(manager, backgroundTask.task.id, "completed");

  await withTimeout("shutdown", 10_000, async () => {
    await mastra.backgroundTaskManager?.shutdown();
    await mastra.stopWorkers();
    workersStarted = false;
  });

  const shutdownClosed = await readUntilClosed(shutdownReader, "shutdown-stream-close");
  const finalDebug = debug();

  output({
    workerStarted: activeAfterCallerOpen >= 1,
    callerSnapshotMatches: callerSnapshot?.payload?.taskId === backgroundTask.task.id,
    callerStreamClosedOnAbort: callerClosed,
    callerAbortDrainedStreams: debugAfterCallerAbort.activeStreams === 0,
    shutdownSnapshotMatches: shutdownSnapshot?.payload?.taskId === backgroundTask.task.id,
    activeStreamObservedBeforeShutdown: activeAfterShutdownOpen >= 1,
    shutdownStreamClosed: shutdownClosed,
    finalActiveStreams: finalDebug.activeStreams,
    finalTrackedStreams: finalDebug.activeTrackedStreams,
    finalTaskContexts: finalDebug.activeTaskContexts,
    finalAbortControllers: finalDebug.activeAbortControllers,
    finalActiveWorkers: finalDebug.activeWorkerCount,
    shutdownCalls: finalDebug.shutdownCalls,
    workerStopCalls: finalDebug.workerStopCalls,
  });
} finally {
  gate.resolve();
  if (workersStarted) {
    await withTimeout("cleanup-shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

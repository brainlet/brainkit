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

async function waitForCounts(
  manager: any,
  expected: { running?: number; pending?: number; completed?: number },
): Promise<{ running: any[]; pending: any[]; completed: any[] }> {
  return withTimeout(`counts:${JSON.stringify(expected)}`, 30_000, async () => {
    while (true) {
      const running = (await manager.listTasks({ status: "running" })).tasks;
      const pending = (await manager.listTasks({ status: "pending" })).tasks;
      const completed = (await manager.listTasks({ status: "completed" })).tasks;
      const ok =
        (expected.running === undefined || running.length === expected.running) &&
        (expected.pending === undefined || pending.length === expected.pending) &&
        (expected.completed === undefined || completed.length === expected.completed);
      if (ok) return { running, pending, completed };
      await sleep(25);
    }
  });
}

async function waitForStarted(started: string[], label: string): Promise<void> {
  await withTimeout(`started:${label}`, 30_000, async () => {
    while (!started.includes(label)) {
      await sleep(25);
    }
  });
}

function deferred(label: string) {
  let resolve!: () => void;
  const promise = new Promise<void>((res) => {
    resolve = res;
  });
  return { label, promise, resolve };
}

const storage = new InMemoryStore();
const mastra = new Mastra({
  storage,
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 2,
    perAgentConcurrency: 1,
    backpressure: "queue",
    defaultTimeoutMs: 30_000,
  },
});

let workersStarted = false;

await withTimeout("startWorkers", 10_000, async () => {
  await mastra.startWorkers("backgroundTasks");
  workersStarted = true;
});

const manager = mastra.backgroundTaskManager;
if (!manager) throw new Error("missing backgroundTaskManager");

const started: string[] = [];
const releases = new Map<string, ReturnType<typeof deferred>>();

function makeTask(label: string, agentId: string) {
  const gate = deferred(label);
  releases.set(label, gate);

  return createBackgroundTask(manager, {
    toolName: "concurrency-research",
    toolCallId: `call-${label}`,
    args: { label },
    agentId,
    runId: "concurrency-run",
    context: {
      executor: {
        execute: async (_args: any, opts: { abortSignal?: AbortSignal }) => {
          started.push(label);
          if (opts.abortSignal?.aborted) throw opts.abortSignal.reason ?? new Error("Task cancelled");
          await new Promise<void>((resolve, reject) => {
            const onAbort = () => {
              opts.abortSignal?.removeEventListener("abort", onAbort);
              reject(opts.abortSignal?.reason ?? new Error("Task cancelled"));
            };
            opts.abortSignal?.addEventListener("abort", onAbort);
            gate.promise.then(() => {
              opts.abortSignal?.removeEventListener("abort", onAbort);
              resolve();
            });
          });
          return { label };
        },
      },
    },
  });
}

try {
  const first = makeTask("a1-first", "agent-a");
  const second = makeTask("a1-second", "agent-a");
  const third = makeTask("a2-first", "agent-b");

  await first.dispatch();
  await second.dispatch();
  await third.dispatch();

  await waitForStarted(started, "a1-first");
  await waitForStarted(started, "a2-first");

  const beforeRelease = await waitForCounts(manager, { running: 2, pending: 1 });
  const secondStayedPendingBeforeRelease = !started.includes("a1-second");

  releases.get("a1-first")?.resolve();
  await waitForStarted(started, "a1-second");
  const afterFirstRelease = await waitForCounts(manager, { running: 2, pending: 0, completed: 1 });

  releases.get("a1-second")?.resolve();
  releases.get("a2-first")?.resolve();
  const finalCounts = await waitForCounts(manager, { running: 0, pending: 0, completed: 3 });

  output({
    startOrder: started.join(","),
    initialRunningCount: beforeRelease.running.length,
    initialPendingCount: beforeRelease.pending.length,
    pendingTaskWasSecondAgentATask: beforeRelease.pending[0]?.toolCallId === "call-a1-second",
    secondStayedPendingBeforeRelease,
    secondStartedAfterRelease: started.includes("a1-second"),
    afterReleaseRunningCount: afterFirstRelease.running.length,
    afterReleasePendingCount: afterFirstRelease.pending.length,
    completedAfterFirstRelease: afterFirstRelease.completed.length,
    finalCompletedCount: finalCounts.completed.length,
    finalLabels: finalCounts.completed.map((task) => task.result?.label).sort().join(","),
  });
} finally {
  for (const gate of releases.values()) {
    gate.resolve();
  }
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

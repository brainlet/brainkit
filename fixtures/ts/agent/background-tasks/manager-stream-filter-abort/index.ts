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

function deferred(label: string) {
  let resolve!: () => void;
  const promise = new Promise<void>((res) => {
    resolve = res;
  });
  return { label, promise, resolve };
}

async function readUntil(
  reader: any,
  predicate: (chunk: any) => boolean,
  label: string,
): Promise<any> {
  return withTimeout(label, 30_000, async () => {
    while (true) {
      const { value, done } = await reader.read();
      if (done) throw new Error(`stream closed before ${label}`);
      if (predicate(value)) return value;
    }
  });
}

async function expectNoChunkWithin(reader: any, ms: number): Promise<boolean> {
  const result = await Promise.race([
    reader.read().then((next) => ({ kind: "chunk", next })),
    sleep(ms).then(() => ({ kind: "timeout" })),
  ]);
  return result.kind === "timeout";
}

async function waitForStarted(started: string[], label: string): Promise<void> {
  await withTimeout(`started:${label}`, 30_000, async () => {
    while (!started.includes(label)) {
      await sleep(25);
    }
  });
}

const storage = new InMemoryStore();
const mastra = new Mastra({
  storage,
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 4,
    perAgentConcurrency: 4,
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

function makeTask(label: string, agentId: string, runId: string) {
  const gate = deferred(label);
  releases.set(label, gate);

  return createBackgroundTask(manager, {
    toolName: "stream-filter-task",
    toolCallId: `call-${label}`,
    args: { label },
    agentId,
    runId,
    context: {
      executor: {
        execute: async () => {
          started.push(label);
          await gate.promise;
          return { label, agentId, runId };
        },
      },
    },
  });
}

try {
  const agentA = makeTask("agent-a", "agent-a", "run-a");
  const agentB = makeTask("agent-b", "agent-b", "run-b");

  await agentA.dispatch();
  await agentB.dispatch();
  await waitForStarted(started, "agent-a");
  await waitForStarted(started, "agent-b");

  const agentAbort = new AbortController();
  const agentStream = manager.stream({
    agentId: "agent-b",
    abortSignal: agentAbort.signal,
  });
  const agentReader = agentStream.getReader();

  const snapshot = await readUntil(
    agentReader,
    (chunk) => chunk.type === "background-task-running",
    "agent-filter-snapshot",
  );

  releases.get("agent-a")?.resolve();
  await sleep(100);
  const agentFilterIgnoredAgentACompletion = await expectNoChunkWithin(agentReader, 200);

  releases.get("agent-b")?.resolve();
  const agentCompletion = await readUntil(
    agentReader,
    (chunk) => chunk.type === "background-task-completed",
    "agent-filter-completion",
  );

  agentAbort.abort();
  const agentClosed = await withTimeout("agent-stream-abort-close", 5_000, async () => {
    const { done } = await agentReader.read();
    return done;
  });

  const taskScoped = makeTask("task-scoped", "agent-c", "run-c");
  await taskScoped.dispatch();
  await waitForStarted(started, "task-scoped");

  const taskAbort = new AbortController();
  const taskStream = manager.stream({
    taskId: taskScoped.task.id,
    abortSignal: taskAbort.signal,
  });
  const taskReader = taskStream.getReader();

  const taskSnapshot = await readUntil(
    taskReader,
    (chunk) => chunk.type === "background-task-running",
    "task-filter-snapshot",
  );

  taskAbort.abort();
  const taskClosed = await withTimeout("task-stream-abort-close", 5_000, async () => {
    const { done } = await taskReader.read();
    return done;
  });

  releases.get("task-scoped")?.resolve();
  await withTimeout("task-scoped-completed", 30_000, async () => {
    while ((await manager.getTask(taskScoped.task.id))?.status !== "completed") {
      await sleep(25);
    }
  });

  output({
    started: started.join(","),
    snapshotAgentMatches: snapshot?.payload?.agentId === "agent-b",
    snapshotRunMatches: snapshot?.payload?.runId === "run-b",
    agentFilterIgnoredAgentACompletion,
    agentCompletionMatches: agentCompletion?.payload?.result?.label === "agent-b",
    agentStreamClosedOnAbort: agentClosed,
    taskSnapshotMatches: taskSnapshot?.payload?.taskId === taskScoped.task.id,
    taskStreamClosedOnAbort: taskClosed,
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

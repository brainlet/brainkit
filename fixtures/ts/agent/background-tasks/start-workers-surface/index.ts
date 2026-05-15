import { InMemoryStore, Mastra } from "agent";
import { output } from "kit";

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

const mastra = new Mastra({
  storage: new InMemoryStore(),
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 2,
    perAgentConcurrency: 2,
  },
});

const workerNames = Array.from((mastra as any).workers ?? []).map((worker: any) => worker.name).join(",");

await withTimeout("startWorkers", 5_000, async () => {
  await mastra.startWorkers();
});

await withTimeout("stopWorkers", 5_000, async () => {
  await mastra.backgroundTaskManager?.shutdown();
  await mastra.stopWorkers();
});

output({
  workerNames,
  hasOrchestrationWorker: workerNames.includes("orchestration"),
  hasBackgroundTaskWorker: workerNames.includes("backgroundTasks"),
  startWorkersReturned: true,
  stopWorkersReturned: true,
});

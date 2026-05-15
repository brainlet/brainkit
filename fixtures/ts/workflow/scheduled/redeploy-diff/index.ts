import { __brainkitMastraBackgroundTaskDebug, InMemoryStore, Mastra, createStep, createWorkflow, z } from "agent";
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

async function waitFor<T>(label: string, predicate: () => Promise<T | undefined> | T | undefined): Promise<T> {
  return withTimeout(label, 20_000, async () => {
    while (true) {
      const value = await predicate();
      if (value) return value;
      await sleep(25);
    }
  });
}

function debug() {
  const snapshot = __brainkitMastraBackgroundTaskDebug?.();
  if (!snapshot) throw new Error("missing Brainkit Mastra lifecycle debug hook");
  return snapshot;
}

const workflowId = "scheduled-redeploy";
const legacyRowId = "wf_scheduled-redeploy";
const rowA = "wf_scheduled-redeploy__a";
const rowB = "wf_scheduled-redeploy__b";
const userRow = "user-created-schedule";

const scheduledStep = createStep({
  id: "scheduled-redeploy-step",
  inputSchema: z.object({ rev: z.number() }),
  outputSchema: z.object({ ok: z.boolean(), rev: z.number() }),
  execute: async ({ inputData }: any) => ({ ok: true, rev: inputData.rev }),
});

function buildSingleWorkflow(cron: string, rev: number) {
  return createWorkflow({
    id: workflowId,
    inputSchema: z.object({ rev: z.number() }),
    outputSchema: z.any(),
    schedule: {
      cron,
      inputData: { rev },
      requestContext: { source: `scheduled-redeploy-rev-${rev}` },
      metadata: { fixture: "scheduled-redeploy", rev },
    },
  })
    .then(scheduledStep)
    .commit();
}

function buildArrayWorkflow(ids: string[]) {
  return createWorkflow({
    id: workflowId,
    inputSchema: z.object({ rev: z.number() }),
    outputSchema: z.any(),
    schedule: ids.map((id, index) => ({
      id,
      cron: "* * * * *",
      inputData: { rev: index + 10 },
      requestContext: { source: `scheduled-redeploy-${id}` },
      metadata: { fixture: "scheduled-redeploy", entry: id },
    })),
  })
    .then(scheduledStep)
    .commit();
}

async function boot(storage: InMemoryStore, workflow: any): Promise<Mastra> {
  const mastra = new Mastra({
    storage,
    workflows: { workflow },
    scheduler: { enabled: true, tickIntervalMs: 60_000 },
  });
  await waitFor("scheduler-running", () => (mastra.scheduler?.isRunning ? mastra : undefined));
  return mastra;
}

async function sortedScheduleIds(schedulesStore: any): Promise<string> {
  const rows = await schedulesStore.listSchedules();
  return rows.map((row: any) => String(row.id)).sort().join(",");
}

const storage = new InMemoryStore();
const activeMastras: Mastra[] = [];
let observed: Record<string, unknown> = {};

try {
  const first = await boot(storage, buildSingleWorkflow("0 9 * * 1", 1));
  activeMastras.push(first);
  const schedulesStore = await waitFor("schedules-store", async () => {
    const store = await first.getStorage()?.getStore?.("schedules");
    return store || undefined;
  });

  const initial = await waitFor("initial-single-row", async () => {
    const row = await schedulesStore.getSchedule(legacyRowId);
    return row || undefined;
  });
  await schedulesStore.updateSchedule(legacyRowId, { status: "paused" });
  const pausedBeforeRedeploy = await schedulesStore.getSchedule(legacyRowId);

  await first.shutdown();
  activeMastras.pop();

  const second = await boot(storage, buildSingleWorkflow("30 14 * * 5", 2));
  activeMastras.push(second);
  const afterSingleRedeploy = await waitFor("single-row-updated", async () => {
    const row = await schedulesStore.getSchedule(legacyRowId);
    return row?.cron === "30 14 * * 5" ? row : undefined;
  });
  await second.shutdown();
  activeMastras.pop();

  const third = await boot(storage, buildArrayWorkflow(["a", "b"]));
  activeMastras.push(third);
  await waitFor("array-rows", async () => {
    const ids = await sortedScheduleIds(schedulesStore);
    return ids === `${rowA},${rowB}` ? ids : undefined;
  });
  const arrayRowIds = await sortedScheduleIds(schedulesStore);
  await third.shutdown();
  activeMastras.pop();

  await schedulesStore.createSchedule({
    id: userRow,
    target: { type: "workflow", workflowId: "unrelated" },
    cron: "0 0 * * *",
    status: "active",
    nextFireAt: Date.now() + 60_000,
    createdAt: Date.now(),
    updatedAt: Date.now(),
  });

  const fourth = await boot(storage, buildArrayWorkflow(["a"]));
  activeMastras.push(fourth);
  await waitFor("stale-array-row-deleted", async () => {
    const ids = await sortedScheduleIds(schedulesStore);
    return ids === `${userRow},${rowA}` ? ids : undefined;
  });
  const finalScheduleIds = await sortedScheduleIds(schedulesStore);
  await fourth.shutdown();
  activeMastras.pop();

  observed = {
    initialScheduleRegistered: initial?.id === legacyRowId,
    pausedStatusBeforeRedeploy: String(pausedBeforeRedeploy?.status || ""),
    statusPreservedAfterRedeploy: String(afterSingleRedeploy?.status || ""),
    cronUpdated: String(afterSingleRedeploy?.cron || ""),
    inputUpdated: Number((afterSingleRedeploy as any)?.target?.inputData?.rev || 0),
    requestContextUpdated: String((afterSingleRedeploy as any)?.target?.requestContext?.source || ""),
    metadataUpdated: String((afterSingleRedeploy as any)?.metadata?.rev || ""),
    nextFireRecomputed: Number(afterSingleRedeploy?.nextFireAt || 0) !== Number(initial?.nextFireAt || 0),
    legacyRemovedOnArrayMigration: !(await schedulesStore.getSchedule(legacyRowId)),
    arrayRowIds,
    staleArrayEntryRemoved: !(await schedulesStore.getSchedule(rowB)),
    userSchedulePreserved: Boolean(await schedulesStore.getSchedule(userRow)),
    finalScheduleIds,
  };
} finally {
  while (activeMastras.length > 0) {
    const mastra = activeMastras.pop();
    if (mastra) {
      await withTimeout("shutdown", 10_000, async () => {
        await mastra.shutdown();
      });
    }
  }
}

const finalDebug = debug();

output({
  ...observed,
  finalActiveSchedulerCount: finalDebug.activeSchedulerCount,
});

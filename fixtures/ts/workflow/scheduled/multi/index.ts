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

const executions: Array<{ userId: string; source: string }> = [];

const scheduledStep = createStep({
  id: "scheduled-multi-step",
  inputSchema: z.object({ userId: z.string() }),
  outputSchema: z.object({ ok: z.boolean(), userId: z.string(), source: z.string() }),
  execute: async ({ inputData, requestContext, runtimeContext }: any) => {
    const ctx = runtimeContext || requestContext;
    const source = ctx?.get?.("source") || "missing";
    executions.push({ userId: inputData.userId, source });
    return { ok: true, userId: inputData.userId, source };
  },
});

const scheduledWorkflow = createWorkflow({
  id: "scheduled-multi",
  inputSchema: z.object({ userId: z.string() }),
  outputSchema: z.any(),
  schedule: [
    {
      id: "morning",
      cron: "* * * * *",
      inputData: { userId: "morning" },
      requestContext: { source: "schedule-multi-morning" },
      metadata: { fixture: "scheduled-multi", window: "morning" },
    },
    {
      id: "evening",
      cron: "* * * * *",
      inputData: { userId: "evening" },
      requestContext: { source: "schedule-multi-evening" },
      metadata: { fixture: "scheduled-multi", window: "evening" },
    },
  ],
})
  .then(scheduledStep)
  .commit();

const storage = new InMemoryStore();
const mastra = new Mastra({
  storage,
  workflows: { scheduledWorkflow },
  scheduler: { enabled: true, tickIntervalMs: 60_000 },
});

let observed: Record<string, unknown> = {};

try {
  const morningId = "wf_scheduled-multi__morning";
  const eveningId = "wf_scheduled-multi__evening";
  const scheduleIds = [eveningId, morningId];

  const schedulesStore = await waitFor("schedules-store", async () => {
    const store = await mastra.getStorage()?.getStore?.("schedules");
    return store || undefined;
  });

  await waitFor("declarative-multi-schedules", async () => {
    const rows = await schedulesStore.listSchedules();
    const ids = rows.map((row: any) => String(row.id)).sort();
    return ids.join(",") === scheduleIds.join(",") ? rows : undefined;
  });
  await waitFor("scheduler-running", () => (mastra.scheduler?.isRunning ? mastra.scheduler : undefined));

  const initialRows = (await schedulesStore.listSchedules()).sort((a: any, b: any) =>
    String(a.id).localeCompare(String(b.id)),
  );
  const initialRowsById = new Map(initialRows.map((row: any) => [String(row.id), row]));

  await withTimeout("start-workers", 10_000, async () => {
    await mastra.startWorkers();
  });
  const debugAfterStart = debug();

  const morningDueAt = Date.now() - 2_000;
  const eveningDueAt = Date.now() - 1_000;
  await schedulesStore.updateSchedule(morningId, { nextFireAt: morningDueAt });
  await schedulesStore.updateSchedule(eveningId, { nextFireAt: eveningDueAt });
  await mastra.scheduler?.tick();

  await waitFor("scheduled-multi-workflow-executions", () => (executions.length >= 2 ? executions : undefined));

  const updatedMorning = await schedulesStore.getSchedule(morningId);
  const updatedEvening = await schedulesStore.getSchedule(eveningId);
  const morningTriggers = await schedulesStore.listTriggers(morningId);
  const eveningTriggers = await schedulesStore.listTriggers(eveningId);
  const morningExpectedRunId = `sched_${morningId}_${morningDueAt}`;
  const eveningExpectedRunId = `sched_${eveningId}_${eveningDueAt}`;
  const debugAfterFire = debug();

  const executionUsers = executions.map((entry) => entry.userId).sort();
  const executionSources = executions.map((entry) => entry.source).sort();

  observed = {
    engineType: String((scheduledWorkflow as any).engineType || ""),
    scheduleCount: initialRows.length,
    scheduleIds: initialRows.map((row: any) => String(row.id)).join(","),
    metadataWindows: initialRows.map((row: any) => String(row.metadata?.window || "")).sort().join(","),
    morningInputUser: String((initialRowsById.get(morningId) as any)?.target?.inputData?.userId || ""),
    eveningInputUser: String((initialRowsById.get(eveningId) as any)?.target?.inputData?.userId || ""),
    workerSubscriptionStarted: debugAfterStart.activeWorkflowEventSubscriptions >= 1,
    schedulerRunningBeforeShutdown: mastra.scheduler?.isRunning === true,
    executionCount: executions.length,
    executionUsers: executionUsers.join(","),
    executionSources: executionSources.join(","),
    morningTriggerOutcome: String(morningTriggers[0]?.outcome || ""),
    eveningTriggerOutcome: String(eveningTriggers[0]?.outcome || ""),
    morningRunIdMatches: morningTriggers[0]?.runId === morningExpectedRunId,
    eveningRunIdMatches: eveningTriggers[0]?.runId === eveningExpectedRunId,
    morningLastRunIdMatches: updatedMorning?.lastRunId === morningExpectedRunId,
    eveningLastRunIdMatches: updatedEvening?.lastRunId === eveningExpectedRunId,
    morningNextFireAdvanced: Number(updatedMorning?.nextFireAt || 0) > morningDueAt,
    eveningNextFireAdvanced: Number(updatedEvening?.nextFireAt || 0) > eveningDueAt,
    workflowEventsHandledAtLeastTwo: debugAfterFire.workflowEventsOk >= 2,
    workflowEventFailures: debugAfterFire.workflowEventsFailed,
  };
} finally {
  await withTimeout("shutdown", 10_000, async () => {
    await mastra.shutdown();
  });
}

const finalDebug = debug();

output({
  ...observed,
  finalActiveWorkflowSubscriptions: finalDebug.activeWorkflowEventSubscriptions,
  finalActiveSchedulerCount: finalDebug.activeSchedulerCount,
});

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

function zonePart(ts: number, timeZone: string, type: string): string {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    year: "numeric",
    month: "numeric",
    day: "numeric",
    hour: "numeric",
    minute: "numeric",
    second: "numeric",
    hour12: false,
  }).formatToParts(new Date(ts));
  return String(parts.find((part) => part.type === type)?.value || "");
}

const executions: Array<{ userId: string; source: string }> = [];

const scheduledStep = createStep({
  id: "scheduled-timezone-step",
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
  id: "scheduled-timezone",
  inputSchema: z.object({ userId: z.string() }),
  outputSchema: z.any(),
  schedule: {
    cron: "0 9 * * *",
    timezone: "America/New_York",
    inputData: { userId: "timezone-user" },
    requestContext: { source: "schedule-timezone-fixture" },
    metadata: { fixture: "scheduled-timezone" },
  },
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
  const scheduleId = "wf_scheduled-timezone";
  const schedulesStore = await waitFor("schedules-store", async () => {
    const store = await mastra.getStorage()?.getStore?.("schedules");
    return store || undefined;
  });

  const initialSchedule = await waitFor("declarative-timezone-schedule", async () => {
    const row = await schedulesStore.getSchedule(scheduleId);
    return row || undefined;
  });
  await waitFor("scheduler-running", () => (mastra.scheduler?.isRunning ? mastra.scheduler : undefined));

  await withTimeout("start-workers", 10_000, async () => {
    await mastra.startWorkers();
  });
  const debugAfterStart = debug();

  const dueAt = Date.now() - 1_000;
  await schedulesStore.updateSchedule(scheduleId, { nextFireAt: dueAt });
  await mastra.scheduler?.tick();

  await waitFor("scheduled-timezone-workflow-execution", () => executions[0]);

  const updatedSchedule = await schedulesStore.getSchedule(scheduleId);
  const triggers = await schedulesStore.listTriggers(scheduleId);
  const expectedRunId = `sched_${scheduleId}_${dueAt}`;
  const debugAfterFire = debug();

  const nextFireAt = Number(updatedSchedule?.nextFireAt || 0);

  observed = {
    engineType: String((scheduledWorkflow as any).engineType || ""),
    scheduleRegistered: initialSchedule?.id === scheduleId,
    scheduleCron: String(initialSchedule?.cron || ""),
    scheduleTimezone: String(initialSchedule?.timezone || ""),
    scheduleMetadata: String(initialSchedule?.metadata?.fixture || ""),
    initialNextFireNYHour: zonePart(Number(initialSchedule?.nextFireAt || 0), "America/New_York", "hour"),
    initialNextFireNYMinute: zonePart(Number(initialSchedule?.nextFireAt || 0), "America/New_York", "minute"),
    updatedNextFireNYHour: zonePart(nextFireAt, "America/New_York", "hour"),
    updatedNextFireNYMinute: zonePart(nextFireAt, "America/New_York", "minute"),
    workerSubscriptionStarted: debugAfterStart.activeWorkflowEventSubscriptions >= 1,
    schedulerRunningBeforeShutdown: mastra.scheduler?.isRunning === true,
    executionCount: executions.length,
    executionUserId: executions[0]?.userId || "",
    executionSource: executions[0]?.source || "",
    triggerOutcome: String(triggers[0]?.outcome || ""),
    triggerRunIdMatches: triggers[0]?.runId === expectedRunId,
    lastRunIdMatches: updatedSchedule?.lastRunId === expectedRunId,
    nextFireAdvanced: nextFireAt > dueAt,
    workflowEventHandled: debugAfterFire.workflowEventsOk >= 1,
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

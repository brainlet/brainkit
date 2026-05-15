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
  id: "scheduled-pause-resume-step",
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
  id: "scheduled-pause-resume",
  inputSchema: z.object({ userId: z.string() }),
  outputSchema: z.any(),
  schedule: {
    cron: "* * * * *",
    inputData: { userId: "paused-user" },
    requestContext: { source: "schedule-pause-resume-fixture" },
    metadata: { fixture: "scheduled-pause-resume" },
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
  const scheduleId = "wf_scheduled-pause-resume";
  const schedulesStore = await waitFor("schedules-store", async () => {
    const store = await mastra.getStorage()?.getStore?.("schedules");
    return store || undefined;
  });

  const initialSchedule = await waitFor("declarative-pause-resume-schedule", async () => {
    const row = await schedulesStore.getSchedule(scheduleId);
    return row || undefined;
  });
  await waitFor("scheduler-running", () => (mastra.scheduler?.isRunning ? mastra.scheduler : undefined));

  await withTimeout("start-workers", 10_000, async () => {
    await mastra.startWorkers();
  });
  const debugAfterStart = debug();

  const dueAt = Date.now() - 1_000;
  await schedulesStore.updateSchedule(scheduleId, { status: "paused", nextFireAt: dueAt });
  await mastra.scheduler?.tick();

  const pausedSchedule = await schedulesStore.getSchedule(scheduleId);
  const pausedTriggers = await schedulesStore.listTriggers(scheduleId);
  const pausedExecutionCount = executions.length;

  await schedulesStore.updateSchedule(scheduleId, { status: "active" });
  await mastra.scheduler?.tick();

  await waitFor("scheduled-pause-resume-execution", () => executions[0]);

  const resumedSchedule = await schedulesStore.getSchedule(scheduleId);
  const triggers = await schedulesStore.listTriggers(scheduleId);
  const expectedRunId = `sched_${scheduleId}_${dueAt}`;
  const debugAfterFire = debug();

  observed = {
    engineType: String((scheduledWorkflow as any).engineType || ""),
    initialStatus: String(initialSchedule?.status || ""),
    pausedStatus: String(pausedSchedule?.status || ""),
    skippedWhilePaused: pausedExecutionCount === 0 && pausedTriggers.length === 0,
    nextFireUnchangedWhilePaused: Number(pausedSchedule?.nextFireAt || 0) === dueAt,
    resumedStatus: String(resumedSchedule?.status || ""),
    workerSubscriptionStarted: debugAfterStart.activeWorkflowEventSubscriptions >= 1,
    schedulerRunningBeforeShutdown: mastra.scheduler?.isRunning === true,
    executionCount: executions.length,
    executionUserId: executions[0]?.userId || "",
    executionSource: executions[0]?.source || "",
    triggerOutcome: String(triggers[0]?.outcome || ""),
    triggerRunIdMatches: triggers[0]?.runId === expectedRunId,
    lastRunIdMatches: resumedSchedule?.lastRunId === expectedRunId,
    nextFireAdvancedAfterResume: Number(resumedSchedule?.nextFireAt || 0) > dueAt,
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

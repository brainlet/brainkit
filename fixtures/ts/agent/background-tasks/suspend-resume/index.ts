import { Agent, createTool, InMemoryStore, Mastra, Memory, z } from "agent";
import { model, output } from "kit";

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
      await sleep(50);
    }
  });
}

const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const approvalResearchTool = createTool({
  id: "approval-research",
  description: 'Research a topic after analyst approval. Use this tool whenever the user asks to research "solana".',
  inputSchema: z.object({
    topic: z.string().describe("The topic to research"),
  }),
  outputSchema: z.object({
    summary: z.string(),
  }),
  background: {
    enabled: true,
    timeoutMs: 30_000,
  },
  execute: async ({ topic }: any, context: any) => {
    const agentContext = context?.agent ?? context ?? {};
    const resumeData = agentContext.resumeData ?? context?.resumeData;
    const suspend = agentContext.suspend ?? context?.suspend;
    if (!resumeData?.approved) {
      await suspend?.({ awaiting: "analyst-approval", topic });
      return { summary: "" };
    }
    return {
      summary: `Research complete on "${topic}": ${resumeData.notes ?? "approved"}.`,
    };
  },
});

const agent = new Agent({
  id: "background-suspend-agent",
  name: "Background Suspend Agent",
  instructions:
    'You must call approval-research when the user asks to research "solana". ' +
    "After the approved research result is available, mention solana and the analyst notes.",
  model: model("openai", "gpt-4o-mini"),
  tools: { approvalResearch: approvalResearchTool },
  memory,
  backgroundTasks: {
    tools: {
      approvalResearch: true,
    },
  },
  maxSteps: 3,
});

const mastra = new Mastra({
  agents: { "background-suspend-agent": agent },
  storage,
  backgroundTasks: {
    enabled: true,
    globalConcurrency: 5,
    perAgentConcurrency: 3,
    defaultTimeoutMs: 30_000,
  },
});

let workersStarted = false;

await withTimeout("startWorkers", 10_000, async () => {
  await mastra.startWorkers("backgroundTasks");
  workersStarted = true;
});

try {
  const liveAgent = mastra.getAgent("background-suspend-agent");
  const stream1 = await withTimeout("initialStreamUntilIdle", 120_000, () =>
    liveAgent.streamUntilIdle('Please research "solana" for me.', {
      memory: {
        thread: { id: "background-suspend-thread" },
        resource: "background-suspend-user",
      },
      maxSteps: 3,
      maxIdleMs: 10_000,
      modelSettings: { temperature: 0 },
    }),
  );

  const chunks1: any[] = [];
  await withTimeout("initialFullStream", 120_000, async () => {
    for await (const chunk of stream1.fullStream) {
      chunks1.push(chunk);
    }
  });

  const bgStarted = chunks1.find((chunk) => chunk.type === "background-task-started");
  const bgSuspended = chunks1.find((chunk) => chunk.type === "background-task-suspended");
  const taskId = bgStarted?.payload?.taskId ?? bgSuspended?.payload?.taskId;
  const manager = mastra.backgroundTaskManager;
  const suspendedTask = taskId ? await manager?.getTask(taskId) : undefined;

  if (taskId) {
    await manager?.resume(taskId, { approved: true, notes: "looks promising" });
  }
  const completedTask = taskId ? await waitForTaskStatus(manager, taskId, "completed") : undefined;

  const stream2 = await withTimeout("followupStreamUntilIdle", 120_000, () =>
    liveAgent.streamUntilIdle("What did the approved research find? Mention the analyst notes.", {
      memory: {
        thread: { id: "background-suspend-thread" },
        resource: "background-suspend-user",
      },
      maxSteps: 3,
      maxIdleMs: 10_000,
      modelSettings: { temperature: 0 },
    }),
  );

  const chunks2: any[] = [];
  await withTimeout("followupFullStream", 120_000, async () => {
    for await (const chunk of stream2.fullStream) {
      chunks2.push(chunk);
    }
  });

  const text = chunks2
    .filter((chunk) => chunk.type === "text-delta")
    .map((chunk) => chunk.payload?.text ?? chunk.delta ?? "")
    .join("")
    .toLowerCase();

  output({
    firstChunkTypes: chunks1.map((chunk) => chunk.type).join(","),
    secondChunkTypes: chunks2.map((chunk) => chunk.type).join(","),
    hasBackgroundStarted: !!bgStarted,
    hasBackgroundSuspended: !!bgSuspended,
    hasNoInitialCompletion: !chunks1.some((chunk) => chunk.type === "background-task-completed"),
    suspendedStatus: suspendedTask?.status ?? "",
    suspendPayloadTopic: bgSuspended?.payload?.suspendPayload?.topic ?? "",
    completedStatus: completedTask?.status ?? "",
    completedResultIncludesTopic: String((completedTask?.result as any)?.summary ?? "")
      .toLowerCase()
      .includes("solana"),
    completedResultIncludesNotes: String((completedTask?.result as any)?.summary ?? "")
      .toLowerCase()
      .includes("promising"),
    followupTextIncludesTopic: text.includes("solana"),
    followupTextIncludesNotes: text.includes("promising"),
  });
} finally {
  if (workersStarted) {
    await withTimeout("shutdown", 10_000, async () => {
      await mastra.backgroundTaskManager?.shutdown();
      await mastra.stopWorkers();
    });
  }
}

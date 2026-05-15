import { Agent, Harness, InMemoryStore, Memory } from "agent";
import { model, output } from "kit";

async function withTimeout<T>(label: string, timeoutMs: number, fn: () => Promise<T>): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      fn(),
      new Promise<T>((_, reject) => {
        timer = setTimeout(() => reject(new Error(`timeout:${label}`)), timeoutMs);
      }),
    ]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

function resolveHarnessModel(modelId: string) {
  const slash = modelId.indexOf("/");
  if (slash < 0) return model("openai", modelId);
  return model(modelId.slice(0, slash), modelId.slice(slash + 1));
}

const events: any[] = [];
const snapshots: any[] = [];
const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 20 },
});

const parentAgent = new Agent({
  name: "harness-forked-subagent-parent",
  model: model("openai", "gpt-4o-mini"),
  instructions: [
    "You are a deterministic Harness fixture agent.",
    "When asked to retrieve a secret code from conversation history, reply with exactly that code and nothing else.",
    "Do not call tools when retrieving the secret code from history.",
  ].join("\n"),
  memory,
});

const harness = new Harness({
  id: "harness-subagents-forked",
  storage,
  memory,
  initialState: { yolo: true },
  modes: [
    {
      id: "default",
      name: "Default",
      default: true,
      defaultModelId: "openai/gpt-4o-mini",
      agent: parentAgent,
    },
  ],
  subagents: [
    {
      id: "context",
      name: "Context Fork",
      description: "Reads the cloned parent thread and answers from inherited conversation context.",
      forked: true,
      instructions: "Unused for forked runs; the parent agent is reused.",
    },
  ],
  resolveModel: resolveHarnessModel,
});

const unsubscribe = harness.subscribe((event) => {
  events.push(event);
  if (event.type === "subagent_start" || event.type === "subagent_end") {
    const displayState = harness.getDisplayState() as any;
    const entry = displayState.activeSubagents?.get(event.toolCallId);
    snapshots.push({
      type: event.type,
      toolCallId: event.toolCallId,
      status: entry?.status,
      agentType: entry?.agentType,
      displayName: entry?.displayName,
      modelId: entry?.modelId,
      forked: entry?.forked,
      result: entry?.result,
    });
  }
});

try {
  await harness.init();
  const parentThread = await harness.createThread({ title: "Fork seed" });
  await memory.saveMessages({
    messages: [
      {
        id: "fixture-fork-seed-user",
        threadId: parentThread.id,
        resourceId: parentThread.resourceId,
        role: "user",
        type: "text",
        createdAt: new Date(),
        content: {
          format: 2,
          parts: [
            {
              type: "text",
              text: "The secret code for this thread is FORK_CONTEXT_42.",
            },
          ],
        } as any,
      } as any,
    ],
  });

  const parentThreadId = harness.getCurrentThreadId();
  const requestContext = await (harness as any).buildRequestContext();
  const toolsets = await (harness as any).buildToolsets(requestContext);
  const subagentTool = toolsets?.harnessBuiltIn?.subagent;
  if (!subagentTool?.execute) throw new Error("subagent tool unavailable");

  const result: any = await withTimeout("forkedSubagent", 180_000, () =>
    subagentTool.execute(
      {
        agentType: "context",
        task: "Read the prior user message in this thread and reply with exactly the secret code it contains.",
        forked: true,
      },
      { requestContext, agent: { toolCallId: "fixture-forked-subagent" } },
    ),
  );

  const displayState = harness.getDisplayState() as any;
  const listedThreads = await harness.listThreads();
  const listedThreadsWithForks = await harness.listThreads({ includeForkedSubagents: true });
  const forkThread = listedThreadsWithForks.find((thread: any) => thread.metadata?.forkedSubagent === true);
  const forkMessages = forkThread?.id
    ? await harness.listMessagesForThread({ threadId: forkThread.id })
    : [];
  const subagentStart = events.find((event) => event.type === "subagent_start");
  const subagentEnd = events.find((event) => event.type === "subagent_end");
  const startSnapshot = snapshots.find((snapshot) => snapshot.type === "subagent_start");
  const endSnapshot = snapshots.find((snapshot) => snapshot.type === "subagent_end");

  output({
    parentThreadCreated: typeof parentThreadId === "string" && parentThreadId.length > 0,
    subagentToolAvailable: Boolean(subagentTool),
    subagentStarted: subagentStart?.type === "subagent_start",
    subagentEnded: subagentEnd?.type === "subagent_end",
    subagentAgentType: subagentStart?.agentType,
    subagentModelId: subagentStart?.modelId,
    subagentForked: subagentStart?.forked,
    resultOk: result?.isError === false,
    resultContainsSecret: String(result?.content || "").includes("FORK_CONTEXT_42"),
    endResultContainsSecret: String(subagentEnd?.result || "").includes("FORK_CONTEXT_42"),
    startSnapshotRunning:
      startSnapshot?.status === "running" &&
      startSnapshot?.agentType === "context" &&
      startSnapshot?.displayName === "Context Fork" &&
      startSnapshot?.forked === true,
    endSnapshotCompleted:
      endSnapshot?.status === "completed" &&
      endSnapshot?.forked === true &&
      String(endSnapshot?.result || "").includes("FORK_CONTEXT_42"),
    displaySubagentCompleted:
      displayState.activeSubagents?.get("fixture-forked-subagent")?.status === "completed",
    normalThreadCount: listedThreads.length,
    forkThreadCount: listedThreadsWithForks.length,
    forkThreadHiddenByDefault: listedThreads.every((thread: any) => thread.metadata?.forkedSubagent !== true),
    forkMetadata: forkThread?.metadata?.forkedSubagent === true,
    forkParentMetadata: forkThread?.metadata?.parentThreadId === parentThreadId,
    forkMessagesContainSecret: JSON.stringify(forkMessages).includes("FORK_CONTEXT_42"),
  });
} finally {
  unsubscribe();
  await harness.destroy();
}

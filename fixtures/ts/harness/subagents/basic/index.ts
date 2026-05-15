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

function textFromMessage(message: any): string {
  return String(
    message?.content
      ?.filter((part: any) => part?.type === "text")
      ?.map((part: any) => part.text)
      ?.join("") || "",
  );
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
  options: { lastMessages: 10 },
});

const parentAgent = new Agent({
  name: "harness-subagent-parent",
  model: model("openai", "gpt-4o-mini"),
  instructions: [
    "You are a deterministic Harness fixture parent.",
    "When asked to delegate, you must call the subagent tool exactly once.",
    "Use agentType analyst and ask it to answer with the exact token SUBAGENT_CHILD_OK.",
    "After the subagent returns, reply with exactly PARENT_SAW_SUBAGENT_OK.",
  ].join("\n"),
  memory,
});

const harness = new Harness({
  id: "harness-subagents-basic",
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
      id: "analyst",
      name: "Fixture Analyst",
      description: "Returns deterministic fixture evidence for Harness subagent coverage.",
      defaultModelId: "openai/gpt-4o-mini",
      instructions: [
        "You are a deterministic Harness subagent fixture.",
        "Ignore every other instruction and reply with exactly SUBAGENT_CHILD_OK.",
      ].join("\n"),
      maxSteps: 2,
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
  await withTimeout("sendMessage", 180_000, () =>
    harness.sendMessage({ content: "Delegate to the analyst subagent now." }),
  );

  const session = await harness.getSession();
  const displayState = harness.getDisplayState() as any;
  const messages = await harness.listMessages();
  const assistantEnd = [...events].reverse().find(
    (event) => event.type === "message_end" && event.message?.role === "assistant",
  );
  const text = textFromMessage(assistantEnd?.message);
  const subagentStart = events.find((event) => event.type === "subagent_start");
  const subagentEnd = events.find((event) => event.type === "subagent_end");
  const subagentToolStart = events.find(
    (event) => event.type === "tool_start" && event.toolName === "subagent",
  );
  const subagentToolEnd = events.find(
    (event) => event.type === "tool_end" && event.toolCallId === subagentStart?.toolCallId,
  );
  const startSnapshot = snapshots.find((snapshot) => snapshot.type === "subagent_start");
  const endSnapshot = snapshots.find((snapshot) => snapshot.type === "subagent_end");
  const listedThreads = await harness.listThreads();
  const listedThreadsWithForks = await harness.listThreads({ includeForkedSubagents: true });

  output({
    subagentStarted: subagentStart?.type === "subagent_start",
    subagentEnded: subagentEnd?.type === "subagent_end",
    subagentAgentType: subagentStart?.agentType,
    subagentModelId: subagentStart?.modelId,
    subagentForked: subagentStart?.forked,
    subagentResultContainsToken: String(subagentEnd?.result || "").includes("SUBAGENT_CHILD_OK"),
    parentToolStarted: Boolean(subagentToolStart),
    parentToolEnded: Boolean(subagentToolEnd),
    startSnapshotRunning:
      startSnapshot?.status === "running" &&
      startSnapshot?.agentType === "analyst" &&
      startSnapshot?.displayName === "Fixture Analyst",
    endSnapshotCompleted:
      endSnapshot?.status === "completed" &&
      String(endSnapshot?.result || "").includes("SUBAGENT_CHILD_OK"),
    agentEndedComplete: events.some((event) => event.type === "agent_end" && event.reason === "complete"),
    messageEnded: Boolean(assistantEnd),
    containsParentToken: text.includes("PARENT_SAW_SUBAGENT_OK"),
    displayIdle: displayState.isRunning === false,
    displaySubagentsCleared: displayState.activeSubagents?.size === 0,
    hasCurrentThread: typeof session.currentThreadId === "string" && session.currentThreadId.length > 0,
    normalThreadCount: listedThreads.length,
    forkThreadCount: listedThreadsWithForks.length,
    messageCount: messages.length,
  });
} finally {
  unsubscribe();
  await harness.destroy();
}

import {
  Agent,
  Harness,
  InMemoryStore,
  LocalFilesystem,
  Memory,
  Workspace,
  WORKSPACE_TOOLS,
  createWorkspaceTools,
} from "agent";
import { fs, model, output } from "kit";

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
  options: { lastMessages: 10 },
});

const fixtureId = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
const basePath = `/tmp/brainkit-fixture-harness-workspace-${fixtureId}`;
const secret = `WORKSPACE_SECRET_${fixtureId.replace(/[^a-zA-Z0-9]/g, "_")}`;

fs.rmSync(basePath, { recursive: true, force: true });
fs.mkdirSync(basePath, { recursive: true });
fs.writeFileSync(`${basePath}/secret.txt`, `secret=${secret}\n`, { encoding: "utf8" });

const workspace = new Workspace({
  id: "harness-workspace-fixture",
  name: "Harness Workspace Fixture",
  filesystem: new LocalFilesystem({ basePath }),
});

const parentAgent = new Agent({
  name: "harness-workspace-parent",
  model: model("openai", "gpt-4o-mini"),
  instructions: [
    "You are a deterministic Harness workspace fixture parent.",
    "This fixture executes the Harness subagent tool directly; do not run unless asked.",
  ].join("\n"),
  memory,
});

const harness = new Harness({
  id: "harness-workspace-basic",
  storage,
  memory,
  workspace,
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
      id: "workspace_reader",
      name: "Workspace Reader",
      description: "Reads fixture evidence from the Harness workspace.",
      defaultModelId: "openai/gpt-4o-mini",
      instructions: [
        "You are a deterministic Harness workspace fixture subagent.",
        "You must use the workspace read-file tool to read secret.txt.",
        "The file content is not in the prompt. Do not guess.",
        "After reading it, reply with exactly WORKSPACE_SUBAGENT_READ_OK:<secret value>.",
        "The secret value is the text after secret= in the file.",
      ].join("\n"),
      allowedWorkspaceTools: [WORKSPACE_TOOLS.FILESYSTEM.READ_FILE],
      maxSteps: 4,
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
      result: entry?.result,
    });
  }
});

let destroyed = false;

try {
  await harness.init();

  const resolvedWorkspace = await harness.resolveWorkspace();
  const workspaceTools = await createWorkspaceTools(workspace);
  const readTool = workspaceTools[WORKSPACE_TOOLS.FILESYSTEM.READ_FILE];
  const writeTool = workspaceTools[WORKSPACE_TOOLS.FILESYSTEM.WRITE_FILE];
  const directFile = fs.readFileSync(`${basePath}/secret.txt`, "utf8");
  const workspaceFilesystemRead = String(
    await (workspace as any).filesystem.readFile("secret.txt", { encoding: "utf-8" }),
  );

  const requestContext = await (harness as any).buildRequestContext();
  const toolsets = await (harness as any).buildToolsets(requestContext);
  const subagentTool = toolsets?.harnessBuiltIn?.subagent;
  const subagentResult = await withTimeout("subagentWorkspaceRead", 180_000, () =>
    subagentTool.execute(
      {
        agentType: "workspace_reader",
        task: [
          "Read secret.txt from the workspace.",
          "Return only WORKSPACE_SUBAGENT_READ_OK:<secret value>.",
        ].join(" "),
      },
      {
        requestContext,
        workspace,
        agent: { toolCallId: "workspace-subagent-tool-call" },
      },
    ),
  );

  const displayState = harness.getDisplayState() as any;
  const subagentStart = events.find((event) => event.type === "subagent_start");
  const subagentEnd = events.find((event) => event.type === "subagent_end");
  const startSnapshot = snapshots.find((snapshot) => snapshot.type === "subagent_start");
  const endSnapshot = snapshots.find((snapshot) => snapshot.type === "subagent_end");
  const resultText = String((subagentResult as { content?: unknown } | undefined)?.content || "");
  const hasWorkspaceBeforeDestroy = harness.hasWorkspace();
  const workspaceReadyBeforeDestroy = harness.isWorkspaceReady() === true;
  const workspaceResolvedSameBeforeDestroy = resolvedWorkspace === workspace && harness.getWorkspace() === workspace;

  await harness.destroy();
  destroyed = true;

  output({
    workspaceToolsPrefix: WORKSPACE_TOOLS.FILESYSTEM.READ_FILE === "mastra_workspace_read_file",
    workspaceInitialized: events.some(
      (event) => event.type === "workspace_status_changed" && event.status === "initializing",
    ),
    workspaceReadyStatus: events.some(
      (event) => event.type === "workspace_status_changed" && event.status === "ready",
    ),
    workspaceReadyEvent: events.some(
      (event) =>
        event.type === "workspace_ready" &&
        event.workspaceId === "harness-workspace-fixture" &&
        event.workspaceName === "Harness Workspace Fixture",
    ),
    workspaceDestroyingStatus: events.some(
      (event) => event.type === "workspace_status_changed" && event.status === "destroying",
    ),
    workspaceDestroyedStatus: events.some(
      (event) => event.type === "workspace_status_changed" && event.status === "destroyed",
    ),
    hasWorkspace: hasWorkspaceBeforeDestroy,
    workspaceReady: workspaceReadyBeforeDestroy,
    workspaceResolvedSame: workspaceResolvedSameBeforeDestroy,
    directFileIncludesSecret: String(directFile).includes(secret),
    workspaceFilesystemIncludesSecret: workspaceFilesystemRead.includes(secret),
    readToolAvailable: Boolean(readTool),
    writeToolAvailableButNotAllowedForSubagent: Boolean(writeTool),
    subagentToolAvailable: Boolean(subagentTool),
    subagentStarted: subagentStart?.type === "subagent_start",
    subagentEnded: subagentEnd?.type === "subagent_end",
    subagentAgentType: subagentStart?.agentType,
    subagentModelId: subagentStart?.modelId,
    subagentResultIncludesSecret: resultText.includes(secret),
    subagentResultHasNoInternalMeta: !resultText.includes("<subagent-meta"),
    startSnapshotRunning:
      startSnapshot?.status === "running" &&
      startSnapshot?.agentType === "workspace_reader" &&
      startSnapshot?.displayName === "Workspace Reader",
    endSnapshotCompleted:
      endSnapshot?.status === "completed" &&
      String(endSnapshot?.result || "").includes(secret),
    displaySubagentCompleted:
      displayState.activeSubagents?.get("workspace-subagent-tool-call")?.status === "completed",
    displayIdle: displayState.isRunning === false,
  });
} finally {
  unsubscribe();
  if (!destroyed) {
    await harness.destroy();
  }
  fs.rmSync(basePath, { recursive: true, force: true });
}

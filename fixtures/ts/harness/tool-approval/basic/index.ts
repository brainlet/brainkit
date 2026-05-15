import { Agent, Harness, InMemoryStore, Memory, createTool, z } from "agent";
import { model, output } from "kit";

async function waitFor<T>(label: string, timeoutMs: number, fn: () => T | undefined): Promise<T> {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const value = fn();
    if (value !== undefined) return value;
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  throw new Error(`timeout:${label}`);
}

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

const events: any[] = [];
const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const deleteRecord = createTool({
  id: "delete-record",
  description: "Delete one record by ID. Always use this when asked to delete a record.",
  inputSchema: z.object({
    id: z.string(),
  }),
  outputSchema: z.object({
    deleted: z.boolean(),
    id: z.string(),
  }),
  requireApproval: true,
  execute: async ({ id }: any) => {
    return { deleted: true, id };
  },
});

const agent = new Agent({
  name: "harness-tool-approval-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions:
    "When asked to delete a record, call the delete-record tool immediately. " +
    "Do not ask for confirmation. After the tool succeeds, reply with exactly DELETE_APPROVED.",
  tools: { "delete-record": deleteRecord },
  memory,
  maxSteps: 3,
});

const harness = new Harness({
  id: "harness-tool-approval",
  storage,
  memory,
  modes: [
    {
      id: "default",
      name: "Default",
      default: true,
      agent,
    },
  ],
});

const unsubscribe = harness.subscribe((event) => {
  events.push(event);
});

try {
  await harness.init();
  const sendPromise = withTimeout("sendMessage", 120_000, () =>
    harness.sendMessage({ content: "Delete record fixture-approval-1." }),
  );

  const approvalEvent = await waitFor("tool_approval_required", 90_000, () =>
    events.find((event) => event.type === "tool_approval_required"),
  );
  const displayWithApproval = harness.getDisplayState() as any;
  const displayApproval =
    displayWithApproval.pendingApproval?.toolCallId === approvalEvent?.toolCallId &&
    displayWithApproval.pendingApproval?.toolName === "delete-record";
  harness.respondToToolApproval({ decision: "approve" });
  await sendPromise;

  const displayAfterApproval = harness.getDisplayState() as any;
  const assistantEnd = [...events].reverse().find(
    (event) => event.type === "message_end" && event.message?.role === "assistant",
  );
  const text = String(
    assistantEnd?.message?.content
      ?.filter((part: any) => part?.type === "text")
      ?.map((part: any) => part.text)
      ?.join("") || "",
  );

  output({
    approvalRequired: approvalEvent?.type === "tool_approval_required",
    approvalToolName: approvalEvent?.toolName,
    approvalArgsContainId: JSON.stringify(approvalEvent?.args || {}).includes("fixture-approval-1"),
    displayApproval,
    toolEnded: events.some((event) => event.type === "tool_end" && event.toolCallId === approvalEvent?.toolCallId),
    agentEndedComplete: events.some((event) => event.type === "agent_end" && event.reason === "complete"),
    messageEnded: Boolean(assistantEnd),
    displayIdle: displayAfterApproval.isRunning === false,
    displayApprovalCleared: displayAfterApproval.pendingApproval === null,
    containsToken: text.includes("DELETE_APPROVED"),
  });
} finally {
  unsubscribe();
  await harness.destroy();
}

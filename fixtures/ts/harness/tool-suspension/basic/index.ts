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

const confirmAction = createTool({
  id: "confirm-action",
  description: "Confirm a deployment action. Always use this when asked to deploy.",
  inputSchema: z.object({
    action: z.string(),
  }),
  outputSchema: z.object({
    confirmed: z.boolean(),
    action: z.string(),
    ticket: z.string(),
  }),
  suspendSchema: z.object({
    action: z.string(),
    reason: z.string(),
  }),
  resumeSchema: z.object({
    confirmed: z.boolean(),
    ticket: z.string(),
  }),
  execute: async ({ action }: any, context: any) => {
    const resumeData = context?.agent?.resumeData ?? context?.workflow?.resumeData ?? context?.resumeData;
    if (resumeData?.confirmed) {
      return {
        confirmed: true,
        action,
        ticket: resumeData.ticket,
      };
    }

    const suspend = context?.suspend ?? context?.agent?.suspend;
    if (!suspend) throw new Error("suspend unavailable");
    await suspend({
      action,
      reason: "Needs live Harness suspension confirmation",
    });

    return {
      confirmed: false,
      action,
      ticket: "pending",
    };
  },
});

const agent = new Agent({
  name: "harness-tool-suspension-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions:
    "When asked to deploy, call the confirm-action tool immediately with action DEPLOY_FIXTURE. " +
    "Do not ask for confirmation. After the tool returns, reply with exactly DEPLOY_CONFIRMED.",
  tools: { confirmAction },
  memory,
  maxSteps: 4,
});

const harness = new Harness({
  id: "harness-tool-suspension",
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
  initialState: { yolo: true },
});

const unsubscribe = harness.subscribe((event) => {
  events.push(event);
});

try {
  await harness.init();
  await withTimeout("sendMessage", 120_000, () =>
    harness.sendMessage({ content: "Deploy the fixture now." }),
  );

  const suspensionEvent = await waitFor("tool_suspended", 5_000, () =>
    events.find((event) => event.type === "tool_suspended"),
  );
  const suspendEnd = events.find((event) => event.type === "agent_end" && event.reason === "suspended");
  const displayWithSuspension = harness.getDisplayState() as any;
  const displaySuspension =
    displayWithSuspension.pendingSuspension?.toolCallId === suspensionEvent?.toolCallId &&
    displayWithSuspension.pendingSuspension?.toolName === "confirmAction";

  events.length = 0;
  await withTimeout("respondToToolSuspension", 120_000, () =>
    harness.respondToToolSuspension({
      resumeData: {
        confirmed: true,
        ticket: "ticket-123",
      },
    }),
  );

  const displayAfterResume = harness.getDisplayState() as any;
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
    suspended: suspensionEvent?.type === "tool_suspended",
    suspendedToolName: suspensionEvent?.toolName,
    suspendPayloadAction: suspensionEvent?.suspendPayload?.action,
    suspendPayloadReason: suspensionEvent?.suspendPayload?.reason,
    agentEndedSuspended: Boolean(suspendEnd),
    displaySuspension,
    resumeStarted: events.some((event) => event.type === "agent_start"),
    toolEnded: events.some((event) => event.type === "tool_end" && event.toolCallId === suspensionEvent?.toolCallId),
    agentEndedComplete: events.some((event) => event.type === "agent_end" && event.reason === "complete"),
    messageEnded: Boolean(assistantEnd),
    displayIdle: displayAfterResume.isRunning === false,
    displaySuspensionCleared: displayAfterResume.pendingSuspension === null,
    containsToken: text.includes("DEPLOY_CONFIRMED"),
  });
} finally {
  unsubscribe();
  await harness.destroy();
}

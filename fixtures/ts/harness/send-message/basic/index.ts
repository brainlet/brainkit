import { Agent, Harness, InMemoryStore, Memory } from "agent";
import { model, output } from "kit";

const events: any[] = [];
const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});

const agent = new Agent({
  name: "harness-fixture-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: [
    "You are a deterministic fixture agent.",
    "When asked, reply with exactly HARNESS_WORKS and no punctuation.",
  ].join("\n"),
});

const harness = new Harness({
  id: "harness-fixture",
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
  await harness.sendMessage({ content: "Reply with the exact fixture token." });

  const session = await harness.getSession();
  const displayState = harness.getDisplayState();
  const messages = await harness.listMessages();
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
    harnessType: typeof Harness,
    sent: true,
    containsToken: text.includes("HARNESS_WORKS"),
    agentStarted: events.some((event) => event.type === "agent_start"),
    agentEnded: events.some((event) => event.type === "agent_end" && event.reason === "complete"),
    messageStarted: events.some((event) => event.type === "message_start"),
    messageEnded: events.some((event) => event.type === "message_end"),
    threadCreated: events.some((event) => event.type === "thread_created"),
    hasCurrentThread: typeof session.currentThreadId === "string" && session.currentThreadId.length > 0,
    currentMode: session.currentModeId,
    threadCount: session.threads.length,
    displayIdle: displayState.isRunning === false,
    messageCount: messages.length,
    text,
  });
} finally {
  unsubscribe();
  await harness.destroy();
}

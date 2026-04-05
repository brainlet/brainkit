// Test: Memory readOnly mode — agent can read history but not save new messages
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

const store = new InMemoryStore();

// Phase 1: write some history with writable memory
const writableMem = new Memory({ storage: store, options: { lastMessages: 10 } });
const writeAgent = new Agent({
  name: "writer",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Remember what the user tells you.",
  memory: writableMem,
});

const threadId = "readonly-" + Date.now();
await writeAgent.generate("My favorite fruit is mango.", {
  memory: { thread: { id: threadId }, resource: "test" },
});

// Phase 2: read-only memory should see history but not add to it
const readOnlyMem = new Memory({ storage: store, options: { lastMessages: 10, readOnly: true } });
const readAgent = new Agent({
  name: "reader",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Answer based on conversation history.",
  memory: readOnlyMem,
});

const result = await readAgent.generate("What is my favorite fruit?", {
  memory: { thread: { id: threadId }, resource: "test" },
});

output({
  hasText: result.text.length > 0,
  knowsMango: result.text.toLowerCase().includes("mango"),
});

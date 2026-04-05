// Test: Working memory with Zod schema instead of string template
import { Agent, Memory, InMemoryStore, z } from "agent";
import { model, output } from "kit";

const store = new InMemoryStore();
const mem = new Memory({
  storage: store,
  options: {
    lastMessages: 10,
    workingMemory: {
      enabled: true,
    },
  },
});

const agent = new Agent({
  name: "schema-wm",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Remember user facts. Update your working memory with what you learn.",
  memory: mem,
});

const threadId = "schema-wm-" + Date.now();
await agent.generate("My name is Bob and I live in Berlin.", {
  memory: { thread: { id: threadId }, resource: "test" },
});

const result = await agent.generate("What do you know about me?", {
  memory: { thread: { id: threadId }, resource: "test" },
});

output({
  hasText: result.text.length > 0,
  knowsBob: result.text.toLowerCase().includes("bob"),
  knowsBerlin: result.text.toLowerCase().includes("berlin"),
});

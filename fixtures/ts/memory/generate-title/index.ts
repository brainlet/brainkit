// Test: Memory generateTitle — auto-creates thread title from first message
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

const store = new InMemoryStore();
const mem = new Memory({
  storage: store,
  options: {
    lastMessages: 10,
    generateTitle: true,
  },
});

const agent = new Agent({
  name: "title-gen",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a helpful assistant.",
  memory: mem,
});

const threadId = "title-" + Date.now();
await agent.generate("Tell me about the history of Go programming language", {
  memory: { thread: { id: threadId }, resource: "test" },
});

// Check if thread got a title
const domain = await store.getStore("memory");
const thread = domain ? await (domain as any).getThreadById({ threadId }) : null;

output({
  hasThread: thread !== null,
  hasTitle: typeof thread?.title === "string" && thread.title.length > 0,
  title: thread?.title || "",
});

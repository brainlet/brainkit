// Test: generateTitle with an explicit model+instructions object.
// The agent runs on gpt-4o-mini but the title is generated using an
// explicit (different-named) config, proving the detached-model path.
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

const store = new InMemoryStore();
const mem = new Memory({
  storage: store,
  options: {
    lastMessages: 10,
    generateTitle: {
      model: model("openai", "gpt-4o-mini"),
      instructions: "Generate a 3-word title that captures the topic.",
    },
  },
});

const agent = new Agent({
  name: "title-custom",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a helpful assistant.",
  memory: mem,
});

const threadId = "title-custom-" + Date.now();
await agent.generate("Explain how goroutines differ from OS threads.", {
  memory: { thread: { id: threadId }, resource: "test" },
});

const domain = await store.getStore("memory");
const thread = domain ? await (domain as any).getThreadById({ threadId }) : null;

output({
  hasThread: thread !== null,
  hasTitle: typeof thread?.title === "string" && thread.title.length > 0,
});

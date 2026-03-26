// Test: observational memory — 3-tier compression
// Requires PostgresStore or MongoDBStore for observational memory support
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

try {
  const store = new InMemoryStore();
  const mem = new Memory({
    storage: store,
    options: {
      lastMessages: 10,
      observationalMemory: true,
    },
  });

  const agent = new Agent({
    name: "om-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Remember everything the user tells you.",
    memory: mem,
  });

  const tid = "om-" + Date.now();
  await agent.generate("My name is Charlie and I live in Paris.", {
    memory: { thread: { id: tid }, resource: "test" },
  });

  const result = await agent.generate("What do you know about me?", {
    memory: { thread: { id: tid }, resource: "test" },
  });

  output({
    hasText: result.text.length > 0,
    knowsCharlie: result.text.toLowerCase().includes("charlie"),
  });
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}

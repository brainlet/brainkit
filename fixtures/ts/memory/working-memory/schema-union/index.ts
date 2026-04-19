// Test: working memory with a Zod union schema — one of several typed
// profile variants. Exercises the SchemaWorkingMemory branch.
import { Agent, Memory, InMemoryStore, z } from "agent";
import { model, output } from "kit";

const store = new InMemoryStore();
const profileSchema = z.union([
  z.object({ type: z.literal("developer"), languages: z.array(z.string()) }),
  z.object({ type: z.literal("designer"), tools: z.array(z.string()) }),
]);

const mem = new Memory({
  storage: store,
  options: {
    lastMessages: 10,
    workingMemory: { enabled: true, schema: profileSchema } as any,
  },
});

const agent = new Agent({
  name: "wm-union",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Learn the user's role then update working memory.",
  memory: mem,
});

const threadId = "wm-union-" + Date.now();
await agent.generate("I'm a developer who writes Go and Rust.", {
  memory: { thread: { id: threadId }, resource: "test" },
});
const result = await agent.generate("What do you know about me?", {
  memory: { thread: { id: threadId }, resource: "test" },
});

output({
  hasText: result.text.length > 0,
  knowsDeveloper: result.text.toLowerCase().includes("developer"),
  knowsLang: result.text.toLowerCase().includes("go") || result.text.toLowerCase().includes("rust"),
});

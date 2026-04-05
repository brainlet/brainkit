// Test: agent memory with PostgreSQL storage (real database)
import { Agent, Memory, PostgresStore } from "agent";
import { model, output } from "kit";

const url = process.env.POSTGRES_URL;
if (!url) throw new Error("POSTGRES_URL not set");

try {
  const store = new PostgresStore({ id: "test-pg", connectionString: url });
  await store.init();
  const mem = new Memory({ storage: store, options: { lastMessages: 10 } });

  const a = new Agent({
    name: "pg-fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Remember what the user tells you.",
    memory: mem,
  });

  const tid = "pg-" + Date.now();
  await a.generate("My favorite animal is a cat.", {
    memory: { thread: { id: tid }, resource: "test-user" },
  });

  const result = await a.generate("What is my favorite animal?", {
    memory: { thread: { id: tid }, resource: "test-user" },
  });

  output({
    text: result.text,
    remembers: result.text.toLowerCase().includes("cat"),
    store: "postgres",
  });
} catch (e: any) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 2000), store: "postgres" });
}

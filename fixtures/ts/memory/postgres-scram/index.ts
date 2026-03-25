// Test: PostgreSQL memory with SCRAM-SHA-256 authentication (password-based)
import { Agent, Memory, PostgresStore } from "agent";
import { model, output } from "kit";

const url = process.env.POSTGRES_URL;
if (!url) throw new Error("POSTGRES_URL not set");

try {
  const store = new PostgresStore({
    id: "test-postgres-scram",
    connectionString: url,
  });

  await store.init();

  const mem = new Memory({
    storage: store,
    options: { lastMessages: 10 },
  });

  const a = new Agent({
    name: "fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a helpful assistant. Remember what the user tells you.",
    memory: mem,
  });

  await a.generate("My favorite number is 42.", {
    memory: { thread: { id: "scram-test-1" }, resource: "test-user" },
  });

  const result = await a.generate("What is my favorite number?", {
    memory: { thread: { id: "scram-test-1" }, resource: "test-user" },
  });

  output({
    text: result.text,
    remembers: result.text.includes("42"),
    auth: "scram-sha-256",
  });
} catch (e: any) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 500) });
}

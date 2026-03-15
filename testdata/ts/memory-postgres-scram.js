// Test: PostgreSQL memory with SCRAM-SHA-256 authentication (password-based)
import { agent, PostgresStore, output } from "brainlet";

const url = globalThis.process?.env?.POSTGRES_URL;
if (!url) throw new Error("POSTGRES_URL not set");

const store = new PostgresStore({
  id: "test-postgres-scram",
  connectionString: url,
});

await store.init();

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: {
    thread: "scram-test-1",
    resource: "test-user",
    storage: store,
  },
});

await a.generate("My favorite number is 42.");

const result = await a.generate("What is my favorite number?");

output({
  text: result.text,
  remembers: result.text.includes("42"),
  auth: "scram-sha-256",
});

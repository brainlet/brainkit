import { Agent, Memory, PostgresStore } from "agent";
import { model, output } from "kit";
const url = process.env.POSTGRES_URL;
if (!url) throw new Error("POSTGRES_URL not set");
try {
  const store = new PostgresStore({ id: "agent-pg", connectionString: url });
  await store.init();
  const mem = new Memory({ storage: store, options: { lastMessages: 10 } });
  const agent = new Agent({ name: "pg-mem", model: model("openai", "gpt-4o-mini"), instructions: "Remember what the user tells you.", memory: mem });
  const tid = "pg-" + Date.now();
  await agent.generate("My favorite language is Rust.", { memory: { thread: { id: tid }, resource: "test" } });
  const r = await agent.generate("What is my favorite language?", { memory: { thread: { id: tid }, resource: "test" } });
  output({ text: r.text, remembers: r.text.toLowerCase().includes("rust"), backend: "postgres" });
} catch(e: any) { output({ error: e.message.substring(0, 200) }); }

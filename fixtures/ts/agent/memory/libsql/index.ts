import { Agent, Memory, LibSQLStore } from "agent";
import { model, output } from "kit";
const url = process.env.LIBSQL_URL;
if (!url) throw new Error("LIBSQL_URL not set");
try {
  const store = new LibSQLStore({ id: "agent-libsql", url });
  const mem = new Memory({ storage: store, options: { lastMessages: 10 } });
  const agent = new Agent({ name: "libsql-mem", model: model("openai", "gpt-4o-mini"), instructions: "Remember what the user tells you.", memory: mem });
  const tid = "libsql-" + Date.now();
  await agent.generate("My pet is a cat named Luna.", { memory: { thread: { id: tid }, resource: "test" } });
  const r = await agent.generate("What is my pet's name?", { memory: { thread: { id: tid }, resource: "test" } });
  output({ text: r.text, remembers: r.text.toLowerCase().includes("luna"), backend: "libsql" });
} catch(e: any) { output({ error: e.message.substring(0, 200) }); }

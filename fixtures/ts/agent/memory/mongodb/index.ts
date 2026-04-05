import { Agent, Memory, MongoDBStore } from "agent";
import { model, output } from "kit";
const url = process.env.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");
try {
  const store = new MongoDBStore({ id: "agent-mongo", url, dbName: "agent_mem_test" });
  await store.init();
  const mem = new Memory({ storage: store, options: { lastMessages: 10 } });
  const agent = new Agent({ name: "mongo-mem", model: model("openai", "gpt-4o-mini"), instructions: "Remember what the user tells you.", memory: mem });
  const tid = "mongo-" + Date.now();
  await agent.generate("My favorite color is teal.", { memory: { thread: { id: tid }, resource: "test" } });
  const r = await agent.generate("What is my favorite color?", { memory: { thread: { id: tid }, resource: "test" } });
  output({ text: r.text, remembers: r.text.toLowerCase().includes("teal"), backend: "mongodb" });
} catch(e: any) { output({ error: e.message.substring(0, 200) }); }

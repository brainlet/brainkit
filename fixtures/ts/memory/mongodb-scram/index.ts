import { Agent, Memory, MongoDBStore } from "agent";
import { model, output } from "kit";
const url = process.env.MONGODB_URL;
if (!url) throw new Error("MONGODB_URL not set");
try {
  const store = new MongoDBStore({ id: "scram-mem", url, dbName: "scram_test_db" });
  await store.init();
  const mem = new Memory({ storage: store, options: { lastMessages: 10 } });
  const agent = new Agent({ name: "scram-agent", model: model("openai", "gpt-4o-mini"), instructions: "Remember what the user tells you.", memory: mem });
  const tid = "scram-" + Date.now();
  await agent.generate("My favorite city is Kyoto.", { memory: { thread: { id: tid }, resource: "scram" } });
  const r = await agent.generate("What is my favorite city?", { memory: { thread: { id: tid }, resource: "scram" } });
  output({ text: r.text, remembers: r.text.toLowerCase().includes("kyoto"), auth: "scram-sha-256" });
} catch(e: any) { output({ error: e.message.substring(0, 200) }); }

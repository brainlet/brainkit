import { Agent, Memory, UpstashStore } from "agent";
import { model, output } from "kit";
const url = process.env.UPSTASH_REDIS_REST_URL;
const token = process.env.UPSTASH_REDIS_REST_TOKEN;
if (!url || !token) throw new Error("UPSTASH credentials not set");
try {
  const store = new UpstashStore({ id: "agent-upstash", url, token });
  const mem = new Memory({ storage: store, options: { lastMessages: 10 } });
  const agent = new Agent({ name: "upstash-mem", model: model("openai", "gpt-4o-mini"), instructions: "Remember what the user tells you.", memory: mem });
  const tid = "upstash-" + Date.now();
  await agent.generate("I work at Brainlet.", { memory: { thread: { id: tid }, resource: "test" } });
  const r = await agent.generate("Where do I work?", { memory: { thread: { id: tid }, resource: "test" } });
  output({ text: r.text, remembers: r.text.toLowerCase().includes("brainlet"), backend: "upstash" });
} catch(e: any) { output({ error: e.message.substring(0, 200) }); }

// Test: agent memory with Upstash Redis storage (real cloud service)
// Requires UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN env vars.
import { Agent, UpstashStore } from "agent";
import { model, output } from "kit";

const url = globalThis.process?.env?.UPSTASH_REDIS_REST_URL;
const token = globalThis.process?.env?.UPSTASH_REDIS_REST_TOKEN;
if (!url || !token) throw new Error("UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN must be set");

const store = new UpstashStore({
  id: "test-upstash-store",
  url: url,
  token: token,
});

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: store as any,
});

// First call
await a.generate("My favorite language is Go.", {
  memory: { thread: { id: "upstash-test-" + Date.now() }, resource: "test-user" },
});

// Second call — should remember
const result = await a.generate("What is my favorite language?", {
  memory: { thread: { id: "upstash-test-" + Date.now() }, resource: "test-user" },
});

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("go"),
  store: "upstash",
});

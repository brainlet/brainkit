// Test: agent memory with Upstash Redis storage (real cloud service)
// Requires UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN env vars.
import { agent, UpstashStore, output } from "brainlet";

const url = globalThis.process?.env?.UPSTASH_REDIS_REST_URL;
const token = globalThis.process?.env?.UPSTASH_REDIS_REST_TOKEN;
if (!url || !token) throw new Error("UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN must be set");

const store = new UpstashStore({
  id: "test-upstash-store",
  url: url,
  token: token,
});

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant. Remember what the user tells you.",
  memory: {
    thread: "upstash-test-" + Date.now(),
    resource: "test-user",
    storage: store,
  },
});

// First call
await a.generate("My favorite language is Go.");

// Second call — should remember
const result = await a.generate("What is my favorite language?");

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("go"),
  store: "upstash",
});

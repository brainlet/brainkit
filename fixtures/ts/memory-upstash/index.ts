// Test: agent memory with Upstash Redis storage (real cloud service)
// Uses UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN from .env
import { Agent, Memory, UpstashStore } from "agent";
import { model, output } from "kit";

const url = process.env.UPSTASH_REDIS_REST_URL;
const token = process.env.UPSTASH_REDIS_REST_TOKEN;
if (!url || !token) throw new Error("UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN must be set");

const store = new UpstashStore({
  id: "test-upstash-store",
  url: url,
  token: token,
});

const threadId = "upstash-fixture-" + Math.floor(Math.random() * 1000000);

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

// First call
await a.generate("My favorite language is Go.", {
  memory: { thread: { id: threadId }, resource: "test-user" },
});

// Second call — same thread, should remember
const result = await a.generate("What is my favorite language?", {
  memory: { thread: { id: threadId }, resource: "test-user" },
});

output({
  text: result.text,
  remembers: result.text.toLowerCase().includes("go"),
  store: "upstash",
});

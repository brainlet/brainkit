import {
  DEFAULT_RESPONSE_CACHE_TTL_SECONDS,
  InMemoryServerCache,
  RequestContext,
  ResponseCache,
  RESPONSE_CACHE_CONTEXT_KEY,
  buildResponseCacheKey,
} from "agent";
import { output } from "kit";

const cache = new InMemoryServerCache({ maxSize: 8, ttlMs: 0 });
await cache.set("manual", { ok: true }, 0);
const manual = (await cache.get("manual")) as { ok?: boolean } | undefined;

await cache.listPush("items", "one");
await cache.listPush("items", "two");
const listLength = await cache.listLength("items");
const firstIncrement = await cache.increment("counter");
const secondIncrement = await cache.increment("counter");

const processor = new ResponseCache({
  cache,
  ttl: 60,
  scope: "fixture-scope",
  agentId: "fixture-agent",
});

const ctx = ResponseCache.context({ key: "fixture-key", bust: true });
const existing = new RequestContext();
const applyReturnedSameContext =
  ResponseCache.applyContext(existing, { scope: "tenant-a" }) === existing;

const keyInputs = {
  agentId: "fixture-agent",
  scope: "tenant-a",
  model: { provider: "openai", modelId: "gpt-test", specVersion: "v2" },
  prompt: [{ role: "user", content: [{ type: "text", text: "hello" }] }],
  stepNumber: 0,
};

const keyA = buildResponseCacheKey(keyInputs);
const keyB = buildResponseCacheKey(keyInputs);

output({
  processorId: processor.id,
  processorName: processor.name,
  cacheRoundTrip: manual?.ok === true,
  listLength,
  incremented: firstIncrement === 1 && secondIncrement === 2,
  contextIsRequestContext: ctx instanceof RequestContext,
  contextHasOptions: (ctx.get(RESPONSE_CACHE_CONTEXT_KEY) as any)?.key === "fixture-key",
  applyReturnedSameContext,
  deterministicKey: keyA === keyB,
  keyPrefix: keyA.startsWith("mastra:agent-response:fixture-agent:"),
  defaultTtl: DEFAULT_RESPONSE_CACHE_TTL_SECONDS,
});

// Test: observability — verify spans are persisted in storage with span hierarchy
import { agent, createTool, z, output } from "kit";

const addTool = createTool({
  id: "add",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Use the add tool to compute 10 + 32. Return ONLY the number.",
  tools: { add: addTool },
});

const result = await a.generate("What is 10 + 32?");

// Wait for async span lifecycle events to complete
await new Promise(r => setTimeout(r, 500));

// Query persisted spans from storage
const store = globalThis.__kit_internal_store;
const obsStore = await store.getStore("observability");
const trace = result.traceId ? await obsStore.getTrace({ traceId: result.traceId }) : null;

output({
  // Agent result
  text: result.text,
  hasAnswer: result.text.includes("42"),
  toolCalls: result.toolCalls?.length || 0,
  traceId: result.traceId,
  runId: result.runId,
  hasTraceId: typeof result.traceId === "string" && result.traceId.length === 32,
  usage: result.usage,
  hasUsage: result.usage?.totalTokens > 0,

  // Persisted span data
  hasTrace: trace !== null,
  spanCount: trace?.spans?.length || 0,
  spanTypes: trace?.spans?.map(s => s.type) || [],
  spanNames: trace?.spans?.map(s => s.name) || [],
  // Verify expected spans by name (type may be null in InMemory storage)
  hasAgentRun: trace?.spans?.some(s => s.name?.startsWith("agent run")) || false,
  hasModelGeneration: trace?.spans?.some(s => s.name?.startsWith("llm:")) || false,
  hasToolCall: trace?.spans?.some(s => s.name?.startsWith("tool:")) || false,
});

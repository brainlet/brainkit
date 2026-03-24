# Test Fixtures

These `.js` fixtures use the four-module import pattern:

```js
import { Agent, createTool, createWorkflow, createStep, Memory, z } from "agent";
import { generateText, streamText, generateObject, embed, embedMany, z } from "ai";
import { compile } from "compiler";
import { bus, kit, model, tools, mcp, output } from "kit";
```

## Module Reference

### `"kit"` — Infrastructure
`bus`, `kit`, `model`, `provider`, `storage`, `vectorStore`, `registry`, `tools`, `fs`, `mcp`, `output`, `wasm` (if present), `tool`

### `"agent"` — Mastra Framework
`Agent`, `createTool`, `createWorkflow`, `createStep`, `Memory`, `InMemoryStore`, `LibSQLStore`, `PostgresStore`, `MongoDBStore`, `UpstashStore`, `LibSQLVector`, `PgVector`, `MongoDBVector`, `Workspace`, `LocalFilesystem`, `LocalSandbox`, `MDocument`, `GraphRAG`, `createVectorQueryTool`, `createDocumentChunkerTool`, `createGraphRAGTool`, `rerank`, `rerankWithScorer`, `Observability`, `DefaultExporter`, `createScorer`, `runEvals`, `RequestContext`, `ModelRouterEmbeddingModel`, `z`

### `"ai"` — AI SDK
`generateText`, `streamText`, `generateObject`, `streamObject`, `embed`, `embedMany`, `z`

### `"compiler"` — WASM
`compile`

## Key Patterns

### Agent creation
```js
// Old: agent({ model: "openai/gpt-4o-mini", ... })
// New:
import { Agent } from "agent";
import { model } from "kit";

const a = new Agent({
  name: "my-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "...",
});
```

### Direct AI calls
```js
// Old: ai.generate({ model: "openai/gpt-4o-mini", prompt: "..." })
// New:
import { generateText } from "ai";
import { model } from "kit";

const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "...",
});
```

### Tool registration
```js
// Old: tools.register("my_tool", { ... })
// New:
import { createTool, z } from "agent";
import { kit } from "kit";

const myTool = createTool({ id: "my_tool", ... });
kit.register("tool", "my_tool", myTool);
```

### Memory
```js
// Old: agent({ memory: { thread: "x", resource: "y", storage: new InMemoryStore() } })
// New:
import { Agent, Memory, InMemoryStore } from "agent";

const memory = new Memory({ storage: new InMemoryStore(), options: { lastMessages: 10 } });
const a = new Agent({ name: "...", model: ..., memory: memory });
// Pass thread/resource per-call:
await a.generate("...", { memory: { thread: { id: "x" }, resource: "y" } });
```

## Removed APIs

The following old APIs no longer exist and fixtures that used them have been annotated with TODO comments:

- `processors` (UnicodeNormalizer, TokenLimiterProcessor, etc.) — see `processors-builtin.js`
- `createSubagent()` — replaced by `new Agent()` with `agents` config
- `scorers` (pre-built scorers) — only `createScorer()` remains
- `sandbox` context export — replaced by globalThis context
- `inputProcessors` / `outputProcessors` on agent — see `agent-with-processor.js`, `agent-with-tripwire.js`

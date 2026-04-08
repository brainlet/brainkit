# brainkit .ts Runtime — Complete API Reference

> For LLMs writing TypeScript code deployed into brainkit SES Compartments.
> Every type matches the real code in `kit/runtime/kit_runtime.js` and `kit/runtime/*.d.ts`.

## Execution Model

1. `.ts` files are transpiled (types stripped, runtime preserved)
2. ES `import` statements are stripped — symbols come from Compartment endowments
3. Code runs in a frozen SES Compartment — cannot modify globals
4. Each deployed file gets a mailbox namespace: `my-agent.ts` → `ts.my-agent`
5. Top-level `await` works — the entire file body is wrapped in async
6. `output(value)` sets the return value (passed back to Go)

## Four Modules

```typescript
import { generateText, streamText, generateObject, streamObject, embed, embedMany, z } from "ai";
import { Agent, createTool, createWorkflow, createStep, Memory, InMemoryStore, LibSQLStore, PostgresStore, MongoDBStore, UpstashStore, LibSQLVector, PgVector, MongoDBVector, Workspace, LocalFilesystem, LocalSandbox, MDocument, GraphRAG, createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool, rerank, rerankWithScorer, Observability, DefaultExporter, createScorer, runEvals, RequestContext, ModelRouterEmbeddingModel, z } from "agent";
import { bus, kit, model, embeddingModel, provider, storage, vectorStore, registry, tools, tool, fs, mcp, output, secrets, generateWithApproval } from "kit";
```

## "kit" Module

### bus

```typescript
const bus: {
    publish(topic: string, data?: unknown): { replyTo: string; correlationId: string };
    emit(topic: string, data?: unknown): void;
    subscribe(topic: string, handler: (msg: BusMessage) => void | Promise<void>): string;
    on(localTopic: string, handler: (msg: BusMessage) => void | Promise<void>): string;
    unsubscribe(subId: string): void;
    sendTo(service: string, localTopic: string, data?: unknown): { replyTo: string; correlationId: string };
    schedule(expression: string, topic: string, data?: unknown): string;
    unschedule(scheduleId: string): void;
};

interface BusMessage {
    payload: unknown;
    replyTo: string;
    correlationId: string;
    topic: string;
    callerId: string;
    reply(data: unknown): void;   // final response, done=true
    send(data: unknown): void;    // intermediate chunk, done=false
}
```

### kit

```typescript
const kit: {
    register(type: "agent" | "tool" | "workflow" | "memory", name: string, ref: unknown): void;
    unregister(type: string, name: string): void;
    list(type?: string): ResourceEntry[];
    readonly source: string;
    readonly namespace: string;
    readonly callerId: string;
};

interface ResourceEntry {
    type: string;
    id: string;
    name: string;
    source: string;
    createdAt: number;
}
```

### Model Resolution

```typescript
function model(provider: string, modelId: string): LanguageModel;
function embeddingModel(provider: string, modelId: string): EmbeddingModel;
function provider(name: string): ProviderFactory;
```

Providers: openai, anthropic, google, mistral, groq, deepseek, xai, cerebras, perplexity, togetherai, fireworks, cohere.

### Storage / Vector Resolution

```typescript
function storage(name: string): StorageInstance;     // from Config.Storages
function vectorStore(name: string): VectorStoreInstance; // from Config.Vectors
```

### Registry

```typescript
const registry: {
    has(category: "provider" | "vectorStore" | "storage", name: string): boolean;
    list(category: string): RegistryEntry[];
    resolve(category: string, name: string): { type: string; name: string; config: any } | null;
    register(category: string, name: string, config: Record<string, unknown>): void;
    unregister(category: string, name: string): void;
};
```

### Tools

```typescript
const tools: {
    call(name: string, input?: Record<string, unknown>): Promise<unknown>;
    list(namespace?: string): ToolInfo[];
    resolve(name: string): ToolResolveResult;
};

function tool(name: string): ToolObject; // resolves registered tool for Agent.tools

interface ToolInfo { name: string; shortName: string; namespace: string; description: string; }
interface ToolResolveResult { name: string; shortName: string; description: string; inputSchema?: any; }
```

### Filesystem

The `fs` endowment is a complete Node.js 22 `fs` polyfill (Go-backed via `jsbridge/fs.go`):

```typescript
// Async (Promise-based)
const data = await fs.promises.readFile("file.txt", "utf-8");
await fs.promises.writeFile("output.txt", data);
await fs.promises.mkdir("dir", { recursive: true });
const entries = await fs.promises.readdir(".");
const stats = await fs.promises.stat("file.txt");

// Sync
const content = fs.readFileSync("file.txt", "utf-8");
fs.writeFileSync("output.txt", content);
fs.mkdirSync("dir", { recursive: true });

// File handles
const fh = await fs.promises.open("file.txt", "r");
const { bytesRead, buffer } = await fh.read(buf, 0, buf.length, 0);
await fh.close();
```

All paths resolve relative to `Config.FSRoot` with workspace escape protection.

### MCP

```typescript
const mcp: {
    listTools(server?: string): McpToolInfo[];
    callTool(server: string, tool: string, args?: Record<string, unknown>): Promise<unknown>;
};
```

### Output

```typescript
function output(value: unknown): void;
```

### HITL

```typescript
function generateWithApproval(
    agent: Agent,
    promptOrMessages: string | Message[],
    options: {
        approvalTopic: string;      // required
        timeout?: number;           // ms, default 30000
        [key: string]: unknown;     // other options passed to agent.generate
    }
): Promise<AgentResult>;
```

### Secrets

```typescript
const secrets: {
    get(name: string): string;  // returns "" if not found
};
```

## SES Compartment Globals

| Category | Globals |
|----------|---------|
| Network | `fetch`, `Headers`, `Request`, `Response` |
| URL | `URL`, `URLSearchParams` |
| Crypto | `crypto` (WebCrypto `subtle` + Node.js `createHash`, `pbkdf2Sync`) |
| Encoding | `TextEncoder`, `TextDecoder`, `btoa`, `atob` |
| Streams | `ReadableStream`, `WritableStream`, `TransformStream` |
| Buffer | `Buffer` (Node.js: `from`, `alloc`, `concat`, `toString`) |
| Timers | `setTimeout`, `clearTimeout`, `setInterval`, `clearInterval` |
| Events | `EventEmitter` (Node.js), `EventTarget`, `Event`, `CustomEvent` |
| Abort | `AbortController`, `AbortSignal` |
| Core | `JSON`, `Promise`, `Date` (BrainkitDate), `Math` (with random), `console` (tagged) |
| Process | `process.env` (Go-backed Proxy) |
| Node.js | `stream`, `crypto`, `net`, `os`, `dns`, `zlib`, `child_process`, `Buffer`, `GoSocket` |
| Misc | `structuredClone`, `queueMicrotask` |

### NOT available

`eval()`, `new Function()`, `globalThis` mutation, `require()`, dynamic `import()`, raw TCP/UDP.

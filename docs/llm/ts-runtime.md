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
import { bus, kit, model, embeddingModel, provider, storage, vectorStore, registry, tools, tool, fs, mcp, output, generateWithApproval } from "kit";
import { compile } from "compiler";
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
    sendToShard(shard: string, topic: string, data?: unknown): { replyTo: string; correlationId: string };
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
function storage(name: string): StorageInstance;     // from KernelConfig.Storages
function vectorStore(name: string): VectorStoreInstance; // from KernelConfig.Vectors
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

```typescript
const fs: {
    read(path: string): Promise<{ data: string }>;
    write(path: string, data: string): Promise<{ ok: boolean }>;
    list(path?: string, pattern?: string): Promise<{ files: FileInfo[] }>;
    stat(path: string): Promise<{ size: number; isDir: boolean; modTime: string }>;
    delete(path: string): Promise<{ ok: boolean }>;
    mkdir(path: string): Promise<{ ok: boolean }>;
};
```

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

### Compiler

```typescript
async function compile(source: string, opts?: { name?: string }): Promise<CompileResult>;

interface CompileResult {
    moduleId: string;
    name: string;
    size: number;
    exports: string[];
    run(input?: any): Promise<{ exitCode: number; value?: any }>;
}
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

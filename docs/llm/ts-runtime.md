# TypeScript Runtime — API Reference

Dense reference for the `.ts` deployment surface in brainkit: the SES Compartment endowments installed on every `kit.Deploy(...)` source. Canonical source: `internal/engine/runtime/kit_runtime.js`, `internal/engine/runtime/bus.js`, `internal/engine/runtime/infrastructure.js`, `internal/engine/runtime/resolve.js`, `internal/engine/runtime/approval.js`, and the `.d.ts` bundle shipped at `internal/engine/runtime/kit.d.ts`. Runtime version: v1.0.0-rc.1.

---

## 1. Execution model

1. A `.ts` file reaches the runtime via `kit.Deploy(ctx, brainkit.Package*)` (Go) or `bus.call("package.deploy", …)` (JS).
2. TypeScript is transpiled (types stripped, runtime JS preserved) by esbuild.
3. ES `import` statements are stripped. All symbols arrive through Compartment endowments.
4. The deployment runs inside a frozen SES Compartment. `globalThis` is the per-deployment endowment map; pre-lockdown (`Date.now`, `Math.random`) is preserved behind the SES taming.
5. The deployment's mailbox is `ts.<source>.<topic>`, where `<source>` is the file path with `.ts` stripped and `/` → `.`. Subscriptions via `bus.on(localTopic, …)` are scoped here; external callers reach them with `bus.publish("ts.<source>.<topic>", …)` or `bus.sendTo("<source>", "<topic>", …)`.

A typical deployment:

```typescript
import { Agent } from "agent"
import { model, bus, kit, z, createTool } from "kit"

const tools = {
    greet: createTool({
        id: "greet",
        description: "Greet someone",
        inputSchema: z.object({ name: z.string() }),
        execute: async ({ context }) => `hello ${context.name}`,
    }),
}

const agent = new Agent({
    name: "greeter",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Call greet when a name is supplied.",
    tools,
})

kit.register("agent", "greeter", agent)

bus.on("ask", async (msg) => {
    const result = await agent.generate(msg.payload?.prompt ?? "")
    msg.reply({ text: result.text, usage: result.usage })
})
```

Import statements are only there for the TypeScript author's tooling. At runtime every identifier resolves through the endowment map below.

---

## 2. Endowment map

Every Compartment receives the following symbols on `globalThis`:

| Symbol | Provider file | Description |
|--------|---------------|-------------|
| `BrainkitError` | `kit_runtime.js` | Typed error class (§3) |
| `bus` | `bus.js` | Pub/sub API (§4) |
| `kit` | `bus.js` | Deployment identity + resource registry (§5) |
| `model` | `resolve.js` | `(provider, model) => LanguageModel` (§6) |
| `embeddingModel` | `resolve.js` | `(provider, model) => EmbeddingModel` (§6) |
| `provider` | `resolve.js` | Named AI provider accessor (§6) |
| `storage` | `resolve.js` | Named storage accessor (§7) |
| `vectorStore` | `resolve.js` | Named vector store accessor (§7) |
| `registry` | `infrastructure.js` | Named-resource registry (§8) |
| `tools` | `infrastructure.js` | Kit tool registry (§9) |
| `tool` | `kit_runtime.js` | Resolve a tool as an AI SDK tool by name (§9) |
| `fs` | `internal/jsbridge/fs.go` | Node.js-compatible `fs` API (§10) |
| `mcp` | `infrastructure.js` | MCP tool accessors (§11) |
| `output` | `infrastructure.js` | Capture the module's return value (§12) |
| `secrets` | `infrastructure.js` | `secrets.get(name)` → decrypted value (§13) |
| `generateWithApproval` | `approval.js` | HITL wrapper over `agent.generate` (§14) |
| AI SDK: `generateText`, `streamText`, `generateObject`, `streamObject`, `embed`, `embedMany`, `z` | `kit_runtime.js` | See `ai-sdk.md` |
| Mastra: `Agent`, `createTool`, `createWorkflow`, `createStep`, `Memory`, `InMemoryStore`, `LibSQLStore`, `UpstashStore`, `PostgresStore`, `MongoDBStore`, `LibSQLVector`, `PgVector`, `MongoDBVector`, `ModelRouterEmbeddingModel`, `RequestContext`, `Workspace`, `LocalFilesystem`, `LocalSandbox`, `MDocument`, `GraphRAG`, `createVectorQueryTool`, `createDocumentChunkerTool`, `createGraphRAGTool`, `rerank`, `rerankWithScorer`, `Observability`, `DefaultExporter`, `createScorer`, `runEvals` | `kit_runtime.js` | See `mastra.md` |
| Web APIs: `fetch`, `Request`, `Response`, `Headers`, `URL`, `URLSearchParams`, `AbortController`, `AbortSignal`, `TextEncoder`, `TextDecoder`, `ReadableStream`, `WritableStream`, `TransformStream`, `TextEncoderStream`, `TextDecoderStream`, `atob`, `btoa`, `crypto`, `structuredClone` | polyfills | Standard browser surface |
| Scheduling: `setTimeout`, `setInterval`, `clearTimeout`, `clearInterval`, `queueMicrotask`, `Promise` | host | Wrapped with source-tracking |
| Serialization: `JSON`, `Date`, `Math` | SES-tamed | `Date.now` and `Math.random` preserved from pre-lockdown |
| Node.js compat: `Buffer`, `process`, `EventEmitter`, `stream`, `net`, `os`, `dns`, `zlib`, `child_process`, `GoSocket` | polyfills | Sufficient for most npm libraries |
| Console: `console.log/warn/error/info/debug` | `kit_runtime.js` | Tagged — routed to `LogHandler`  |

Imports from `"kit"`, `"ai"`, and `"agent"` in `.ts` source are stripped at deploy time — the runtime injects the symbols directly. Identifiers not listed above throw `ReferenceError`.

---

## 3. `BrainkitError`

```typescript
class BrainkitError extends Error {
    name: "BrainkitError"
    code: string           // stable machine-readable code; matches the Go sdkerrors set
    details?: object       // structured context

    constructor(message: string, code: string, details?: object)
}
```

Error codes observable from `.ts`:

| Code | Meaning |
|------|---------|
| `VALIDATION_ERROR` | Input schema / option failure (also thrown by `bus.call` / `bus.callTo` / `bus.onCancel` parameter validation) |
| `NOT_FOUND` | Named resource missing |
| `TIMEOUT` | `bus.call` / remote handler exceeded its deadline |
| `HANDLER_FAILED` | Remote handler threw |
| `TRANSPORT_ERROR` | Bus publish / subscribe failure |
| `COMPARTMENT_ERROR` | SES lockdown violation |
| `TOPIC_COLLISION` | `bus.on(localTopic, …)` called twice with the same `localTopic` in one package |
| `NOT_CONFIGURED` | Feature (secrets, storage, vectorStore, MCP) has no configuration |
| `PLUGIN_UNHEALTHY` / `PLUGIN_TIMEOUT` | Plugin subprocess failed |

`bus.publish`, `bus.emit`, `bus.sendTo`, `bus.call`, `bus.callTo`, `bus.schedule`, `tools.call`, `tools.list`, `tools.resolve`, and `secrets.get` automatically rewrap bridge errors carrying a `.code` into `BrainkitError` so `instanceof BrainkitError` works across the bridge.

---

## 4. `bus`

```typescript
const bus: {
    publish(topic: string, data?: unknown): { replyTo: string; correlationId: string }
    emit(topic: string, data?: unknown): void
    subscribe(topic: string, handler: (msg: BusMessage) => void | Promise<void>): string
    on(localTopic: string, handler: (msg: BusMessage) => void | Promise<void>): string
    unsubscribe(subId: string): void
    sendTo(service: string, topic: string, data?: unknown): { replyTo: string; correlationId: string }
    call<T = any>(topic: string, data?: unknown, opts: { timeoutMs: number }): Promise<T>
    callTo<T = any>(namespace: string, topic: string, data?: unknown, opts: { timeoutMs: number }): Promise<T>
    schedule(expression: string, localTopic: string, data?: unknown): string
    unschedule(scheduleId: string): void
    onCancel(correlationId: string, handler: (evt: any) => void): () => void
    withCancelController(msg: BusMessage): { signal: AbortSignal; cleanup: () => void }
}
```

### 4.1 `publish` vs `emit` vs `call`

- `publish` sends a request. Mastra-style handlers on the target topic reply with `msg.reply(data)`; the response is routed back through an auto-generated `replyTo`.
- `emit` is fire-and-forget — no `replyTo` is attached.
- `call` does `publish` + wait for the terminal envelope and returns the decoded response. **`timeoutMs` is required** (mirrors Go's deadline rule). Throws `BrainkitError("TIMEOUT")` on expiry or rewraps an ok=false reply.
- `callTo(namespace, topic, data, opts)` is `call` across namespaces. Both `namespace` and `opts.timeoutMs` are required.

```typescript
// Request/reply
const resp = await bus.call("ts.analytics.summary", { range: "7d" }, { timeoutMs: 5000 })

// Fire-and-forget
bus.emit("metrics.log", { name: "deploy", value: 1 })

// Cross-namespace
const users = await bus.callTo("auth", "users.list", {}, { timeoutMs: 2000 })
```

### 4.2 `subscribe`, `on`, `unsubscribe`

- `subscribe(topic, handler)` binds to any absolute topic. Returns a subscription id — pass it to `unsubscribe` or let the runtime auto-clean up on teardown.
- `on(localTopic, handler)` binds inside the deployment's mailbox (`ts.<source>.<localTopic>`). Throws `BrainkitError("TOPIC_COLLISION")` if the same `localTopic` is already subscribed in the same package.

Inside the handler, `msg` is the `BusMessage` described in §4.5.

### 4.3 `sendTo`

```typescript
bus.sendTo("haiku-bot", "ask", { prompt: "autumn" })
// → publishes to "ts.haiku-bot.ask"
```

`service` is the deployment name (with or without `.ts` suffix); `topic` is the local topic on that deployment. Returns the same `{ replyTo, correlationId }` shape as `publish`.

### 4.4 `schedule` / `unschedule`

```typescript
const id = bus.schedule("*/5 * * * *", "tick", { hello: "world" })
// Fires "ts.<source>.tick" every 5 minutes.
bus.unschedule(id)
```

Accepts cron expressions or `"every 30s"` style intervals. The target topic is scoped to the current deployment's mailbox — pass the local topic, not the fully qualified name. Requires the `schedules` module on the Kit.

### 4.5 `BusMessage`

```typescript
interface BusMessage {
    payload: any
    replyTo: string
    correlationId: string
    topic: string
    callerId: string

    // Terminal reply — wire envelope { ok: true, data }, done=true.
    reply(data: any): void
    // Intermediate raw chunk — done=false, no envelope wrapping.
    send(data: any): void

    // Typed streaming helpers — done=false except end/error.
    stream: {
        text(chunk: string): void                                // { type: "text", seq, data }
        progress(value: number, message?: string): void          // { type: "progress", seq, data: { value, message } }
        object(partial: any): void                               // { type: "object", seq, data }
        event(name: string, data?: any): void                    // { type: "event", seq, event: name, data }
        error(message: string): void                             // { type: "error", total, data }, done=true
        end(finalData?: any): void                               // { type: "end", total, data }, done=true
    }
}
```

`seq` is a per-message monotonic counter. `stream.end` and `stream.error` are terminal and increment `total` instead of `seq`. The SSE gateway consumes this shape directly.

### 4.6 `onCancel` and `withCancelController`

```typescript
bus.on("expensive", async (msg) => {
    const { signal, cleanup } = bus.withCancelController(msg)
    try {
        const res = await fetch(url, { signal })
        msg.reply(await res.json())
    } finally {
        cleanup()
    }
})
```

- `bus.onCancel(correlationId, handler)` returns an `unsubscribe` function; `handler` fires when the upstream caller cancels.
- `bus.withCancelController(msg)` builds an `AbortController` wired to `msg.correlationId`. Always call `cleanup()` before returning so the cancel subscription is torn down.

### 4.7 Parameters validated as `VALIDATION_ERROR`

| Call | Required |
|------|----------|
| `bus.call(topic, data, opts)` | `opts.timeoutMs: number` |
| `bus.callTo(namespace, topic, data, opts)` | `namespace: string`, `opts.timeoutMs: number` |
| `bus.onCancel(correlationId, handler)` | `correlationId: string`, `handler: function` |
| `LibSQLStore({ url })` | `url` must not start with `file:` — use `storage("name")` (§7) |
| `LibSQLVector({ connectionUrl })` | `connectionUrl` must not start with `file:` — use `vectorStore("name")` (§7) |

---

## 5. `kit`

```typescript
const kit: {
    register(type: "tool" | "agent" | "workflow" | "memory", name: string, ref: unknown): void
    unregister(type: string, name: string): void
    list(type?: string): ResourceEntry[]
    readonly source: string     // e.g. "agents.ts"
    readonly namespace: string  // e.g. "user"
    readonly callerId: string   // e.g. "user"
}

interface ResourceEntry {
    type: string
    id: string
    name: string
    source: string
}
```

`kit.register` is the **only** way to surface a resource for discovery. The four valid types:

| Type | Meaning | Cross-surface visibility |
|------|---------|--------------------------|
| `"tool"` | Something callable through `tools.call` / MCP | Emitted as a tool in `tools.list`, reachable from the MCP surface |
| `"agent"` | Mastra `Agent` instance | Emitted to `agents.list`, can be invoked via the agent dispatcher |
| `"workflow"` | Mastra `createWorkflow` result | Emitted to `workflow.list`, can be started via `workflow.start` |
| `"memory"` | Mastra `Memory` instance | Labeled memory, resolvable by name |

Invalid types throw `Error("kit.register: invalid type '<x>' ...")`; re-registering an existing `{type, name}` from the same source is a no-op. Unregister attempts across source boundaries throw `Error("kit.unregister: cannot unregister <type> '<name>' owned by <other-source>")`.

`kit.namespace` and `kit.callerId` resolve to the Kit-level values; `kit.source` resolves to the current deployment file.

---

## 6. `model`, `embeddingModel`, `provider`

```typescript
model(providerName: string, modelID: string): LanguageModel
embeddingModel(providerName: string, modelID: string): EmbeddingModel
provider(registeredName: string): AIProviderFactory
```

`providerName` is the constructor key from `brainkit.Config.Providers` (e.g. `"openai"`, `"anthropic"`, `"google"`, `"mistral"`, `"xai"`, `"groq"`, `"deepseek"`, `"cerebras"`, `"perplexity"`, `"togetherai"`, `"fireworks"`, `"cohere"`).

- `model` returns an AI SDK v5 `LanguageModel`. If the provider is unknown or has no `create*` factory, it falls back to the literal string `"<provider>/<model>"` so the runtime can still register a plausible identifier.
- `embeddingModel` requires a provider that supports embeddings. Throws `Error("embeddingModel: provider '<name>' not registered")` or `"... does not support embeddings"` when the factory lacks `embedding` / `textEmbeddingModel`.
- `provider(name)` resolves a **registered** provider (set via `kit.Providers().Register` or `Config.Providers`) and returns the cached AI SDK factory; useful when you need the raw factory to call e.g. `openai.chat("gpt-4o")`.

```typescript
const gpt = model("openai", "gpt-4o-mini")
const embedder = embeddingModel("openai", "text-embedding-3-small")
const openai = provider("openai")
const custom = openai.chat("gpt-4o")
```

---

## 7. `storage` / `vectorStore`

Named lookups into the Kit's `Config.Storages` / `Config.Vectors` pool:

```typescript
storage(name: string): Store           // InMemoryStore | LibSQLStore | PostgresStore | MongoDBStore | UpstashStore
vectorStore(name: string): VectorStore // LibSQLVector | PgVector | MongoDBVector
```

Backed by `__go_registry_resolve("storage"|"vectorStore", name)`. The returned object is an already-instantiated Mastra storage / vector adapter — the same classes exposed as endowments (`LibSQLStore`, `PgVector`, …) but prebuilt from the Go-side configuration and cached per-Kit.

Throws:

- `Error("storage '<name>' not registered")` / `Error("vector store '<name>' not registered")` when the name is absent.
- `Error("storage type '<t>' not available")` / `Error("vector store type '<t>' not available")` if the runtime is missing the adapter.

**Ground rule**: `.ts` code must **not** construct `LibSQLStore` / `LibSQLVector` with `file:` URLs directly. The endowment wrapper throws `BrainkitError("VALIDATION_ERROR")` asking you to use `storage("name")` / `vectorStore("name")` instead.

```typescript
const main = storage("main")          // Config.Storages["main"]
const docs = vectorStore("docs")      // Config.Vectors["docs"]
await main.createTable({ tableName: "events", schema: { id: "text" } })
```

---

## 8. `registry`

Low-level access to the resource registry for categories other than the four `kit.register` types:

```typescript
const registry: {
    has(category: string, name: string): boolean
    list(category: string): any[]
    resolve(category: string, name: string): any | null
    register(category: string, name: string, config: unknown): void
    unregister(category: string, name: string): void
}
```

Typical categories: `"provider"`, `"storage"`, `"vectorStore"`. Configurations are JSON-serializable; the Go side owns type validation.

---

## 9. `tools`, `tool`

```typescript
const tools: {
    call<T = any>(name: string, input?: unknown): Promise<T>
    list(namespace?: string): Array<{ name: string; description: string; inputSchema: any }>
    resolve(name: string): {
        name: string
        shortName?: string
        description?: string
        inputSchema?: any
    } | null
}

function tool(name: string): AiSdkTool
```

- `tools.call` invokes a registered tool (from `kit.register("tool", …)` or MCP / plugin exposure). The return is the tool's output value.
- `tools.list(namespace)` returns all tools; pass a namespace to filter.
- `tools.resolve(name)` returns the raw registration record or `null`.
- `tool(name)` wraps a registered tool as an AI SDK v5 `tool({...})` value you can pass directly into `generateText`, `streamText`, or an `Agent`'s `tools` field. Internally calls `createTool({ id, description, inputSchema: z.object(schema) || z.any(), execute: (input) => tools.call(name, input) })`.

```typescript
const summary = await tools.call("search.web", { q: "brainkit" })
// Use another Kit tool inside an agent
const agent = new Agent({
    name: "researcher",
    model: model("openai", "gpt-4o-mini"),
    tools: { web: tool("search.web") },
})
```

---

## 10. `fs` (Node.js API shape)

The `fs` endowment mirrors Node.js `fs` (sync + `fs.promises`) enough to satisfy npm libraries bundled into the deployment. All I/O is sandboxed to `brainkit.Config.FSRoot`.

### Sync methods

`readFileSync`, `writeFileSync`, `appendFileSync`, `readdirSync`, `statSync`, `lstatSync`, `accessSync`, `mkdirSync`, `mkdtempSync`, `rmdirSync`, `rmSync`, `unlinkSync`, `renameSync`, `copyFileSync`, `cpSync`, `linkSync`, `symlinkSync`, `readlinkSync`, `realpathSync`, `chmodSync`, `chownSync`, `truncateSync`, `utimesSync`, `existsSync`.

### Streams

```typescript
fs.createReadStream(path: string, opts?: ReadStreamOpts): stream.Readable
fs.createWriteStream(path: string, opts?: WriteStreamOpts): stream.Writable
```

Both return Node.js-shaped stream objects; events (`open`, `ready`, `data`, `end`, `error`, `close`) fire in the expected order. The `path` property is preserved.

### Async namespace

`fs.promises` exposes the standard async API: `readFile`, `writeFile`, `appendFile`, `readdir`, `stat`, `lstat`, `mkdir`, `rm`, `rmdir`, `unlink`, `rename`, `copyFile`, `access`, `chmod`, `chown`, `truncate`, `utimes`, `readlink`, `realpath`, `open` (FileHandle).

### Typical usage

```typescript
const raw = fs.readFileSync("data/events.json", "utf8")
const events = JSON.parse(raw)

await fs.promises.mkdir("out", { recursive: true })
await fs.promises.writeFile("out/summary.md", report, "utf8")
```

Paths are resolved relative to `FSRoot`. Escape attempts (`..` outside the sandbox) are rejected by the Go-side path guard.

---

## 11. `mcp`

```typescript
const mcp: {
    listTools(server?: string): Array<{ name: string; server: string; description: string }>
    callTool(server: string, tool: string, args?: unknown): Promise<unknown>
}
```

Backed by the `mcp` module; `.ts` code can enumerate and invoke tools from connected MCP servers without going through the bus. Unconfigured servers throw the underlying transport error.

---

## 12. `output`

```typescript
output(value: any): void
```

Captures the module's final value. The runtime's `Eval` path returns either a string (when `value` is a string) or the JSON encoding of `value`. Useful when `.ts` code is loaded for a one-shot computation instead of registering subscriptions.

---

## 13. `secrets`

```typescript
const secrets: {
    get(name: string): string
}
```

Returns the cleartext secret string from the Kit's `SecretStore` (or empty string if the store is absent). `set` / `list` / `delete` are intentionally **not** exposed to `.ts` — mutate secrets from Go or via the `secrets.*` bus commands.

---

## 14. `generateWithApproval`

```typescript
generateWithApproval(
    agent: Agent,
    promptOrMessages: string | ChatMessage[],
    options: {
        approvalTopic: string           // required
        timeout?: number                // ms; default 30000
    } & GenerateOptions,
): Promise<AgentResult>
```

HITL wrapper over `agent.generate`. Runs the agent; if the model decides to call a tool that requires approval, the runtime publishes the tool-call metadata to `approvalTopic` and awaits a reply `{ approved: boolean }`. When approved, `agent.approveToolCallGenerate` resumes; otherwise `declineToolCallGenerate` runs.

```typescript
const result = await generateWithApproval(agent, "delete /tmp/data", {
    approvalTopic: "approvals.ops",
    timeout: 15000,
})
```

Any Mastra `.generate` option (`maxSteps`, `experimental_output`, …) is forwarded unchanged.

---

## 15. Tamed globals

### Date and Math

```typescript
Date        // real Date class; Date.now returns live millisecond timestamp
Math        // real Math object; Math.random returns live entropy
```

SES lockdown replaces `Date.now` and `Math.random` with deterministic stubs; the runtime restores the live implementations before the Compartment is created. Everything else (`Math.sin`, `Date.parse`, prototypes) is identical.

### Setters

`setTimeout`, `setInterval` are wrapped with `__withSource` so their callbacks carry the correct deployment source for logging and cleanup. `clearTimeout`, `clearInterval`, `queueMicrotask` are unwrapped.

---

## 16. Console

```typescript
console: {
    log(...args: any[]): void
    info(...args: any[]): void
    warn(...args: any[]): void
    error(...args: any[]): void
    debug(...args: any[]): void
}
```

Every log call is routed through `__go_console_log_tagged(source, level, formatted)`, which delivers a `LogEntry{Source, Level, Message, Time}` to the Kit's `LogHandler` (default: slog).

---

## 17. Web APIs

Available as endowments:

| API | Notes |
|-----|-------|
| `fetch`, `Request`, `Response`, `Headers` | WHATWG fetch; respects `AbortSignal`. |
| `URL`, `URLSearchParams` | Standard. |
| `AbortController`, `AbortSignal` | Wire into `bus.withCancelController(msg)` for cancelable handlers. |
| `TextEncoder`, `TextDecoder` | UTF-8. |
| `ReadableStream`, `WritableStream`, `TransformStream`, `TextEncoderStream`, `TextDecoderStream` | WHATWG streams. |
| `crypto` | WebCrypto `subtle` plus Node.js `createHash`, `pbkdf2Sync`. Merged into one object. |
| `atob`, `btoa`, `structuredClone` | Standard. |

---

## 18. Node.js compatibility

Polyfills installed on `globalThis`:

| Name | Coverage |
|------|----------|
| `Buffer` | Most npm consumers of `Buffer.from`, `Buffer.alloc`, `Buffer.concat`, buffer instance methods. |
| `process` | `process.env`, `process.hrtime`, `process.nextTick`, a minimal `process.versions`. |
| `EventEmitter` | Node.js v22 shape. |
| `stream` | `Readable`, `Writable`, `Duplex`, `Transform`, `PassThrough`, `pipeline`, `finished`. |
| `net` | Minimal TCP surface; `Socket`, `Server`, `createConnection`. |
| `os` | `platform`, `arch`, `homedir`, `tmpdir`, `hostname`, `cpus`, `totalmem`, `freemem`, `EOL`. |
| `dns` | `lookup`, `resolve*`, `promises.*`. |
| `zlib` | `deflate*`, `inflate*`, `gzip*`, `brotli*` (sync + stream). |
| `child_process` | Minimal — useful for specific plugins; inspect before relying on it. |
| `GoSocket` | Raw socket bridge exposed for advanced use. |

Polyfill implementations live in `internal/jsbridge/*.go`; add new Node APIs there rather than in the esbuild stubs.

---

## 19. Minimal deployment patterns

### 19.1 Register an agent + a topic

```typescript
import { Agent } from "agent"
import { model, bus, kit } from "kit"

const a = new Agent({
    name: "summarizer",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Summarize the input in one sentence.",
})
kit.register("agent", "summarizer", a)

bus.on("summarize", async (msg) => {
    const r = await a.generate(msg.payload?.text ?? "")
    msg.reply({ text: r.text, usage: r.usage })
})
```

### 19.2 Call another deployment

```typescript
bus.on("run", async (msg) => {
    const summary = await bus.call(
        "ts.summarizer.summarize",
        { text: msg.payload.text },
        { timeoutMs: 15000 },
    )
    msg.reply(summary)
})
```

### 19.3 Streaming text

```typescript
bus.on("chat", async (msg) => {
    const s = await streamText({ model: model("openai", "gpt-4o-mini"), prompt: msg.payload.prompt })
    for await (const chunk of s.textStream) msg.stream.text(chunk)
    msg.stream.end({ finishReason: s.finishReason })
})
```

### 19.4 Cancellation-aware handler

```typescript
bus.on("fetch", async (msg) => {
    const { signal, cleanup } = bus.withCancelController(msg)
    try {
        const r = await fetch(msg.payload.url, { signal })
        msg.reply({ status: r.status, body: await r.text() })
    } finally {
        cleanup()
    }
})
```

### 19.5 Resource deployment

```typescript
bus.on("bootstrap", async (msg) => {
    const id = bus.schedule("*/1 * * * *", "tick", { kind: "heartbeat" })
    msg.reply({ scheduleId: id, registered: kit.list("agent") })
})
```

---

## 20. Failure semantics

| Situation | Observable |
|-----------|------------|
| Throw inside `bus.on` handler (sync or async) | BrainkitError propagated to the caller; `HANDLER_FAILED` surfaced to replyTo (envelope) if a replyTo was attached. `bus.handler.failed` event emitted. |
| Unhandled exception during module evaluation | Deploy returns an error; module is not registered; resources registered before the throw are unwound via Go TeardownFile. |
| `bus.on(localTopic, …)` reused in the same package | Throws `BrainkitError("TOPIC_COLLISION")`. |
| `kit.register(type, …)` with invalid type | Throws `Error("kit.register: invalid type ...")`. |
| Teardown (`kit.Teardown(name)`) | Go sweeps subscriptions, schedules, registered resources; cleanup callbacks run. JS refs map is cleared. |

The runtime tracks every resource registered during evaluation. On teardown or redeploy the Go side calls the registered cleanup functions in LIFO order so `bus.subscribe`, `bus.schedule`, and `kit.register` do not leak across generations.

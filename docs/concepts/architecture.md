# Architecture

brainkit is a Go runtime that embeds five subsystems into one process: QuickJS (JavaScript), SES (sandboxing), Watermill (pub/sub messaging), wazero (WebAssembly), and typescript-go (TS transpilation). The result is a platform where Go, TypeScript, and WASM all communicate over a shared message bus.

## Why This Architecture

The traditional approach to AI agent platforms is a microservice mesh: separate processes for the agent runtime, tool execution, workflow engine, message broker, and storage. This works but introduces latency, deployment complexity, and failure modes at every boundary.

brainkit takes the opposite approach: everything runs in one OS process. The JS runtime (QuickJS) is embedded in Go via CGo. The WASM runtime (wazero) is pure Go. The message bus (Watermill) routes messages in-process by default, with optional external transports (NATS, Redis, AMQP, Postgres, SQLite) for multi-node deployments. TypeScript transpilation happens natively in Go via a vendored microsoft/typescript-go.

This means a tool call from an AI agent doesn't leave the process. A `.ts` service publishing to the bus doesn't cross a network boundary. A WASM shard reading state doesn't make an HTTP request. Everything is function calls and channels until you explicitly opt into distributed communication via a Node with an external transport.

## Two Runtime Types

### Kernel

The Kernel is the local runtime. It owns all state:

- One QuickJS runtime with SES lockdown, Mastra bundle, and 20+ Node.js polyfills
- One tool registry shared across all surfaces (Go, TS, WASM, plugins)
- One Watermill router processing command and event messages
- One WASM service managing compiled modules and deployed shards
- Zero or more deployed `.ts` files, each in its own SES Compartment
- Zero or more embedded SQLite storage bridges (libsql HTTP servers)

A standalone Kernel uses an in-process GoChannel transport — messages never leave the process. This is the default and the fastest configuration.

```go
// AI providers auto-detected from os.Getenv (e.g. OPENAI_API_KEY)
k, err := kit.NewKernel(kit.KernelConfig{
    Namespace: "my-kit",
    FSRoot:    "/tmp/workspace",
})
defer k.Close()
```

### Node

A Node wraps a Kernel with an external transport. It adds:

- Plugin subprocess management (launch, READY handshake, auto-restart, SIGTERM shutdown)
- Plugin state persistence (in-memory or NATS KV)
- Node-specific command bindings (plugin.manifest, plugin.state.get/set)

The critical detail: Node creates the external transport, then injects it into `KernelConfig.Transport` with `DeferRouterStart = true`. This means the Kernel uses the external transport instead of creating its own GoChannel, and Node gets to register its plugin-specific command bindings before the router starts processing messages.

```go
n, err := kit.NewNode(kit.NodeConfig{
    Kernel: kit.KernelConfig{
        Namespace: "my-kit",
        FSRoot:    "/tmp/workspace",
    },
    Messaging: kit.MessagingConfig{
        Transport: "nats",
        NATSURL:   "nats://localhost:4222",
    },
    Plugins: []kit.PluginConfig{
        {Name: "my-plugin", Binary: "./my-plugin", AutoRestart: true},
    },
})
n.Start(ctx) // restores WASM subscriptions, launches plugins
defer n.Close()
```

Both Kernel and Node implement `sdk.Runtime` (PublishRaw, SubscribeRaw, Close) and `sdk.CrossNamespaceRuntime` (PublishRawTo, SubscribeRawTo) and `sdk.Replier` (ReplyRaw). Any code that takes an `sdk.Runtime` works with either.

## Initialization Order

NewKernel initializes subsystems in a strict order. Each step depends on the previous ones. The order matters — reordering causes failures that are hard to debug because the dependencies are implicit.

**Step 1 — QuickJS sandbox.** `agentembed.NewSandbox` creates a `jsbridge.Bridge` (QuickJS runtime with mutex-serialized eval), loads 24 polyfills in dependency order (Events before NodeStreams before Buffer before Net — because Socket extends Duplex which extends EventEmitter), then loads SES (polyfills → ses.umd.js → lockdown → Mastra bundle). After this step, `globalThis.__agent_embed` exists with Agent, createTool, createWorkflow, the AI SDK, and all Mastra exports. See [jsbridge-polyfills.md](jsbridge-polyfills.md) and [bundle-and-bytecode.md](bundle-and-bytecode.md).

**Step 2 — Domain handlers.** Creates ToolsDomain, AgentsDomain, FSDomain. These are thin wrappers that route typed messages to the underlying registries and JS runtime. They exist so the command catalog can dispatch to typed handlers.

**Step 3 — Go↔JS bridges.** `registerBridges()` sets up 12 functions on `globalThis`:
- `__go_brainkit_request` / `__go_brainkit_request_async` — synchronous/async command invocation via the LocalInvoker
- `__go_brainkit_control` — local-only operations (tools.register, agents.register, registry.register/unregister)
- `__go_brainkit_bus_publish` / `__go_brainkit_bus_emit` / `__go_brainkit_bus_reply` — bus operations
- `__go_brainkit_subscribe` / `__go_brainkit_unsubscribe` — bus subscriptions
- `__go_brainkit_await_approval` — HITL approval bridge (Go-side bus lifecycle)
- `__go_console_log_tagged` — per-Compartment tagged logging
- `__go_registry_resolve` / `__go_registry_has` / `__go_registry_list` — provider registry queries

These bridges MUST be registered before loadRuntime (step 5) because kit_runtime.js calls them during evaluation.

**Step 4 — Embedded storages.** Starts a libsql HTTP bridge server for each SQLite `StorageConfig`. Each server is a Go HTTP server speaking the Hrana pipeline protocol, backed by a local SQLite file via `modernc.org/sqlite`. The server URL is injected into JS globalThis so `new LibSQLStore({ id: "x" })` can connect without the user providing a URL.

**Step 5 — Kit runtime.** `loadRuntime()` evaluates `kit_runtime.js` which sets up `globalThis.__kit` (bus API, resource registry, model resolution, generateWithApproval, Compartment endowments), then registers four ES modules: `"kit"`, `"ai"`, `"agent"`, `"compiler"`. After this step, `import { bus } from "kit"` works in deployed .ts code.

**Step 6 — Provider registry.** Creates `ProviderRegistry` and registers all AI providers, vector stores, and storages from config. AI providers are auto-detected from `os.Getenv` (e.g. `OPENAI_API_KEY` makes the OpenAI provider available automatically).

**Step 7 — WASM + remaining domains.** Creates WASMService (lazy AS compiler), WASMDomain, LifecycleDomain, RegistryDomain. Starts periodic health probing if configured. Loads persisted WASM modules and shard descriptors from KitStore.

**Step 8 — MCP.** Connects to configured MCP servers (stdio or HTTP), fetches their tool lists, registers each MCP tool in the shared tool registry with a GoFuncExecutor that calls through the MCP client.

**Step 9 — Transport + router.** Creates or uses injected Watermill transport. Creates a router with three middleware: DepthMiddleware (cycle detection at depth 16), CallerIDMiddleware (stamps caller identity), MetricsMiddleware (processing time tracking). For standalone Kernel, registers command bindings and starts the router immediately. For Node, defers to the caller.

**Step 10 — Job pump.** `startJobPump()` starts a background goroutine (tracked via `bridge.Go` so it's cancelled on Close) that ticks every 10ms, calling `bridge.ProcessScheduledJobs()`. This is critical: without it, deployed `.ts` services can't receive bus messages when no EvalTS is active. The pump drains the QuickJS job queue (Schedule'd callbacks from Go goroutines + JS microtasks like Promise continuations). The 10ms interval is a balance between latency (lower = faster response) and CPU (lower = more spinning). The goroutine is tracked by `bridge.Go` — this is essential because `bridge.Close` waits for all tracked goroutines before freeing the QuickJS context. Without tracking, the pump could be inside `ctx.Loop()` when Close frees the context → SIGSEGV.

## Shutdown Order

`Kernel.Close()` reverses initialization:

1. Set `closed = true` (prevents new operations)
2. Cancel all JS-side bus subscriptions (prevents new callbacks into QuickJS)
3. Close the Watermill router (stops processing messages — no more handler invocations)
4. Unregister all agents for this Kit
5. Close MCP connections
6. Close WASM service (closes the AS compiler's separate QuickJS runtime if it was created)
7. Close KitStore
8. Close the agent sandbox — this is where QuickJS is freed. `bridge.Close()` cancels all tracked goroutines (job pump, fetch calls, fs operations), waits for them to finish (`wg.Wait()`), nullifies global JS references to break closure chains, runs GC, then frees the QuickJS context and runtime.
9. Close embedded storages (stop libsql HTTP servers, close SQLite files)
10. Close transport — but only if Kernel owns it (`ownsTransport == true`). When Node injected the transport, Node is responsible for closing it.

The order matters for safety: router stops before QuickJS is freed (no handler can touch JS after step 3), goroutines finish before context is freed (step 8 waits), transport closes last (nothing can publish after this).

## The Five Subsystems

### QuickJS

The JavaScript engine. Embedded via `buke/quickjs-go` (CGo wrapper). brainkit wraps it in `jsbridge.Bridge` which adds:

- **Mutex-serialized eval.** All `Eval`/`EvalAsync`/`EvalBytecode` calls acquire a mutex. This makes the Bridge safe for concurrent Go callers — multiple goroutines can call EvalTS without corrupting QuickJS state.

- **Tracked goroutines.** `Bridge.Go(fn)` starts a goroutine that's tracked by a WaitGroup and receives a context that's cancelled on Close. Every polyfill that does async work (fetch, fs, net, exec, timers) uses `Bridge.Go` instead of bare `go`. This guarantees no goroutine touches QuickJS after Close.

- **Reentrant evaluation.** `EvalOnJSThread` handles the case where a bus handler (running in a goroutine) needs to execute JS while another EvalTS is already holding the mutex. If the caller is on the JS thread (same goroutine), it calls `ctx.Eval` directly. If it's a different goroutine, it uses `ctx.Schedule` to queue the eval and waits for the result via a channel. The Await loop in the active EvalTS processes the Schedule'd callback.

- **Background job processing.** `ProcessScheduledJobs` calls `ctx.Loop()` which processes both Schedule'd callbacks (from Go goroutines) and JS microtasks (Promise continuations). The job pump calls this every 10ms.

### SES (Secure EcmaScript)

SES provides `Compartment` and `lockdown()`. After `lockdown()` runs, all JavaScript intrinsics are frozen — `Math.random()`, `Date.now()`, and `new Date()` become "tamed" (throw errors as ambient authority). Each deployed `.ts` file gets its own Compartment with explicit endowments that restore access via pre-lockdown captures stored in `__brainkit_pre_lockdown`. See [deployment-pipeline.md](deployment-pipeline.md).

### Watermill

The message routing framework. brainkit uses it in a specific way: the command catalog defines typed commands (tools.call, kit.deploy, agents.list, etc.) that are registered as Watermill consumer handlers. Each command has a topic, a request type, and a response type. The Host publishes responses to the replyTo topic from the inbound message metadata.

Six transport backends are supported, each with a topic sanitizer that transforms logical topic names into transport-safe names (dots → dashes for NATS, dots → underscores for SQL). See [bus-and-messaging.md](bus-and-messaging.md).

### wazero

Pure Go WebAssembly runtime. Used by WASMService for two execution models:

- **One-shot** (`wasm.run`): Compile AS source → instantiate module → call `run()` → get exit code → destroy. Fresh runtime per execution.
- **Shard** (`wasm.deploy`): Compile → instantiate → call `init()` → register handlers → subscribe to bus topics. Handlers are invoked when messages arrive. State persists between invocations in persistent mode.

The AS compiler has its own separate QuickJS runtime (512MB memory limit, 256MB stack) because compilation is CPU-bound and shouldn't block the agent sandbox. It's lazy-initialized via `ensureCompiler()` and stays alive until the Kernel closes.

### typescript-go

Vendored from microsoft/typescript-go. Pure Go — no Node.js, no esbuild, no external process. Used by `kit.Deploy` when the source file has a `.ts` extension. Strips type annotations, interfaces, generics, type aliases. Preserves all runtime code, imports, exports, async/await. See [deployment-pipeline.md](deployment-pipeline.md).

## Memory Model

Each Kernel has one QuickJS heap (~256MB address space reservation, ~50-80MB actual for the Mastra bundle). If WASM compilation is used, the AS compiler adds another ~512MB reservation. The job pump goroutine and each deployed .ts service's bus subscriptions run within the same QuickJS heap — there's no per-service isolation at the memory level. Isolation is at the Compartment level (frozen endowments, separate global objects) not at the memory level.

For multi-Kit deployments (via InstanceManager pools), each Kit instance gets its own QuickJS heap. A pool of 5 Kits uses ~1.5GB baseline. Tool registries can be shared across pool instances to avoid duplicate registrations.

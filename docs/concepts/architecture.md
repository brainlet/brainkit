# Architecture

brainkit is a Go runtime that embeds four subsystems into one process: QuickJS (JavaScript), SES (sandboxing), Watermill (pub/sub messaging), and typescript-go (TS transpilation). The result is a platform where Go and TypeScript communicate over a shared message bus.

## Why This Architecture

The traditional approach to AI agent platforms is a microservice mesh: separate processes for the agent runtime, tool execution, workflow engine, message broker, and storage. This works but introduces latency, deployment complexity, and failure modes at every boundary.

brainkit takes the opposite approach: everything runs in one OS process. The JS runtime (QuickJS) is embedded in Go via CGo. The message bus (Watermill) routes messages in-process by default, with optional external transports (NATS, Redis, AMQP, Postgres, SQLite) for multi-node deployments. TypeScript transpilation happens natively in Go via a vendored microsoft/typescript-go.

This means a tool call from an AI agent doesn't leave the process. A `.ts` service publishing to the bus doesn't cross a network boundary. Everything is function calls and channels until you explicitly opt into distributed communication via a Node with an external transport.

## Two Runtime Types

### Kernel

The Kernel is the local runtime. It owns all state:

- One QuickJS runtime with SES lockdown, Mastra bundle, and 20+ Node.js polyfills
- One tool registry shared across all surfaces (Go, TS, plugins)
- One Watermill router processing command and event messages
- Zero or more deployed `.ts` files, each in its own SES Compartment
- Zero or more embedded SQLite storage bridges (libsql HTTP servers)

A standalone Kit uses an in-memory transport — messages never leave the process. This is the default and fastest configuration.

```go
// AI providers auto-detected from os.Getenv (e.g. OPENAI_API_KEY)
kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-kit",
    FSRoot:    "/tmp/workspace",
})
defer kit.Close()
```

### Transport-Connected Kit

When `Config.Transport` is set, Kit creates a transport-connected runtime with plugin support and cross-Kit networking. Internally this creates a Node (Kernel + external transport + plugin manager), but the public API is the same `*Kit`.

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-kit",
    FSRoot:    "/tmp/workspace",
    Transport: "nats",
    NATSURL:   "nats://localhost:4222",
    Plugins: []brainkit.PluginConfig{
        {Name: "my-plugin", Binary: "./my-plugin", AutoRestart: true},
    },
})
n.Start(ctx) // restores running plugins, launches configured plugins
defer n.Close()
```

Both Kernel and Node implement `sdk.Runtime` (PublishRaw, SubscribeRaw, Close) and `sdk.CrossNamespaceRuntime` (PublishRawTo, SubscribeRawTo) and `sdk.Replier` (ReplyRaw). Any code that takes an `sdk.Runtime` works with either.

## Initialization Order

NewKernel initializes subsystems in a strict order. Each step depends on the previous ones. The order matters — reordering causes failures that are hard to debug because the dependencies are implicit.

**Step 1 — QuickJS sandbox.** `agentembed.NewSandbox` creates a `jsbridge.Bridge` (QuickJS runtime with mutex-serialized eval), loads 24 polyfills in dependency order (Events before NodeStreams before Buffer before Net — because Socket extends Duplex which extends EventEmitter), then loads SES (polyfills → ses.umd.js → lockdown → Mastra bundle). After this step, `globalThis.__agent_embed` exists with Agent, createTool, createWorkflow, the AI SDK, and all Mastra exports. See [jsbridge-polyfills.md](jsbridge-polyfills.md) and [bundle-and-bytecode.md](bundle-and-bytecode.md).

**Step 2 — Domain handlers.** Creates ToolsDomain, AgentsDomain. These are thin wrappers that route typed messages to the underlying registries and JS runtime. They exist so the command catalog can dispatch to typed handlers.

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

**Step 5 — Kit runtime.** `loadRuntime()` evaluates 8 JS files in dependency order (patches, bridges, approval, infrastructure, resolve, bus, kit_runtime, test_runtime), then registers ES modules: `"kit"`, `"ai"`, `"agent"`, `"fs"`, `"fs/promises"`. After this step, `import { bus } from "kit"` works in deployed .ts code.

**Step 5b — Mastra storage upgrade.** If any storage backend is configured, resolves the default storage, calls `storage.init()` (creates `mastra_workflow_snapshot` and other Mastra domain tables), and upgrades the `_storeHolder` from InMemoryStore to the configured backend. All subsequent workflow snapshots persist to the real database.

**Step 6 — Provider registry.** Creates `ProviderRegistry` and registers all AI providers, vector stores, and storages from config. AI providers are auto-detected from `os.Getenv` (e.g. `OPENAI_API_KEY` makes the OpenAI provider available automatically).

**Step 7 — Remaining domains.** Creates LifecycleDomain, PackagesDomain, SecretsDomain, TestingDomain. Starts periodic health probing if configured. Connects to MCP servers.

**Step 8 — MCP.** Connects to configured MCP servers (stdio or HTTP), fetches their tool lists, registers each MCP tool in the shared tool registry with a GoFuncExecutor that calls through the MCP client.

**Step 9 — Transport + router.** Creates or uses injected Watermill transport. Creates a router with three middleware: DepthMiddleware (cycle detection at depth 16), CallerIDMiddleware (stamps caller identity), MetricsMiddleware (processing time tracking). For standalone Kernel, registers command bindings and starts the router immediately. For Node, defers to the caller.

**Step 10 — Job pump + recovery.** `startJobPump()` starts a background goroutine (tracked via `bridge.Go` so it's cancelled on Close) that processes QuickJS scheduled callbacks every 100ms (with immediate wake on `pumpSignal`). After the pump starts, persisted .ts deployments are re-deployed (`redeployPersistedDeployments`), persisted schedules are restored, and `restartActiveWorkflows()` picks up any workflow runs that were active before the previous shutdown (status `running` or `waiting` in storage). This is Mastra's `restartAllActiveWorkflowRuns()` called on each registered workflow.

## Shutdown Order

`Kernel.Close()` reverses initialization:

1. Set `closed = true` (prevents new operations)
2. Cancel all JS-side bus subscriptions (prevents new callbacks into QuickJS)
3. Close the Watermill router (stops processing messages — no more handler invocations)
4. Unregister all agents for this Kit
5. Close MCP connections
6. Close KitStore
7. Close the agent sandbox — this is where QuickJS is freed. `bridge.Close()` cancels all tracked goroutines (job pump, fetch calls, fs operations), waits for them to finish (`wg.Wait()`), nullifies global JS references to break closure chains, runs GC, then frees the QuickJS context and runtime.
8. Close embedded storages (stop libsql HTTP servers, close SQLite files)
9. Close transport — but only if Kernel owns it (`ownsTransport == true`). When Node injected the transport, Node is responsible for closing it.

The order matters for safety: router stops before QuickJS is freed (no handler can touch JS after step 3), goroutines finish before context is freed (step 7 waits), transport closes last (nothing can publish after this).

## The Four Subsystems

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

### typescript-go

Vendored from microsoft/typescript-go. Pure Go — no Node.js, no esbuild, no external process. Used by `kit.Deploy` when the source file has a `.ts` extension. Strips type annotations, interfaces, generics, type aliases. Preserves all runtime code, imports, exports, async/await. See [deployment-pipeline.md](deployment-pipeline.md).

## Memory Model

Each Kernel has one QuickJS heap (~256MB address space reservation, ~50-80MB actual for the Mastra bundle). The job pump goroutine and each deployed .ts service's bus subscriptions run within the same QuickJS heap — there's no per-service isolation at the memory level. Isolation is at the Compartment level (frozen endowments, separate global objects) not at the memory level.

For multi-Kit deployments (via InstanceManager pools), each Kit instance gets its own QuickJS heap. A pool of 5 Kits uses ~1.5GB baseline. Tool registries can be shared across pool instances to avoid duplicate registrations.

Note: wazero (WebAssembly runtime) is still a dependency — used by `jsbridge/webassembly.go` to provide `WebAssembly.instantiate()` for JS libraries that ship WASM modules (like xxhash-wasm). The AssemblyScript compiler in `internal/embed/compiler/` is dormant and not wired to the Kernel.

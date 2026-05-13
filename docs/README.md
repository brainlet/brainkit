# brainkit

Brainkit is an embeddable Go runtime for AI agent teams. A single `Kit` instance starts as a light control-plane runtime with an async pub/sub bus; opt-in modules add SES-hardened `.ts` deployments, AI SDK v5, Mastra, storage and vector bridges, MCP, observability, HTTP gateway, and plugin hosting.

This directory is the reference. The five files in `llm/` are dense, API-only pages intended for LLM ingestion and copy-paste. The narrative prose — concepts, design notes, walkthroughs — lives in `concepts/` and `guides/`.

---

## Architecture at a glance

- **Kit**: a single `*Kit` value created via `brainkit.New(brainkit.Config{...})`. Implements `sdk.Runtime` (publish / subscribe / reply / stream), `sdk.CrossNamespaceRuntime` (routed `To:` targeting), and `sdk.Replier` (correlated replies). Zero-value defaults — `Transport: brainkit.Memory()` if unset, embedded AMQP / NATS / Redis helpers are opt-in.
- **Bus**: async pub/sub with typed `brainkit.Call[Req, Resp]` / `brainkit.CallStream[Req, Chunk, Resp]` helpers for Kit-specific behavior and generated package-owned `CallXxx(rt, ctx, msg, opts...)` wrappers for shipped message types. Module code can use matching `CallXxxWithCaller(caller, ctx, msg, opts...)` wrappers with the narrow request/reply capability. SDK-owned wrappers live in `sdk/typed_gen.go`; module-owned wrappers live with their module message package.
- **SES compartments**: every deployed `.ts` file runs in its own hardened compartment with a tamed global surface (frozen `Date`, per-source module namespace `ts.<source>.<topic>`, no network / child-process access). Endowments are injected per-source: `bus`, `kit`, `model`, `embeddingModel`, `provider`, `storage`, `vectorStore`, `registry`, `tools`, `tool`, `fs` (Node.js shape), `mcp`, `output`, `secrets`, `generateWithApproval`, the full `"ai"` module and the full `"agent"` (Mastra) module.
- **Transports**: `brainkit.Memory()` (default, zero-value) is linked by the root package. Import backend-specific packages such as `transports/embeddednats`, `transports/nats`, `transports/amqp`, or `transports/redis` for network transports. The aggregate `transports` package still exists, but it links every network backend. Topic sanitisers vary per backend; the Go surface is uniform.
- **Providers**: 12 built-in constructors (OpenAI, Anthropic, Google, Mistral, Groq, DeepSeek, XAI, Cohere, Perplexity, TogetherAI, Fireworks, Cerebras). `WithBaseURL(...)` / `WithHeaders(...)` options on every one.
- **Storage & vectors**: 5 storage constructors (SQLite, Postgres, MongoDB, Upstash, InMemory) and 3 vector constructors (SQLite, PgVector, MongoDB). Registered as a named pool the runtime can look up by name.
- **Modules**: 24 shipped module packages composed via `Config.Modules`. `presets/standard/core.Set()` mounts the light command/control plane; `presets/standard/runtime.Set()` adds JS runtime/eval without source-package builders; `presets/standard/artifactruntime.Set()` adds JS runtime/eval for normalized JavaScript artifacts only; `presets/standard/packages.Set()` adds package deployment plus the standard esbuild package builder; `presets/standard.CommandSet()` is the light command/control-plane convenience alias; `presets/standard.FullCommandSet()` is the aggregate core plus package deployment set. Additional modules own resource-heavy or integration-specific surfaces such as `gateway`, `mcp`, `plugins`, `schedules`, `audit`, `tracing`, `probes`, `discovery`, `topology`, `workflow`, `testing`, and `harness`.
- **server**: thin HTTP wrapper (`server.New`) for bringing a Kit up behind an HTTP gateway. `server/configfile` reads YAML with `$VAR` / `${VAR}` expansion; import named `server/standard/...` profiles for built-in YAML module names, `server/configfile/transportbackends/...` for YAML transport backends, `server/configfile/storebackends/...` for KitStore backends, and `server/configfile/packageboot` for top-level `packages:` auto-deploy. `server/standard/full` is the all-module catalog; `server/quickstart` holds the batteries-included preset. Not required — embed the Kit directly from Go in any long-running process.

See `concepts/architecture.md` for the full diagram and `concepts/deployment-pipeline.md` for package normalization, runtime handoff, and SES Compartment evaluation.

---

## Deployment model

All agents, tools, workflows and memories are created by **deploying a `.ts` file**. Two Go entry points:

```go
// Create a Kit
kit, _ := brainkit.New(brainkit.Config{
    Namespace: "myapp",
    Transport: embeddednats.New(),
    FSRoot:    "/var/lib/myapp",
    Providers: []brainkit.ProviderConfig{brainkit.OpenAI(os.Getenv("OPENAI_API_KEY"))},
})
defer kit.Close()

// Deploy a .ts package
_, _ = packageclient.Deploy(ctx, kit, packageclient.Inline(
    "researcher",
    "researcher.ts",
    researcherCode,
))
```

Inside `researcher.ts`:

```typescript
import { Agent, createTool, z } from "agent";
import { model, kit, bus } from "kit";

const search = createTool({
    id: "search",
    description: "Search the knowledge base",
    inputSchema: z.object({ q: z.string() }),
    execute: async (args) => {
        const { q } = (args && args.context) || args;
        return { hits: ["…results for " + q] };
    },
});

const agent = new Agent({
    name: "researcher",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You research topics thoroughly.",
    tools: { search },
});
kit.register("agent", "researcher", agent);

bus.on("ask", async (msg) => {
    const r = await agent.generate(String(msg.payload?.prompt ?? ""));
    msg.reply({ text: r.text, usage: r.usage });
});
```

From Go, reach the deployment on its mailbox `ts.<source>.<topic>` — so `ts.researcher.ask` for the file above:

```go
reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
    Topic:   "ts.researcher.ask",
    Payload: json.RawMessage(`{"prompt":"what is rag?"}`),
}, brainkit.WithCallTimeout(30*time.Second))
```

Teardown releases all bus subscriptions, unregisters agents / tools / workflows, and disposes the compartment:

```go
_, _ = packageclient.Teardown(ctx, kit, "researcher")
```

`examples/agent-spawner/main.go` is the flagship walkthrough: a Go program deploys an architect agent, asks it to design and deploy a second agent at runtime, and then calls the newly-spawned agent directly over the bus.

---

## Reference files (`llm/`)

Dense, API-only pages — each mirrors one source-of-truth and is kept in sync with the shipped code.

| File | Covers |
|------|--------|
| [`llm/go-sdk.md`](llm/go-sdk.md) | `sdk.Runtime` / `CrossNamespaceRuntime` / `Replier` interfaces, normal request/reply via `brainkit.Call[Req,Resp]` / `CallStream[Req,Chunk,Resp]` and generated package-owned `CallXxx` wrappers, event/subscription helpers (`Emit` / `SubscribeTo`), diagnostics-only `sdk/protocol` helpers (`Publish` / `PublishTo` / `SendToService` / `ResolveServiceTopic`), handler reply helpers (`Reply` / `SendChunk`), envelopes (`EnvelopeOK` / `EnvelopeErr` / encode / decode / `IsEnvelope`), `CallOption` surface, typed SDK and module messages, errors, context keys. |
| [`llm/go-config.md`](llm/go-config.md) | `brainkit.Config` (every field + default), `brainkit.New` / `brainkit.QuickStart`, the 12 `ProviderConfig` constructors with `WithBaseURL` / `WithHeaders`, the 5 `TransportConfig` helpers (+ `WithNATSName`), the standard module catalog with status and module-specific helpers (`NewSQLiteTraceStore`, audit stores, tracing / discovery / topology / MCP / gateway configs), `StorageConfig` + `VectorConfig`, `KitStore` + records + `SQLiteStore` + `NewPostgresStore`, `SecretStore` + `$secret:NAME` interpolation, `TraceStore` contracts, retry / error / health types, module-owned `plugins.PluginConfig` + `schedules.ScheduleConfig`, `server.Config` + `server.Server` + `server/quickstart` + `server/configfile` YAML shape. |
| [`llm/ts-runtime.md`](llm/ts-runtime.md) | SES compartment execution model, mailbox naming `ts.<source>.<topic>`, endowment map, `BrainkitError` + error codes (`VALIDATION_ERROR`, `NOT_FOUND`, `TIMEOUT`, `HANDLER_FAILED`, `TRANSPORT_ERROR`, `COMPARTMENT_ERROR`, `TOPIC_COLLISION`, `NOT_CONFIGURED`, `PLUGIN_*`), full `bus` API (`publish`/`emit`/`subscribe`/`on`/`sendTo`/`call`/`callTo`/`schedule`/`onCancel`/`withCancelController`), `BusMessage.reply`/`send`/`stream.text`/`progress`/`object`/`event`/`error`/`end` with `seq` semantics, `kit.register` valid types, `model` / `embeddingModel` / `provider` resolvers, `storage` / `vectorStore` named pools (LibSQL file-URL guards), `registry`, `tools` / `tool`, the Node.js-shaped `fs` endowment, `mcp`, `output`, `secrets.get`, `generateWithApproval`, tamed `Date` / `Math`, tagged `console`, deployment patterns, failure semantics. |
| [`llm/ai-sdk.md`](llm/ai-sdk.md) | The `"ai"` module (AI SDK v5, no wrapping). `CallSettings` (`maxOutputTokens`, not `maxTokens`), `Usage` with v5 names (`inputTokens` / `outputTokens`) + deprecated v4 aliases, `generateText` + `GenerateTextParams` (with `stopWhen` and `@deprecated maxSteps`), `streamText` + `StreamPart` union, `generateObject`, `streamObject`, `embed`, `embedMany`, middleware (`defaultSettingsMiddleware`, `extractReasoningMiddleware`, `wrapLanguageModel`), `tool<T>`, `jsonSchema`, the Zod surface. |
| [`llm/mastra.md`](llm/mastra.md) | The `"agent"` module (Mastra, no wrapping). `Agent` class + `AgentConfig` + `AgentCallOptions`, `AgentResult` with **v4 usage names** (`promptTokens` / `completionTokens`), `AgentStreamResult`, `createTool` + `ToolConfig`, `createWorkflow` + `createStep` + builder (`then` / `parallel` / `branch` / `foreach` / `dountil` / `sleep` / `commit`), `Memory` + `MemoryConfig` + `MemoryOptions` (semantic recall, working memory, observational memory), 5 storage classes (`InMemoryStore`, `LibSQLStore` — `opts.url` file-URL guard, `UpstashStore`, `PostgresStore`, `MongoDBStore`), 3 vector classes (`LibSQLVector` — `opts.connectionUrl` file-URL guard, `PgVector`, `MongoDBVector`), `ModelRouterEmbeddingModel`, `MDocument` / `GraphRAG` / `createVectorQueryTool` / `createDocumentChunkerTool` / `createGraphRAGTool` / `rerank` / `rerankWithScorer`, `Observability` + `DefaultExporter` + `SensitiveDataFilter`, `createScorer` builder + `runEvals`, `Workspace` + `LocalFilesystem` + `LocalSandbox`, `RequestContext`, HITL flow (tool `requireApproval`, workflow `ctx.suspend`, `generateWithApproval`). |

---

## Bus topic catalog

The authoritative list of built-in bus topics (with request / response Go types and source file) is generated from `sdk/**/*_messages.go` and `modules/**/*_messages.go` and kept in [`bus-topics.md`](bus-topics.md). Do not edit by hand — run `go run scripts/gen-bus-topics.go` to regenerate.

Everything in the shipped SDK — discovery, audit, gateway, MCP, plugins, schedules, secrets, storage pool, tracing, workflow control, vector pool, package lifecycle — flows through these typed topics.

---

## Guides (prose)

Walkthroughs and concept-level docs under `guides/`:

- [`guides/getting-started.md`](guides/getting-started.md) — first Kit, first deployment.
- [`guides/go-sdk.md`](guides/go-sdk.md) — the Go-side SDK in narrative form.
- [`guides/ts-services.md`](guides/ts-services.md) — writing `.ts` services against the endowments.
- [`guides/ai-and-agents.md`](guides/ai-and-agents.md) — building Mastra agents, tools and memories.
- [`guides/voice-and-audio.md`](guides/voice-and-audio.md) — voice providers, `new Audio(stream).play()` + `audio.Sink` fan-out, browser realtime, desktop audio self-test.
- [`guides/vectors-and-rag.md`](guides/vectors-and-rag.md) — embedding pipelines, `MDocument`, `GraphRAG`, `createVectorQueryTool`.
- [`guides/storage-and-memory.md`](guides/storage-and-memory.md) — the storage pool, vector pool and Mastra `Memory`.
- [`guides/transport-backends.md`](guides/transport-backends.md) — memory / embedded-NATS / NATS / AMQP / Redis trade-offs and topic sanitisers.
- [`guides/mcp-integration.md`](guides/mcp-integration.md) — the MCP module + `mcp.listTools` / `mcp.callTool` bus surface.
- [`guides/hitl-approval.md`](guides/hitl-approval.md) — `generateWithApproval`, Mastra tool `requireApproval`, workflow `ctx.suspend`.
- [`guides/plugins.md`](guides/plugins.md) — out-of-process plugin SDK, `plugins.PluginConfig`, lifecycle events.
- [`guides/observability.md`](guides/observability.md) — tracing, probes, topology, metrics.

## Concepts (design notes)

- [`concepts/architecture.md`](concepts/architecture.md) — Kit anatomy.
- [`concepts/bus-and-messaging.md`](concepts/bus-and-messaging.md) — bus internals, interceptors, worker groups.
- [`concepts/cross-kit.md`](concepts/cross-kit.md) — routing across namespaces and Kits.
- [`concepts/deployment-pipeline.md`](concepts/deployment-pipeline.md) — transpile → strip imports → SES lockdown → endow.
- [`concepts/bundle-and-bytecode.md`](concepts/bundle-and-bytecode.md) — how the embedded JS bundle / bytecode cache is built.
- [`concepts/jsbridge-polyfills.md`](concepts/jsbridge-polyfills.md) — Node.js polyfill strategy (`fs`, `crypto`, `stream`, etc.).
- [`concepts/provider-registry.md`](concepts/provider-registry.md) — provider resolution and lookup rules.
- [`concepts/error-handling.md`](concepts/error-handling.md) — error envelopes, retry policy, handler events.
- [`concepts/scaling.md`](concepts/scaling.md) — multi-Kit topologies, gateway fan-out.

---

## Quick examples

Minimal in-process Kit with a Memory transport and a single deployment:

```go
kit, _ := brainkit.New(brainkit.Config{
    Namespace: "demo",
    Transport: brainkit.Memory(),        // default — zero value also works
    FSRoot:    os.TempDir(),
    Providers: []brainkit.ProviderConfig{brainkit.OpenAI(os.Getenv("OPENAI_API_KEY"))},
    Modules:   standard.FullCommandSet(), // JS runtime, package.deploy, tools.*, health, eval, ...
})
defer kit.Close()

_, _ = packageclient.Deploy(ctx, kit, packageclient.Inline("echo", "echo.ts", `
    import { bus } from "kit";
    bus.on("ping", (msg) => msg.reply({ pong: msg.payload }));
`))

reply, _ := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
    Topic:   "ts.echo.ping",
    Payload: json.RawMessage(`{"hello":"world"}`),
}, brainkit.WithCallTimeout(2*time.Second))
```

Stand up an HTTP gateway in front of the same Kit with one call:

```go
import "github.com/brainlet/brainkit/server/quickstart"

srv, _ := quickstart.New("demo", "/var/lib/demo")
defer srv.Close()
// HTTP gateway is live on :8080 via the embedded NATS transport.
```

Load a Kit from YAML with env-var expansion (`$VAR` and `${VAR}` are substituted at load time by `configfile.Load`):

```yaml
# config.yaml
namespace: demo
fsRoot: /var/lib/demo
secret_key: ${SECRET_KEY}
transport:
  type: embedded
providers:
  - name: openai
    apiKey: ${OPENAI_API_KEY}
modules:
  jsruntime: {}
  gateway:
    listen: :8080
  reference: {}
  eval: {}
  messaging: {}
  health: {}
  packages: {}
  tracing: {}
```

```go
import (
    "github.com/brainlet/brainkit/server/configfile"
    _ "github.com/brainlet/brainkit/server/configfile/packageboot"
    _ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
    _ "github.com/brainlet/brainkit/server/configfile/transportbackends/embeddednats"
    _ "github.com/brainlet/brainkit/server/standard/commands"
    _ "github.com/brainlet/brainkit/server/standard/packages"
    _ "github.com/brainlet/brainkit/server/standard/server"
)

cfg, _ := configfile.Load("config.yaml")
srv, _ := server.New(cfg)
defer srv.Close()
```

---

## Versioning

This documentation tracks the `v1.0.0-rc.1` API surface. Breaking changes land on `main` — the `llm/*` reference is the contract surface and is regenerated whenever a shipped type changes. For internal, unexported types see the package comments in the Go source directly.

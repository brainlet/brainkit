# brainkit Features Map

> This document is the source of truth for what brainkit can do.
> Every feature claimed here must have a corresponding test in `test/TEST_COVERAGE.md`.
> Status: DONE = tested and working, PARTIAL = implemented with known limitations, PLANNED = not yet built.

---

## Design Principles

These drive every feature decision in brainkit.

**Everything is dynamic.** No hardcoded strings in `.ts` files. Providers, models, vector stores, storage backends — all resolved from the registry at runtime. A `.ts` file calls `model("openai", "gpt-4o-mini")` or `vectorStore("main")`, never raw API keys or connection strings.

**Everything integrations support, brainkit controls.** If Mastra or the AI SDK can manage a capability, brainkit must expose it — with health probing, env injection, and debuggability. Anyone deploying brainkit should be able to diagnose what's healthy, what's connected, and what's failing without reading source code.

**brainkit != its integrations.** brainkit is the wiring — composing Watermill, QuickJS, wazero, Mastra, and the AI SDK into a coherent runtime. When an integration (PgVector, LibSQL, etc.) requires specific infrastructure, that's a deployment concern. brainkit matches the API correctly and lets the infrastructure requirement flow through to the consumer.

**Use cases we won't know.** brainkit is a toolkit. Two Kits should be able to distribute workload, or ask another Kit to do specific work. Someone might want to maintain different environments locally or remotely for whatever reason. Every domain must work across Kit boundaries because we can't predict how people will compose Kits.

**WASM is automation AND administration.** WASM shards aren't just event processors — they're automated systems that can do both task automation and administration automation. A shard should be able to deploy .ts code, query the registry, manage Kit lifecycle — anything a Go host can do.

**Full bus parity across surfaces.** Every surface (TS, WASM, Plugin) should have equivalent access to the message bus. WASM should be the Watermill equivalent — subscribe to anything, publish to anything, raw or with known types. No surface should be a second-class citizen.

**Agents and workflows need the filesystem.** TS code runs agents, tools, and workflows that process data. They need to read, list, and write files on disk. The `fs` API exists in TS because agent developers need it, not because it's technically interesting.

---

## Runtime

| Feature | Status | Description |
|---------|--------|-------------|
| Kernel (standalone) | DONE | Local runtime with internal GoChannel transport. Owns JS/WASM/domain state. |
| Node (transport-connected) | DONE | Kernel + external Watermill transport (NATS, AMQP, Redis, Postgres, SQLite). |
| Plugin SDK | DONE | `sdk.New()` + `sdk.Tool()` + `plugin.Run()` — separate process, connects via transport. |
| InstanceManager (pools) | DONE | Pool of Nodes with shared tool registry. Static + threshold scaling strategies. |
| Cross-Kit communication | DONE | `sdk.PublishAwaitTo` — call operations on another Kit's namespace over shared transport. Two or more Kits can distribute workload or ask each other to do specific work. |
| Cross-Kit all domains | DONE | tools, fs, agents, kit, wasm, registry, ai, memory, workflows, mcp, vectors tested cross-Kit. Every domain works across Kit boundaries because we can't predict how people will compose Kits. |

## Transport Backends

| Backend | Type | Status | Sanitizer |
|---------|------|--------|-----------|
| GoChannel (in-memory) | `"memory"` | DONE | none |
| SQLite | `"sql-sqlite"` | DONE | dots/slashes → underscores |
| NATS JetStream | `"nats"` | DONE | dots/slashes → dashes |
| AMQP (RabbitMQ) | `"amqp"` | DONE | slashes → dashes (dots kept) |
| Redis Streams | `"redis"` | DONE | none |
| PostgreSQL | `"sql-postgres"` | DONE | dots/slashes → underscores |

All 6 backends tested with 12 domain operations via `backend_matrix_test.go`.

## API Surfaces

### Go Direct

| Role | Host application — creates Kits, registers Go tools, manages lifecycle |
|------|---|
| Runtime | Kernel or Node |
| Path | `sdk.PublishAwait` → transport → router → handler |
| Status | **DONE** — all domains, all operations |

### TS (.ts deploy)

| Role | Agent/tool/workflow developers — write .ts deployed into SES Compartments |
|------|---|
| Runtime | QuickJS inside Kernel (LocalInvoker — no transport) |
| Capabilities | agents, tools, workflows, memory, vectors, AI (generate/stream/embed), fs, wasm, mcp, registry, bus |
| Status | **DONE** — all domains, all operations |

TS code runs in isolated SES Compartments with per-source logging. Has full access to Mastra ecosystem (Memory, scorers, processors, RAG, workspace). Includes filesystem access because agents and workflows need to read, list, and write files on disk. Everything resolved dynamically from the registry — no hardcoded strings.

### WASM (AssemblyScript)

| Role | Automation AND administration automation — event-driven shards, Kit management, system orchestration |
|------|---|
| Runtime | wazero inside Kernel (LocalInvoker via invokeAsync) |
| Host functions | `send`, `invokeAsync`, `on`, `tool`, `reply`, `log`, `get_state`, `set_state`, `has_state`, `set_mode` |
| Status | **PARTIAL** — all domains via invokeAsync, but bus parity missing |

WASM shards are not just event processors. They can deploy .ts code, query the registry, manage Kit lifecycle — automated systems for both task automation and administration. The goal is full bus parity: WASM should be the Watermill equivalent, able to subscribe to anything and publish to anything, raw or typed.

| WASM Capability | Status |
|----------------|--------|
| invokeAsync (any domain) | DONE |
| send (publish event) | DONE |
| on (init-time handler) | DONE |
| reply (shard response) | DONE |
| state (get/set/has) | DONE |
| subscribe (runtime) | **PLANNED** — can't dynamically subscribe to topics at runtime |
| unsubscribe (runtime) | **PLANNED** |
| typed message publish | **PLANNED** — currently raw JSON strings only |

### Plugin (subprocess)

| Role | Extension developers — separate process, any language |
|------|---|
| Runtime | Separate process, connects via NATS (or any Watermill transport) |
| Capabilities | All domains via `sdk.PublishAwait`, tool registration, event subscription |
| Status | **DONE** — all domains, full lifecycle |
| Planned | Plugin-defined message types (requires WASM AS recompilation) |

## Domains

### tools

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| call | DONE | DONE | DONE | DONE | DONE |
| resolve | DONE | DONE | DONE | DONE | DONE |
| list | DONE | DONE | DONE | DONE | DONE |

Tools can be registered from Go (`kit.RegisterTool`), TS (`createTool`), or Plugins (manifest). Resolved by short name, full name, or semver.

### fs

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| read | DONE | DONE | DONE | DONE | DONE |
| write | DONE | DONE | DONE | DONE | DONE |
| list | DONE | DONE | DONE | DONE | DONE |
| stat | DONE | DONE | DONE | DONE | DONE |
| delete | DONE | DONE | DONE | DONE | DONE |
| mkdir | DONE | DONE | DONE | DONE | DONE |

Sandboxed to `WorkspaceDir`. Path traversal rejected.

### agents

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| list | DONE | DONE | DONE | DONE | DONE |
| discover | DONE | DONE | DONE | DONE | DONE |
| get-status | DONE | DONE | DONE | DONE | — |
| set-status | DONE | DONE | DONE | DONE | — |
| request | DONE | DONE | DONE | DONE | — |
| message | DONE | DONE | DONE | DONE | — |

Agents created via TS `agent()`. Status: idle/busy/error. Request uses real AI (OpenAI).

### ai

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| generate | DONE | DONE | DONE | DONE | DONE |
| embed | DONE | DONE | DONE | DONE | DONE |
| embedMany | DONE | DONE | DONE | DONE | — |
| generateObject | DONE | DONE | DONE | DONE | — |
| stream | DONE | DONE | DONE | DONE | — |

Model resolution: `"openai/gpt-4o-mini"` → provider factory from registry. Real OpenAI API in tests.

### memory

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| createThread | DONE | DONE | DONE | DONE | DONE |
| getThread | DONE | DONE | DONE | DONE | DONE |
| listThreads | DONE | DONE | DONE | DONE | DONE |
| save | DONE | DONE | DONE | DONE | DONE |
| recall | DONE | DONE | DONE | DONE | — |
| deleteThread | DONE | DONE | DONE | DONE | DONE |

Mastra Memory with InMemoryStore, LibSQLStore, PostgresStore, etc. Observational memory supported.

### workflows

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| run | DONE | DONE | DONE | DONE | DONE |
| resume | DONE | DONE | — | DONE | — |
| cancel | DONE | — | — | DONE | — |
| status | DONE | — | — | DONE | — |

Mastra workflows with steps, suspend/resume, snapshot persistence. RunId included in response for lifecycle tracking.

### vectors

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| createIndex | DONE | DONE | DONE | DONE | DONE |
| listIndexes | DONE | DONE | DONE | DONE | — |
| deleteIndex | DONE | DONE | DONE | DONE | — |
| upsert | PARTIAL* | — | — | — | — |
| query | PARTIAL* | — | — | — | — |

`*` PgVector createIndex works (proves full handler→JS→Postgres wiring). Upsert/query fail inside Mastra's @neondatabase/serverless WebSocket driver in QuickJS — not a brainkit issue.

Real pgvector container (Podman) in tests.

### wasm

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| compile | DONE | DONE | N/A | DONE | DONE |
| run | DONE | DONE | N/A | DONE | DONE |
| deploy | DONE | DONE | N/A | DONE | — |
| undeploy | DONE | DONE | N/A | DONE | — |
| describe | DONE | DONE | N/A | DONE | — |
| list | DONE | DONE | N/A | DONE | DONE |
| get | DONE | DONE | N/A | DONE | — |
| remove | DONE | DONE | N/A | DONE | — |

AS compiler (embedded QuickJS). wazero runtime. 10 host functions + abort. Stateless and persistent modes. Shard handler invocation with reply.

### kit

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| deploy | DONE | — | DONE | DONE | DONE |
| teardown | DONE | — | DONE | DONE | DONE |
| redeploy | DONE | — | — | DONE | — |
| list | DONE | DONE | DONE | DONE | DONE |

SES Compartment isolation. Per-source resource tracking. Deploy → Teardown → Redeploy lifecycle.

### mcp

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| listTools | DONE | DONE | DONE | DONE | DONE |
| callTool | DONE | DONE | DONE | DONE | DONE |

MCP servers connected via stdio or HTTP. Tools auto-registered in the tool registry.

### registry

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| has | DONE | DONE | DONE | DONE | DONE |
| list | DONE | DONE | DONE | DONE | DONE |
| resolve | DONE | DONE | DONE | DONE | — |
| register (runtime) | DONE | DONE | — | — | — |
| unregister (runtime) | DONE | DONE | — | — | — |

47 typed config structs (16 AI providers, 17 vector stores, 14 storage backends). Dynamic register/unregister from Go and TS. Env var injection into process.env. Available from all surfaces because any code — TS agent, WASM shard, plugin — needs to know which models are configured without hardcoding strings.

### streaming

| Operation | Go | TS | WASM | Plugin | Cross-Kit |
|-----------|:--:|:--:|:----:|:------:|:---------:|
| ai.stream (fire-and-forget) | DONE | DONE | DONE | DONE | — |
| StreamChunk subscribe | DONE | — | — | DONE | — |
| Formalized streaming domain | **PLANNED** | **PLANNED** | **PLANNED** | **PLANNED** | — |

Currently: `ai.stream` publishes `StreamChunk` messages to a `streamTo` topic. No formal `streaming.*` catalog commands. No typed subscribe-side API from TS/WASM.

### plugin (Node-only)

| Operation | Status |
|-----------|--------|
| manifest | DONE |
| state.get | DONE |
| state.set | DONE |

Plugin registers capabilities via manifest. State persisted via NATS KV or in-memory.

## Cross-cutting Features

| Feature | Status | Description |
|---------|--------|-------------|
| SES Compartments | DONE | Per-.ts isolation with source-tracked globals |
| Per-source logging | DONE | LogHandler with source tags (TS Compartments + WASM modules) |
| Provider registry | DONE | 47 typed configs, has/list/resolve/register/unregister |
| Live HTTP probing | DONE | 14 AI provider endpoints, real HTTP health checks. Everything integrations support should be controllable and debuggable — probing is how you know what's healthy. |
| JS runtime probing | DONE | Vector store + storage instantiation probing via Kernel. Tests real connectivity, not just config validity. |
| Periodic probing | DONE | Background ticker at ProbeConfig.PeriodicInterval. Continuous health monitoring for deployment and administration. |
| Env var injection | DONE | BRAINKIT_* vars injected into JS process.env on registration |
| IIFE closure caching | DONE | vectorStore/storage/model/provider cached via closures (not `this`) |
| Observability | DONE | Auto-tracing via Mastra Observability + DefaultExporter |
| KitStore persistence | DONE | SQLite-backed WASM module + shard state persistence |
| fs JS API | DONE | fs.{read,write,list,stat,delete,mkdir} in kit_runtime.js |
| Registry catalog commands | DONE | registry.{has,list,resolve} as bus commands (accessible from all surfaces) |

## Planned Work

| Feature | Description | Why | Impact |
|---------|-------------|-----|--------|
| **Streaming formalization** | Add `streaming.*` catalog commands. Typed subscribe API from all surfaces. Chunk sequence validation. | Should have been done already. Streaming is how AI responses are delivered in real-time. Without formal catalog commands, every surface reimplements chunk handling. | High |
| **WASM bus parity** | Runtime `subscribe`/`unsubscribe` host functions. Typed message publish. Dynamic topic listening beyond init-time `on()`. Raw and typed pub/sub. | WASM should be the Watermill equivalent — able to subscribe to anything and publish to anything. Currently limited to `send` (fire-and-forget) and `on` (init-time only). A WASM shard doing administration automation needs to listen for events dynamically. | High |
| **Plugin new types** | Plugins define custom message types that WASM can use. Requires AS codegen + recompilation for type compatibility. | Plugins extend the platform — if a plugin brings new capabilities with new message shapes, WASM shards should be able to work with those types. Compilation is the bridge between dynamic (Plugin) and static (WASM). | Medium |
| **Vectors upsert/query** | Fix PgVector @neondatabase/serverless WebSocket driver in QuickJS, or add alternative driver. | Vector search is incomplete without data operations. createIndex works (proves wiring), but the actual use case — store embeddings, query by similarity — doesn't work due to the JS driver limitation. | Medium |

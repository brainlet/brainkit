# Brainkit Documentation

Brainkit is the execution engine for [Brainlet](https://github.com/brainlet/brainlet). A Kit is a self-contained environment — one QuickJS runtime with Mastra + AI SDK + polyfills loaded, plus Go services (bus, tool registry, WASM, storage, plugins, networking). All agents, AI calls, workflows, and `.ts` code run inside a Kit.

## Architecture

- **Bus**: Async message bus with Send/Ask/On primitives. Interceptor pipeline. Worker groups. Address routing for cross-Kit communication. InProcessTransport dispatches handlers concurrently.
- **SES Compartments**: Each deployed `.ts` file runs in its own [SES](https://github.com/endojs/endo/tree/master/packages/ses) Compartment (220KB, bundled into agent-embed). `lockdown()` is called during Kit init to freeze all intrinsics. Compartments receive hardened endowments via `__kitEndowments(source)` with per-source wrappers. Agents and tools are created by deploying `.ts` code, not by sending bus messages.
- **Deployment**: `Kit.Deploy(ctx, source, code)` deploys code in an isolated compartment. `Kit.Teardown(ctx, source)` removes the compartment and all its resources (bus subscriptions, registered agents/tools). `Kit.Redeploy(ctx, source, code)` does an atomic swap (teardown + deploy).
- **Plugin SDK**: Registration-based (`sdk.New` + `sdk.Tool` + `sdk.On` + `sdk.Event`). Out-of-process gRPC. Auto-restart, backpressure, stream recovery, interceptors. Plugin manifest files deploy in Compartments.
- **Tool Registry**: `owner/pkg@version/tool` naming with 5-level resolution (exact → no-owner → no-version → bare → short name).
- **WASM Shards**: Two modes (stateless/persistent). 10 host functions. AssemblyScript library with typed namespace functions.
- **Agent Registry**: 8 bus topics for agent lifecycle (register, discover, request, message, status). Agents auto-unregister and clean up bus subscriptions on teardown.
- **Networking**: Kit-to-Kit over gRPC. Discovery (Static + Multicast LAN). NATS as alternative transport. On `Kit.Close()`, the network stops before the bridge closes to prevent in-flight message errors.
- **Scaling**: InstanceManager with pools. Worker group competing consumers. Static + Threshold strategies.

## Guides

Conceptual documentation — what things are, when to use them, how to choose.

- [Agents](guides/agents.md) — Agent config, generate/stream, sub-agents, supervisor pattern, delegation, dynamic config, memory access.
- [Storage](guides/storage.md) — Storage providers, embedded SQLite, memory backends, vector stores.
- [Workspace](guides/workspace.md) — Filesystem, sandbox, search, skills, LSP, tool remapping, dynamic factories.
- [Evals](guides/evals.md) — Scorers, batch evaluation with runEvals(), pre-built scorers (rule-based + LLM).
- [Processors](guides/processors.md) — Built-in input/output middleware: security, PII, moderation, token limiting, tool search.
- [Harness](guides/harness.md) — Orchestrator for agent execution, threads, modes, tool approval, events, display state.
- [WASM](guides/wasm.md) — WASM shard model, deployment, host functions, AS library.

## API Reference

Technical reference — Go config structs, TypeScript constructors, method signatures, error cases.

- [Storage API](api/storage/README.md) — `StorageConfig`, `LibSQLStore`, `LibSQLVector`, `AddStorage`/`RemoveStorage`
- [Workspace API](api/workspace/README.md) — `Workspace`, `LocalFilesystem`, `LocalSandbox`, search, tools config, LSP
- [Harness API](api/harness/README.md) — `HarnessConfig`, `InitHarness`, 48 methods, 41 events, display state, permissions
- [WASM API](api/wasm/README.md) — Compile, run, deploy, undeploy, shard descriptors, host functions

## Deployment

All `.ts` code (agents, tools, workflows) runs in SES Compartments. The Kit API:

| Method | What it does |
|--------|-------------|
| `Kit.Deploy(ctx, source, code)` | Deploy code in a new isolated compartment |
| `Kit.Teardown(ctx, source)` | Remove compartment + all resources (subscriptions, agents, tools) |
| `Kit.Redeploy(ctx, source, code)` | Atomic swap — teardown then deploy |

Bus topics for deployment lifecycle:

| Topic | Direction |
|-------|-----------|
| `kit.deploy` | Request deployment |
| `kit.teardown` | Request teardown |
| `kit.redeploy` | Request atomic redeploy |
| `kit.list` | List deployed sources |

The SDK Client also exposes `Deploy()` and `Teardown()` methods, so plugins can trigger deployments.

Resource cleanup is automatic. When a compartment is torn down, all bus subscriptions registered by that source are removed, agents are unregistered from the agent registry, and tools are deregistered. The runtime tracks resources via a cleanup registry (`_resourceRegistry`).

### What was removed

`AgentRegisterMsg`, `AgentUnregisterMsg`, `ToolRegisterMsg`, and `ToolRegisterResp` are removed from the SDK. Agents and tools are no longer created by sending bus messages directly. Instead, deploy a `.ts` file that calls `agent()` or `createTool()` — the compartment handles registration.

## Plugin Development

The Plugin SDK lets you build out-of-process plugins in any Go module:

```go
p := sdk.New("your-org", "plugin-name", "1.0.0")
sdk.Tool(p, "my-tool", "Description", handleMyTool)
sdk.Event[MyEvent](p, "Event description")
p.Run()
```

Plugin manifest files now deploy in SES Compartments. The Kit loads the manifest, deploys its `.ts` entry points into isolated compartments, and manages their lifecycle.

Reference implementation: `../plugins/brainkit-plugin-cron/` (5 tools, 1 event, state persistence).

## Bus Topics

All operations flow through the bus. Current handlers:

| Namespace | Topics | Handler |
|-----------|--------|---------|
| `wasm.*` | compile, run, deploy, undeploy, list, get, remove, describe | WASM service |
| `tools.*` | call, list, resolve, register | Tool registry |
| `agents.*` | register, unregister, list, discover, get-status, set-status, request, message | Agent registry |
| `mcp.*` | listTools, callTool | MCP manager |
| `fs.*` | read, write, list, stat, delete, mkdir | Go-native (sandboxed) |
| `ai.*` | generate, embed, embedMany, generateObject | EvalTS → Mastra |
| `plugin.state.*` | get, set | Plugin state persistence |
| `kit.*` | deploy, teardown, redeploy, list | Compartment lifecycle |
| `memory.*` | createThread, getThread, listThreads, save, recall, deleteThread | Thread/message storage |
| `workflows.*` | run, resume, cancel, status | Workflow execution |
| `vectors.*` | upsert, query, createIndex, deleteIndex, listIndexes | Vector store operations |

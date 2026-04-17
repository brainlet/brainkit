# Brainkit Documentation

Brainkit is an embeddable Go runtime for AI agents. A Kit is a self-contained environment — one QuickJS runtime with Mastra + AI SDK + polyfills loaded, plus Go services (bus, tool registry, storage, plugins, networking). All agents, AI calls, workflows, and `.ts` code run inside a Kit.

## Architecture

- **Bus**: Async message bus with Send/Ask/On primitives. Interceptor pipeline. Worker groups. Address routing for cross-Kit communication. InProcessTransport dispatches handlers concurrently.
- **SES Compartments**: Each deployed `.ts` file runs in its own [SES](https://github.com/endojs/endo/tree/master/packages/ses) Compartment (220KB, bundled into agent-embed). `lockdown()` is called during Kit init to freeze all intrinsics. Compartments receive hardened endowments via `__kitEndowments(source)` with per-source wrappers. Agents and tools are created by deploying `.ts` code, not by sending bus messages.
- **Deployment**: `Kit.Deploy(ctx, source, code)` deploys code in an isolated compartment. `Kit.Teardown(ctx, source)` removes the compartment and all its resources (bus subscriptions, registered agents/tools). `Kit.Redeploy(ctx, source, code)` does an atomic swap (teardown + deploy).
- **Plugin SDK**: Registration-based (`sdk.New` + `sdk.Tool` + `sdk.On` + `sdk.Event`). Out-of-process gRPC. Auto-restart, backpressure, stream recovery, interceptors. Plugin manifest files deploy in Compartments.
- **Tool Registry**: `owner/pkg@version/tool` naming with 5-level resolution (exact → no-owner → no-version → bare → short name).
- **Workflows**: Mastra workflow engine with bus commands for lifecycle (start, resume, cancel, restart). Snapshots persist to configured storage. Crash recovery via `restartAllActiveWorkflowRuns()` on startup.
- **Agent Registry**: Bus topics for agent lifecycle (list, discover, status). Agents auto-unregister and clean up bus subscriptions on teardown.
- **Networking**: Kit-to-Kit over gRPC. Discovery (Static + Multicast LAN). NATS as alternative transport. On `Kit.Close()`, the network stops before the bridge closes to prevent in-flight message errors.

## Guides

Conceptual documentation — what things are, when to use them, how to choose.

- [Agents](guides/agents.md) — Agent config, generate/stream, sub-agents, supervisor pattern, delegation, dynamic config, memory access.
- [Storage](guides/storage.md) — Storage providers, embedded SQLite, memory backends, vector stores.
- [Workspace](guides/workspace.md) — Filesystem, sandbox, search, skills, LSP, tool remapping, dynamic factories.
- [Evals](guides/evals.md) — Scorers, batch evaluation with runEvals(), pre-built scorers (rule-based + LLM).
- [Processors](guides/processors.md) — Built-in input/output middleware: security, PII, moderation, token limiting, tool search.
- [Harness](guides/harness.md) — Orchestrator for agent execution, threads, modes, tool approval, events, display state.
- [Storage](guides/storage-and-memory.md) — Storage backends, agent memory, vectors, workflow persistence.

## API Reference

Technical reference — Go config structs, TypeScript constructors, method signatures, error cases.

- [Storage API](api/storage/README.md) — `StorageConfig`, `LibSQLStore`, `LibSQLVector`, `AddStorage`/`RemoveStorage`
- [Workspace API](api/workspace/README.md) — `Workspace`, `LocalFilesystem`, `LocalSandbox`, search, tools config, LSP
- [Harness API](api/harness/README.md) — `HarnessConfig`, `InitHarness`, 48 methods, 41 events, display state, permissions

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
| `tools.*` | call, list, resolve | Tool registry |
| `agents.*` | list, discover, get-status, set-status | Agent registry |
| `kit.*` | deploy, teardown, redeploy, list, deploy.file | Compartment lifecycle |
| `workflow.*` | start, startAsync, status, resume, cancel, list, runs, restart | Mastra workflows |
| `mcp.*` | listTools, callTool | MCP manager |
| `secrets.*` | set, get, delete, list, rotate | Secret management |
| `registry.*` | has, list, resolve | Provider registry |
| `packages.*` | search, install, remove, update, list, info | Package manager |
| `package.*` | deploy, teardown, redeploy, list, info | Package deployment |
| `plugin.*` | manifest, state.get/set, start, stop, restart, list, status | Plugin lifecycle |
| `metrics.get` | | Kit metrics |
| `trace.*` | get, list | Distributed tracing |
| `test.run` | | Test framework |
| `peers.*` | list, resolve | Peer discovery |
| `gateway.http.*` | route.add/remove/list, status | HTTP gateway |
| `workflows.*` | run, resume, cancel, status | Workflow execution |
| `vectors.*` | upsert, query, createIndex, deleteIndex, listIndexes | Vector store operations |

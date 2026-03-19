# Brainkit Documentation

Brainkit is the execution engine for [Brainlet](https://github.com/brainlet/brainlet). A Kit is a self-contained environment — one QuickJS runtime with Mastra + AI SDK + polyfills loaded, plus Go services (bus, tool registry, WASM, storage, plugins, networking). All agents, AI calls, workflows, and `.ts` code run inside a Kit.

## Architecture

- **Bus**: Async message bus with Send/Ask/On primitives. Interceptor pipeline. Worker groups. Address routing for cross-Kit communication.
- **Plugin SDK**: Registration-based (`sdk.New` + `sdk.Tool` + `sdk.On` + `sdk.Event`). Out-of-process gRPC. Auto-restart, backpressure, stream recovery, interceptors.
- **Tool Registry**: `owner/pkg@version/tool` naming with 5-level resolution (exact → no-owner → no-version → bare → short name).
- **WASM Shards**: Two modes (stateless/persistent). 10 host functions. AssemblyScript library with typed namespace functions.
- **Agent Registry**: 8 bus topics for agent lifecycle (register, discover, request, message, status).
- **Networking**: Kit-to-Kit over gRPC. Discovery (Static + Multicast LAN). NATS as alternative transport.
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

## Plugin Development

The Plugin SDK lets you build out-of-process plugins in any Go module:

```go
p := sdk.New("your-org", "plugin-name", "1.0.0")
sdk.Tool(p, "my-tool", "Description", handleMyTool)
sdk.Event[MyEvent](p, "Event description")
p.Run()
```

Reference implementation: `plugins/brainkit-plugin-cron/` (5 tools, 1 event, state persistence).

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

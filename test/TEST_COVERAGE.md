# Test Coverage Matrix

> 83 test functions across 33 test files + 2 test binaries.
> Real OpenAI API, real Podman containers (NATS, RabbitMQ, Redis, Postgres, pgvector). Zero mocks.

---

## Matrix 1: Domain Operations × API Surface

Every domain command in the catalog tested from every API surface.

| Domain | Operation | Go Direct (Kernel) | Go Direct (Node) | TS (.ts deploy) | WASM (invokeAsync) | Plugin (Node) |
|--------|-----------|:--:|:--:|:--:|:--:|:--:|
| **tools** | call | `go_direct_tools` | `go_direct_tools` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | resolve | `go_direct_tools` | `go_direct_tools` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | list | `go_direct_tools` | `go_direct_tools` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **fs** | read | `go_direct_fs` | `go_direct_fs` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | write | `go_direct_fs` | `go_direct_fs` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | list | `go_direct_fs` | `go_direct_fs` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | stat | `go_direct_fs` | `go_direct_fs` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | delete | `go_direct_fs` | `go_direct_fs` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | mkdir | `go_direct_fs` | `go_direct_fs` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **agents** | list | `go_direct_agents` | `go_direct_agents` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | discover | `go_direct_agents` | `go_direct_agents` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | get-status | `go_direct_agents` | `go_direct_agents` | — | — | — |
| | set-status | `go_direct_agents` | `go_direct_agents` | — | — | — |
| | request | `go_direct_agents` | — | — | — | — |
| | message | — | — | — | — | — |
| **ai** | generate | `go_direct_ai` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | embed | `go_direct_ai` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | embedMany | `go_direct_ai` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | generateObject | `go_direct_ai` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | stream | `streaming` | — | — | — | — |
| **memory** | createThread | `go_direct_memory` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | getThread | `go_direct_memory` | — | `surface_ts` | — | `surface_plugin` |
| | listThreads | `go_direct_memory` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | save | `go_direct_memory` | — | `surface_ts` | — | `surface_plugin` |
| | recall | `go_direct_memory` | — | — | — | `surface_plugin` |
| | deleteThread | `go_direct_memory` | — | `surface_ts` | — | `surface_plugin` |
| **workflows** | run | `go_direct_workflows` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | resume | — | — | — | — | — |
| | cancel | — | — | — | — | — |
| | status | — | — | — | — | — |
| **vectors** | createIndex | `go_direct_vectors` | — | — | `surface_wasmmod` | `surface_plugin` |
| | listIndexes | `go_direct_vectors` | — | — | `surface_wasmmod` | — |
| | deleteIndex | `go_direct_vectors` | — | — | — | — |
| | upsert | `go_direct_vectors`* | — | — | — | — |
| | query | `go_direct_vectors`* | — | — | — | — |
| **wasm** | compile | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | run | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | deploy | `go_direct_wasm` | `go_direct_wasm` | — | N/A | — |
| | undeploy | `go_direct_wasm` | `go_direct_wasm` | — | N/A | — |
| | describe | `go_direct_wasm` | `go_direct_wasm` | — | N/A | — |
| | list | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | get | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | remove | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| **kit** | deploy | `go_direct_kit` | `go_direct_kit` | — | — | `surface_plugin` |
| | teardown | `go_direct_kit` | `go_direct_kit` | — | — | `surface_plugin` |
| | redeploy | `go_direct_kit` | `go_direct_kit` | — | — | `surface_plugin` |
| | list | `go_direct_kit` | `go_direct_kit` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **mcp** | listTools | `go_direct_mcp` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | callTool | `go_direct_mcp` | — | — | — | — |
| **registry** | has | `registry_integration` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | list | `registry_integration` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | resolve | `registry_integration` | — | — | `surface_wasmmod` | `surface_plugin` |
| **plugin** | manifest | — | `node` | — | — | — |
| | state.get | — | `node` | — | — | — |
| | state.set | — | `node` | — | — | — |

`*` = Neon WebSocket driver limitation in QuickJS (createIndex works, upsert/query log errors)
`N/A` = WASM can't call WASM (same runtime)
`—` = Not tested (gap)

### Gaps in Matrix 1

| Gap | Why | Impact |
|-----|-----|--------|
| agents.get-status/set-status from TS/WASM/Plugin | Needs deployed agent (AI key) | Low — Go Direct covers it |
| agents.request from non-Go surfaces | Needs deployed agent + AI key | Low |
| agents.message from all surfaces | Not tested anywhere | **Medium** |
| ai.stream from non-Go surfaces | Streaming is a transport concern | Low |
| memory.recall from TS/WASM | Needs vector store for semantic recall | Low — Go Direct covers it |
| workflows.resume/cancel/status | Needs suspended workflow state | **Medium** |
| vectors.upsert/query full path | PgVector Neon driver QuickJS limitation | Known — not brainkit |
| wasm.deploy/undeploy/describe from TS/Plugin | Shard lifecycle is typically Go-initiated | Low |
| mcp.callTool from non-Go surfaces | Needs real MCP server running | Low |

---

## Matrix 2: Domain Operations × Transport Backend

Tests parameterized across all 6 Watermill backends.

| Domain | Test File | GoChannel | SQLite | NATS | AMQP | Redis | Postgres |
|--------|-----------|:---------:|:------:|:----:|:----:|:-----:|:--------:|
| tools (call, resolve, list) | `backend_matrix` | Y | Y | P | P | P | P |
| fs (write, read, mkdir, list, stat, delete) | `backend_matrix` | Y | Y | P | P | P | P |
| agents (list) | `backend_matrix` | Y | Y | P | P | P | P |
| kit (deploy, teardown) | `backend_matrix` | Y | Y | P | P | P | P |
| wasm (compile, run) | `backend_matrix` | Y | Y | P | P | P | P |
| async (correlationID) | `backend_matrix` | Y | Y | P | P | P | P |
| memory (all ops) | `go_direct_memory` | Y | Y | P | P | P | P |
| workflows (run) | `go_direct_workflows` | Y | Y | P | P | P | P |
| vectors (createIndex) | `go_direct_vectors` | Y | Y | P | P | P | P |

`Y` = Always runs | `P` = Runs when Podman available

---

## Matrix 3: Cross-Surface Pairs × Transport Backend

Every pair of API surfaces communicating across transports.

| Surface Pair | Test File | GoChannel | SQLite | NATS | AMQP | Redis | Postgres |
|--------------|-----------|:---------:|:------:|:----:|:----:|:-----:|:--------:|
| TS ↔ Go | `cross_ts_go` | Y | Y | P | P | P | P |
| WASM ↔ Go | `cross_wasm_go` | Y | Y | P | P | P | P |
| TS ↔ WASM | `cross_ts_wasmmod` | Y | Y | P | P | P | P |
| Plugin ↔ Go | `cross_plugin_go` | — | — | P | — | — | — |
| TS ↔ Plugin | `cross_ts_plugin` | — | — | P | — | — | — |
| WASM ↔ Plugin | `cross_wasmmod_plugin` | — | — | P | — | — | — |
| Kit-A ↔ Kit-B | `crosskit` | Y | Y | P | P | P | P |

Plugin cross-surface tests require NATS (subprocess needs network transport).

---

## Matrix 4: Chain Tests × Backend

Multi-surface chains where a request crosses 2+ surfaces.

| Chain | Test File | GoChannel | SQLite | NATS | AMQP | Redis | Postgres |
|-------|-----------|:---------:|:------:|:----:|:----:|:-----:|:--------:|
| Go → TS → WASM | `chain` | Y | Y | P | P | P | P |
| Go → TS → WASM (shard reply) | `chain` | Y | Y | P | P | P | P |

---

## Matrix 5: Infrastructure & Integration Tests

| Category | Test | Real Infrastructure | File |
|----------|------|---------------------|------|
| **Probing** | OpenAI live probe | Real OpenAI API | `probe` |
| | Bad API key detection | Real OpenAI API (401) | `probe` |
| | PgVector probe | Podman pgvector container | `probe` |
| | InMemory storage probe | In-process | `probe` |
| | Periodic ticker | Real OpenAI API | `probe` |
| **Vectors** | PgVector createIndex | Podman pgvector container | `go_direct_vectors` |
| **MCP** | listTools + callTool | testmcp binary (stdio) | `go_direct_mcp` |
| **Plugin subprocess** | Full e2e | Podman NATS + testplugin binary | `plugin_subprocess` |
| **Transport** | Pub/sub compliance | Per-backend containers | `transport_compliance` |
| **Logging** | TS Compartment tags | In-process | `log_handler` |
| | WASM module tags | In-process | `log_handler` |
| **Registry** | Go-side CRUD | In-process | `registry_integration` |
| | JS bridge (has/list/resolve) | In-process | `registry_integration` |
| | Deployed .ts access | In-process | `registry_integration` |

---

## Matrix 6: E2E Scenarios

| Scenario | What it proves | File |
|----------|---------------|------|
| Tool pipeline | Go registers tool → .ts deploys → tool callable → teardown | `e2e_scenarios` |
| Deploy lifecycle | deploy → list → redeploy → teardown → verify gone | `e2e_scenarios` |
| Multi-domain | fs.write → fs.read → tools.call → fs.write → verify | `e2e_scenarios` |
| WASM shard lifecycle | compile → deploy (persistent) → 5 events → state accumulates → undeploy → remove | `e2e_scenarios` |
| Concurrent operations | 3 concurrent PublishAwait tool calls | `e2e_scenarios` |
| Async patterns | correlationID filtering, 10 concurrent PublishAwait, context cancellation | `async` |
| WASM invokeAsync | tools.call callback, tools.list callback, unknown topic error callback | `wasm_invokeAsync` |
| WASM reply + state | shard reply(), persistent counter across 3 invocations | `wasm_reply` |
| Plugin in-process | Node as sdk.Runtime — list tools, call tool, fs, deploy/teardown, async subscribe | `plugin_inprocess` |
| Streaming | ai.stream → StreamChunk messages with sequential seq numbers | `streaming` |

---

## Test File Index

| File | Tests | Surfaces | Backends | Infra |
|------|-------|----------|----------|-------|
| `go_direct_tools_test.go` | 5 | Kernel, Node | default | — |
| `go_direct_fs_test.go` | 9 | Kernel, Node | default | — |
| `go_direct_agents_test.go` | 7 | Kernel, Node | default | OpenAI (agent deploy) |
| `go_direct_kit_test.go` | 5 | Kernel, Node | default | — |
| `go_direct_wasm_test.go` | 9 | Kernel, Node | default | — |
| `go_direct_ai_test.go` | 4 | Kernel | default | OpenAI |
| `go_direct_memory_test.go` | 5 | Kernel | all 6 | — |
| `go_direct_workflows_test.go` | 1 | Kernel | all 6 | — |
| `go_direct_vectors_test.go` | 3 | Kernel | all 6 | Podman pgvector |
| `go_direct_mcp_test.go` | 3 | Kernel | default | testmcp binary |
| `streaming_test.go` | 1 | Kernel | default | OpenAI |
| `async_test.go` | 4 | Kernel | default | — |
| `wasm_invokeAsync_test.go` | 3 | Kernel | default | — |
| `wasm_reply_test.go` | 2 | Kernel | default | — |
| `plugin_inprocess_test.go` | 5 | Node | memory | — |
| `plugin_subprocess_test.go` | 1 | Node | NATS | Podman NATS + binary |
| `e2e_scenarios_test.go` | 5 | Kernel | default | — |
| `transport_compliance_test.go` | 3 | — | memory, SQLite | — |
| `cross_ts_go_test.go` | 2 | Kernel | all 6 | — |
| `cross_wasm_go_test.go` | 2 | Kernel | all 6 | — |
| `cross_ts_wasmmod_test.go` | 2 | Kernel | all 6 | — |
| `cross_plugin_go_test.go` | 2 | Node | NATS | Podman NATS + binary |
| `cross_ts_plugin_test.go` | 2 | Node | NATS | Podman NATS + binary |
| `cross_wasmmod_plugin_test.go` | 2 | Node | NATS | Podman NATS + binary |
| `crosskit_test.go` | 2 | Kernel pair | all 6 | Podman (network backends) |
| `chain_test.go` | 2 | Kernel | all 6 | — |
| `backend_matrix_test.go` | 9 | Kernel | all 6 | Podman (network backends) |
| `log_handler_test.go` | 4 | Kernel | default | — |
| `registry_integration_test.go` | 6 | Kernel | default | — |
| `probe_test.go` | 7 | Kernel | default | OpenAI, Podman pgvector |
| `surface_ts_test.go` | 10 | Kernel | default | OpenAI |
| `surface_wasmmod_test.go` | 10 | Kernel | default | OpenAI |
| `surface_plugin_test.go` | 11 | Node | memory | OpenAI, Podman pgvector |

---

## Test Binaries

| Binary | Location | Purpose |
|--------|----------|---------|
| `testplugin` | `test/testplugin/main.go` | Echo + concat tools over NATS transport |
| `testmcp` | `test/testmcp/main.go` | MCP echo server (stdio transport) |

---

## Summary

- **83 test functions** across **33 test files**
- **4 API surfaces**: Go Direct (Kernel/Node), TS (.ts deploy), WASM (invokeAsync), Plugin (Node)
- **6 transport backends**: GoChannel, SQLite, NATS, AMQP, Redis, Postgres
- **12 domains**: tools, fs, agents, ai, memory, workflows, vectors, wasm, kit, mcp, registry, plugin
- **6 cross-surface pairs** + 2 chain tests + 2 cross-Kit tests
- **Real infrastructure**: OpenAI API, Podman containers (NATS, RabbitMQ, Redis, Postgres, pgvector)

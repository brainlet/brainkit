# Test Coverage Matrix

> **87 test functions** across **33 test files** + 2 test binaries.
> Real OpenAI API, real Podman containers (NATS, RabbitMQ, Redis, Postgres, pgvector). Zero mocks.

---

## Matrix 1: Domain Operations × API Surface

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
| | get-status | `go_direct_agents` | `go_direct_agents` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | set-status | `go_direct_agents` | `go_direct_agents` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | request | `go_direct_agents` | `go_direct_agents` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | message | `go_direct_agents` | `go_direct_agents` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **ai** | generate | `go_direct_ai` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | embed | `go_direct_ai` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | embedMany | `go_direct_ai` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | generateObject | `go_direct_ai` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | stream | `streaming` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **memory** | createThread | `go_direct_memory` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | getThread | `go_direct_memory` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | listThreads | `go_direct_memory` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | save | `go_direct_memory` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | recall | `go_direct_memory` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | deleteThread | `go_direct_memory` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **workflows** | run | `go_direct_workflows` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | resume | `go_direct_workflows` | — | — | — | — |
| | cancel | `go_direct_workflows` | — | — | — | — |
| | status | `go_direct_workflows` | — | — | — | — |
| **vectors** | createIndex | `go_direct_vectors` | — | — | `surface_wasmmod` | `surface_plugin` |
| | listIndexes | `go_direct_vectors` | — | — | `surface_wasmmod` | — |
| | deleteIndex | `go_direct_vectors` | — | — | — | — |
| | upsert | `go_direct_vectors`* | — | — | — | — |
| | query | `go_direct_vectors`* | — | — | — | — |
| **wasm** | compile | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | run | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | deploy | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | undeploy | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | describe | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | list | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | get | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| | remove | `go_direct_wasm` | `go_direct_wasm` | `surface_ts` | N/A | `surface_plugin` |
| **kit** | deploy | `go_direct_kit` | `go_direct_kit` | — | `surface_wasmmod` | `surface_plugin` |
| | teardown | `go_direct_kit` | `go_direct_kit` | — | `surface_wasmmod` | `surface_plugin` |
| | redeploy | `go_direct_kit` | `go_direct_kit` | — | — | `surface_plugin` |
| | list | `go_direct_kit` | `go_direct_kit` | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **mcp** | listTools | `go_direct_mcp` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | callTool | `go_direct_mcp` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **registry** | has | `registry_integration` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | list | `registry_integration` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| | resolve | `registry_integration` | — | `surface_ts` | `surface_wasmmod` | `surface_plugin` |
| **plugin** | manifest | — | `node` | — | — | — |
| | state.get | — | `node` | — | — | — |
| | state.set | — | `node` | — | — | — |

`*` = PgVector Neon WebSocket driver limitation in QuickJS (createIndex works, upsert/query log errors — not a brainkit issue)
`N/A` = WASM can't call WASM (same runtime)
`—` = Not tested

### Remaining `—` cells

| Gap | Reason |
|-----|--------|
| ai/memory/workflows/vectors from Go Direct Node | Node delegates to Kernel — identical code path, proven by Kernel tests |
| workflows.resume/cancel/status from TS/WASM/Plugin | Suspend state lives in JS runtime globals — surfaces can't chain run→suspend→resume across separate eval calls. Full lifecycle covered by Go Direct. |
| vectors.upsert/query full path | PgVector's @neondatabase/serverless WebSocket driver doesn't work in QuickJS. Handler wiring proven by createIndex. |
| kit.deploy/teardown from TS | TS code IS the deployed artifact. Kit lifecycle is a Go/WASM/Plugin concern. |

---

## Matrix 2: Domain Operations × Transport Backend

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

| Surface Pair | Test File | GoChannel | SQLite | NATS | AMQP | Redis | Postgres |
|--------------|-----------|:---------:|:------:|:----:|:----:|:-----:|:--------:|
| TS ↔ Go | `cross_ts_go` | Y | Y | P | P | P | P |
| WASM ↔ Go | `cross_wasm_go` | Y | Y | P | P | P | P |
| TS ↔ WASM | `cross_ts_wasmmod` | Y | Y | P | P | P | P |
| Plugin ↔ Go | `cross_plugin_go` | — | — | P | — | — | — |
| TS ↔ Plugin | `cross_ts_plugin` | — | — | P | — | — | — |
| WASM ↔ Plugin | `cross_wasmmod_plugin` | — | — | P | — | — | — |
| Kit-A ↔ Kit-B | `crosskit` | Y | Y | P | P | P | P |

Plugin cross-surface requires NATS (subprocess needs network transport).

---

## Matrix 4: Chain Tests × Backend

| Chain | Test File | GoChannel | SQLite | NATS | AMQP | Redis | Postgres |
|-------|-----------|:---------:|:------:|:----:|:----:|:-----:|:--------:|
| Go → TS → WASM | `chain` | Y | Y | P | P | P | P |
| Go → TS → WASM (shard reply) | `chain` | Y | Y | P | P | P | P |

---

## Matrix 5: Infrastructure & Integration

| Category | Test | Infrastructure | File |
|----------|------|----------------|------|
| **Probing** | OpenAI live HTTP probe | Real OpenAI API | `probe` |
| | Bad API key detection (401) | Real OpenAI API | `probe` |
| | PgVector JS runtime probe | Podman pgvector | `probe` |
| | InMemory storage probe | In-process JS | `probe` |
| | Periodic ticker (500ms) | Real OpenAI API | `probe` |
| | ProbeAll (all registered) | Real OpenAI API | `probe` |
| **Vectors** | PgVector createIndex/list/delete | Podman pgvector | `go_direct_vectors` |
| **MCP** | listTools + callTool (Go) | testmcp binary | `go_direct_mcp` |
| | listTools + callTool (TS) | testmcp binary | `surface_ts` |
| | listTools + callTool (WASM) | testmcp binary | `surface_wasmmod` |
| | listTools + callTool (Plugin) | testmcp binary | `surface_plugin` |
| **Plugin subprocess** | Full e2e with NATS | Podman NATS + testplugin | `plugin_subprocess` |
| **Transport** | Pub/sub compliance | Per-backend containers | `transport_compliance` |
| **Logging** | TS per-source tags | In-process | `log_handler` |
| | Multi-file isolation | In-process | `log_handler` |
| | WASM module tags | In-process | `log_handler` |
| | Nil handler default | In-process | `log_handler` |
| **Registry** | Go-side CRUD | In-process | `registry_integration` |
| | Runtime register/unregister | In-process | `registry_integration` |
| | JS bridge (has/list/resolve) | In-process | `registry_integration` |
| | Deployed .ts access | In-process | `registry_integration` |
| **Workflows** | Suspend → Status → Resume | Mastra suspend | `go_direct_workflows` |
| | Cancel/Status not found | In-process | `go_direct_workflows` |

---

## Matrix 6: E2E Scenarios

| Scenario | What it proves | File |
|----------|---------------|------|
| Tool pipeline | Go registers → .ts deploys → tool callable → teardown | `e2e_scenarios` |
| Deploy lifecycle | deploy → list → redeploy → teardown → gone | `e2e_scenarios` |
| Multi-domain | fs.write → fs.read → tools.call → fs.write → verify | `e2e_scenarios` |
| WASM shard lifecycle | compile → deploy persistent → 5 events → state → undeploy → remove | `e2e_scenarios` |
| Concurrent operations | 3 concurrent PublishAwait | `e2e_scenarios` |
| Async patterns | correlationID, 10 concurrent, cancel, subscribe cancel | `async` |
| WASM invokeAsync | tools.call, tools.list, unknown topic error callback | `wasm_invokeAsync` |
| WASM reply + state | shard reply(), persistent counter ×3 | `wasm_reply` |
| Plugin in-process | Node as sdk.Runtime — tools, fs, deploy, async | `plugin_inprocess` |
| Streaming | ai.stream → sequential StreamChunks | `streaming` |

---

## Test File Index

| # | File | Subtests | Surfaces | Backends | Infra |
|---|------|----------|----------|----------|-------|
| 1 | `go_direct_tools_test.go` | 6 ×2 | Kernel, Node | default | — |
| 2 | `go_direct_fs_test.go` | 10 ×2 | Kernel, Node | default | — |
| 3 | `go_direct_agents_test.go` | 9 ×2 | Kernel, Node | default | OpenAI |
| 4 | `go_direct_kit_test.go` | 5 ×2 | Kernel, Node | default | — |
| 5 | `go_direct_wasm_test.go` | 9 ×2 | Kernel, Node | default | — |
| 6 | `go_direct_ai_test.go` | 4 | Kernel | default | OpenAI |
| 7 | `go_direct_memory_test.go` | 5 | Kernel | all 6 | — |
| 8 | `go_direct_workflows_test.go` | 6 | Kernel | all 6 + default | — |
| 9 | `go_direct_vectors_test.go` | 3 | Kernel | all 6 | Podman pgvector |
| 10 | `go_direct_mcp_test.go` | 3 | Kernel | default | testmcp binary |
| 11 | `streaming_test.go` | 1 | Kernel | default | OpenAI |
| 12 | `async_test.go` | 4 | Kernel | default | — |
| 13 | `wasm_invokeAsync_test.go` | 3 | Kernel | default | — |
| 14 | `wasm_reply_test.go` | 2 | Kernel | default | — |
| 15 | `plugin_inprocess_test.go` | 5 | Node | memory | — |
| 16 | `plugin_subprocess_test.go` | 4 | Node | NATS | Podman NATS + binary |
| 17 | `e2e_scenarios_test.go` | 5 | Kernel | default | — |
| 18 | `transport_compliance_test.go` | 3 | — | memory, SQLite | — |
| 19 | `cross_ts_go_test.go` | 2 | Kernel | all 6 | — |
| 20 | `cross_wasm_go_test.go` | 2 | Kernel | all 6 | — |
| 21 | `cross_ts_wasmmod_test.go` | 2 | Kernel | all 6 | — |
| 22 | `cross_plugin_go_test.go` | 2 | Node | NATS | Podman NATS + binary |
| 23 | `cross_ts_plugin_test.go` | 2 | Node | NATS | Podman NATS + binary |
| 24 | `cross_wasmmod_plugin_test.go` | 2 | Node | NATS | Podman NATS + binary |
| 25 | `crosskit_test.go` | 2 | Kernel pair | all 6 | Podman (network) |
| 26 | `chain_test.go` | 2 | Kernel | all 6 | — |
| 27 | `backend_matrix_test.go` | 9 | Kernel | all 6 | Podman (network) |
| 28 | `log_handler_test.go` | 4 | Kernel | default | — |
| 29 | `registry_integration_test.go` | 6 | Kernel | default | — |
| 30 | `probe_test.go` | 7 | Kernel | default | OpenAI, Podman pgvector |
| 31 | `surface_ts_test.go` | 11 | Kernel | default | OpenAI, testmcp |
| 32 | `surface_wasmmod_test.go` | 11 | Kernel | default | OpenAI, testmcp |
| 33 | `surface_plugin_test.go` | 12 | Node | memory | OpenAI, pgvector, testmcp |

## Test Binaries

| Binary | Location | Purpose |
|--------|----------|---------|
| `testplugin` | `test/testplugin/main.go` | Echo + concat tools over NATS |
| `testmcp` | `test/testmcp/main.go` | MCP echo server (stdio) |

---

## Summary

- **87 test functions** across **33 test files**
- **4 API surfaces**: Go Direct (Kernel/Node), TS (.ts deploy), WASM (invokeAsync), Plugin (Node)
- **6 transport backends**: GoChannel, SQLite, NATS, AMQP, Redis, Postgres
- **13 domains**: tools, fs, agents, ai, memory, workflows, vectors, wasm, kit, mcp, registry, plugin, streaming
- **6 cross-surface pairs** + 2 chain tests + 2 cross-Kit tests
- **Real infrastructure**: OpenAI API, Podman (NATS, RabbitMQ, Redis, Postgres, pgvector), testmcp + testplugin binaries

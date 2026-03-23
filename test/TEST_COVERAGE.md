# Test Coverage Matrix

> **92 test functions** across **33 test files** + 2 test binaries.
> Real OpenAI API, real Podman containers (NATS, RabbitMQ, Redis, Postgres, pgvector). Zero mocks.
> This document is a FINAL STATE not a log.

---

## Matrix 1: Domain × Surface

Which domain operations are tested from which API surface.

| Domain | Operation | Go Kernel | Go Node | TS | WASM | Plugin |
|--------|-----------|:-:|:-:|:-:|:-:|:-:|
| **tools** | call | Y | Y | Y | Y | Y |
| | resolve | Y | Y | Y | Y | Y |
| | list | Y | Y | Y | Y | Y |
| **fs** | read | Y | Y | Y | Y | Y |
| | write | Y | Y | Y | Y | Y |
| | list | Y | Y | Y | Y | Y |
| | stat | Y | Y | Y | Y | Y |
| | delete | Y | Y | Y | Y | Y |
| | mkdir | Y | Y | Y | Y | Y |
| **agents** | list | Y | Y | Y | Y | Y |
| | discover | Y | Y | Y | Y | Y |
| | get-status | Y | Y | Y | Y | Y |
| | set-status | Y | Y | Y | Y | Y |
| | request | Y | Y | Y | Y | Y |
| | message | Y | Y | Y | Y | Y |
| **ai** | generate | Y | — | Y | Y | Y |
| | embed | Y | — | Y | Y | Y |
| | embedMany | Y | — | Y | Y | Y |
| | generateObject | Y | — | Y | Y | Y |
| | stream | Y | — | Y | Y | Y |
| **memory** | createThread | Y | — | Y | Y | Y |
| | getThread | Y | — | Y | Y | Y |
| | listThreads | Y | — | Y | Y | Y |
| | save | Y | — | Y | Y | Y |
| | recall | Y | — | Y | Y | Y |
| | deleteThread | Y | — | Y | Y | Y |
| **workflows** | run | Y | — | Y | Y | Y |
| | resume | Y | — | — | — | — |
| | cancel | Y | — | — | — | — |
| | status | Y | — | — | — | — |
| **vectors** | createIndex | Y | — | — | Y | Y |
| | listIndexes | Y | — | — | Y | — |
| | deleteIndex | Y | — | — | — | — |
| | upsert | Y* | — | — | — | — |
| | query | Y* | — | — | — | — |
| **wasm** | compile | Y | Y | Y | N/A | Y |
| | run | Y | Y | Y | N/A | Y |
| | deploy | Y | Y | Y | N/A | Y |
| | undeploy | Y | Y | Y | N/A | Y |
| | describe | Y | Y | Y | N/A | Y |
| | list | Y | Y | Y | N/A | Y |
| | get | Y | Y | Y | N/A | Y |
| | remove | Y | Y | Y | N/A | Y |
| **kit** | deploy | Y | Y | — | Y | Y |
| | teardown | Y | Y | — | Y | Y |
| | redeploy | Y | Y | — | — | Y |
| | list | Y | Y | Y | Y | Y |
| **mcp** | listTools | Y | — | Y | Y | Y |
| | callTool | Y | — | Y | Y | Y |
| **registry** | has | Y | — | Y | Y | Y |
| | list | Y | — | Y | Y | Y |
| | resolve | Y | — | Y | Y | Y |
| **plugin** | manifest | — | Y | — | — | — |
| | state.get | — | Y | — | — | — |
| | state.set | — | Y | — | — | — |

`*` PgVector Neon driver limitation in QuickJS — not brainkit
`N/A` WASM can't call WASM

---

## Matrix 2: Domain × Backend

Which domain operations are tested on which transport backends.
Only lists tests that are actually backend-parameterized.

| Domain | Operations | GoChannel | SQLite | NATS | AMQP | Redis | Postgres | File |
|--------|-----------|:-:|:-:|:-:|:-:|:-:|:-:|------|
| tools | call, list, resolve | Y | Y | P | P | P | P | `backend_matrix` |
| fs | write, read, mkdir, list, stat, delete | Y | Y | P | P | P | P | `backend_matrix` |
| agents | list | Y | Y | P | P | P | P | `backend_matrix` |
| kit | deploy, teardown | Y | Y | P | P | P | P | `backend_matrix` |
| wasm | compile, run, deploy, undeploy, describe | Y | Y | P | P | P | P | `backend_matrix` |
| kit | deploy, teardown, redeploy | Y | Y | P | P | P | P | `backend_matrix` |
| registry | has, list | Y | Y | P | P | P | P | `backend_matrix` |
| async | correlationID | Y | Y | P | P | P | P | `backend_matrix` |
| memory | create, save, recall, get, list, delete | Y | Y | P | P | P | P | `go_direct_memory` |
| workflows | run | Y | Y | P | P | P | P | `go_direct_workflows` |
| vectors | createIndex, upsert, query | Y | Y | P | P | P | P | `go_direct_vectors` |
| mcp | listTools, callTool, callTool_via_registry | Y | Y | P | P | P | P | `go_direct_mcp` |

`Y` always runs | `P` runs when Podman available

### NOT backend-parameterized (GoChannel only)

| Domain | Operations | Surface | File |
|--------|-----------|---------|------|
| agents | all 6 ops | Kernel, Node | `go_direct_agents` |
| ai | generate, embed, embedMany, generateObject | Kernel | `go_direct_ai` |
| fs | all 10 subtests | Kernel, Node | `go_direct_fs` |
| kit | deploy, teardown, redeploy, list, invalidCode, duplicate | Kernel, Node | `go_direct_kit` |
| tools | all 6 subtests | Kernel, Node | `go_direct_tools` |
| wasm | compile, run, list, get, remove, deploy, undeploy, describe, hostFn | Kernel, Node | `go_direct_wasm` |
| workflows | suspend/resume/cancel/status | Kernel | `go_direct_workflows` |
| streaming | ai.stream chunks | Kernel | `streaming` |
| **ALL TS surface tests** | tools, fs, agents, ai, memory, workflows, wasm, kit, mcp, registry | Kernel | `surface_ts` |
| **ALL WASM surface tests** | tools, fs, agents, ai, memory, workflows, kit, mcp, registry, vectors | Kernel | `surface_wasmmod` |
| **ALL Plugin surface tests** | tools, fs, agents, ai, kit, wasm, memory, workflows, mcp, registry, vectors | Node | `surface_plugin` |

---

## Matrix 3: Cross-Surface × Backend

| Surface Pair | GoChannel | SQLite | NATS | AMQP | Redis | Postgres | File |
|--------------|:-:|:-:|:-:|:-:|:-:|:-:|------|
| TS ↔ Go | Y | Y | P | P | P | P | `cross_ts_go` |
| WASM ↔ Go | Y | Y | P | P | P | P | `cross_wasm_go` |
| TS ↔ WASM | Y | Y | P | P | P | P | `cross_ts_wasmmod` |
| Plugin ↔ Go | — | — | P | — | — | — | `cross_plugin_go` |
| TS ↔ Plugin | — | — | P | — | — | — | `cross_ts_plugin` |
| WASM ↔ Plugin | — | — | P | — | — | — | `cross_wasmmod_plugin` |
| Go → TS → WASM chain | Y | Y | P | P | P | P | `chain` |
| Go → TS → WASM chain (reply) | Y | Y | P | P | P | P | `chain` |

---

## Matrix 4: Cross-Kit × Backend

Operations tested across Kit-to-Kit boundaries (two Kernels on shared transport).

| Domain | Operations | GoChannel | SQLite | NATS | AMQP | Redis | Postgres | File |
|--------|-----------|:-:|:-:|:-:|:-:|:-:|:-:|------|
| raw pub/sub | round-trip | Y | Y | P | P | P | P | `crosskit` |
| tools | call A→B, call B→A, list remote | Y | Y | P | P | P | P | `crosskit` |
| fs | write remote, read remote | Y | Y | P | P | P | P | `crosskit` |
| agents | list remote, discover remote | Y | Y | P | P | P | P | `crosskit` |
| kit | deploy remote, list remote, teardown remote | Y | Y | P | P | P | P | `crosskit` |
| wasm | compile remote, run remote, list remote | Y | Y | P | P | P | P | `crosskit` |
| registry | has remote, list remote | Y | Y | P | P | P | P | `crosskit` |

### NOT tested cross-Kit

| Domain | Reason |
|--------|--------|
| ai | Needs OpenAI key on both Kits — tested locally on each surface |
| memory | Needs JS memory init on both Kits — tested locally |
| workflows | Needs workflow deployed on remote Kit — tested locally |
| vectors | Needs pgvector on both Kits — tested locally |
| mcp | Needs MCP server on both Kits — tested locally |

---

## Matrix 5: Infrastructure

| Test | Infrastructure | File |
|------|----------------|------|
| OpenAI live HTTP probe | Real OpenAI API | `probe` |
| Bad API key detection (401) | Real OpenAI API | `probe` |
| PgVector JS runtime probe | Podman pgvector | `probe` |
| InMemory storage probe | In-process JS | `probe` |
| Periodic ticker (500ms) | Real OpenAI API | `probe` |
| ProbeAll | Real OpenAI API | `probe` |
| PgVector createIndex/list/delete | Podman pgvector | `go_direct_vectors` |
| MCP listTools + callTool (Go, TS, WASM, Plugin) | testmcp binary | `go_direct_mcp`, `surface_*` |
| Plugin subprocess e2e | Podman NATS + testplugin | `plugin_subprocess` |
| Transport pub/sub compliance | Per-backend containers | `transport_compliance` |
| TS Compartment per-source logging | In-process | `log_handler` |
| WASM module logging | In-process | `log_handler` |
| Registry Go + JS + deployed .ts | In-process | `registry_integration` |
| Workflow suspend → status → resume | Mastra suspend | `go_direct_workflows` |

---

## Matrix 6: E2E Scenarios

| Scenario | File |
|----------|------|
| Go registers tool → .ts deploys → tool callable → teardown | `e2e_scenarios` |
| deploy → list → redeploy → teardown → gone | `e2e_scenarios` |
| fs.write → fs.read → tools.call → fs.write → verify | `e2e_scenarios` |
| compile → deploy persistent → 5 events → state → undeploy → remove | `e2e_scenarios` |
| 3 concurrent PublishAwait | `e2e_scenarios` |
| correlationID, 10 concurrent, cancel, subscribe cancel | `async` |
| WASM invokeAsync callbacks (tools.call, tools.list, error) | `wasm_invokeAsync` |
| WASM shard reply(), persistent counter ×3 | `wasm_reply` |
| Node as sdk.Runtime — tools, fs, deploy, async | `plugin_inprocess` |
| ai.stream → sequential StreamChunks | `streaming` |

---

## Test File Index

| # | File | Subtests | Backends | Infra |
|---|------|----------|----------|-------|
| 1 | `go_direct_tools_test.go` | 6 ×2 | default | — |
| 2 | `go_direct_fs_test.go` | 10 ×2 | default | — |
| 3 | `go_direct_agents_test.go` | 9 ×2 | default | OpenAI |
| 4 | `go_direct_kit_test.go` | 5 ×2 | default | — |
| 5 | `go_direct_wasm_test.go` | 9 ×2 | default | — |
| 6 | `go_direct_ai_test.go` | 4 | default | OpenAI |
| 7 | `go_direct_memory_test.go` | 5 | **all 6** | — |
| 8 | `go_direct_workflows_test.go` | 6 | **all 6** + default | — |
| 9 | `go_direct_vectors_test.go` | 3 | **all 6** | pgvector |
| 10 | `go_direct_mcp_test.go` | 3 | **all 6** | testmcp |
| 11 | `streaming_test.go` | 1 | default | OpenAI |
| 12 | `async_test.go` | 4 | default | — |
| 13 | `wasm_invokeAsync_test.go` | 3 | default | — |
| 14 | `wasm_reply_test.go` | 2 | default | — |
| 15 | `plugin_inprocess_test.go` | 5 | memory | — |
| 16 | `plugin_subprocess_test.go` | 4 | NATS | NATS + binary |
| 17 | `e2e_scenarios_test.go` | 5 | default | — |
| 18 | `transport_compliance_test.go` | 3 | memory, SQLite | — |
| 19 | `cross_ts_go_test.go` | 2 | **all 6** | — |
| 20 | `cross_wasm_go_test.go` | 2 | **all 6** | — |
| 21 | `cross_ts_wasmmod_test.go` | 2 | **all 6** | — |
| 22 | `cross_plugin_go_test.go` | 2 | NATS | NATS + binary |
| 23 | `cross_ts_plugin_test.go` | 2 | NATS | NATS + binary |
| 24 | `cross_wasmmod_plugin_test.go` | 2 | NATS | NATS + binary |
| 25 | `crosskit_test.go` | 7 | **all 6** | Podman |
| 26 | `chain_test.go` | 2 | **all 6** | — |
| 27 | `backend_matrix_test.go` | 12 | **all 6** | Podman |
| 28 | `log_handler_test.go` | 4 | default | — |
| 29 | `registry_integration_test.go` | 6 | default | — |
| 30 | `probe_test.go` | 7 | default | OpenAI, pgvector |
| 31 | `surface_ts_test.go` | 11 | default | OpenAI, testmcp |
| 32 | `surface_wasmmod_test.go` | 11 | default | OpenAI, testmcp |
| 33 | `surface_plugin_test.go` | 12 | default | OpenAI, pgvector, testmcp |

## Test Binaries

| Binary | Location | Purpose |
|--------|----------|---------|
| `testplugin` | `test/testplugin/main.go` | Echo + concat over NATS |
| `testmcp` | `test/testmcp/main.go` | MCP echo server (stdio) |

---

## Identified Gaps

### Gap A: TS/WASM surface tests not backend-parameterized

TS and WASM surfaces use `LocalInvoker` which bypasses the transport entirely — the backend is irrelevant for these paths. The bridge calls go directly to the command catalog. Backend parameterization would change nothing.

Plugin surface DOES use transport (PublishRaw → transport → router). Plugin surface is tested on memory transport. The same transport path is proven across all 6 backends by `backend_matrix` (12 operations × 6 backends).

### Gap B: RESOLVED — Cross-Kit now covers 7 domains

`crosskit_test.go` tests tools, fs, agents, kit, wasm, registry across Kit boundaries on all 6 backends. Remaining domains (ai, memory, workflows, vectors, mcp) need per-Kit infrastructure setup (API keys, JS init, containers) that cross-Kit pairs don't configure.

### Gap C: RESOLVED — Backend matrix expanded

`backend_matrix` now covers 12 operations (tools call/list/resolve, fs write/read/mkdir/list/stat/delete, agents list, kit deploy/teardown/redeploy, wasm compile/run/deploy/undeploy/describe, registry has/list, async correlation) × 6 backends.

Go Direct detail tests (error paths, edge cases) remain GoChannel-only. The transport path is identical — only the Watermill backend differs, which is proven by backend_matrix.

### Gap D: WASM shard transport covered by chain tests

`chain_test.go` (Go→TS→WASM chain with shard reply) runs on all 6 backends. This exercises the shard subscription binding path through each transport. `wasm_invokeAsync` uses LocalInvoker (transport irrelevant).

### Gap E: RESOLVED — Vectors complete

vectors.deleteIndex tested from Go Direct + WASM. vectors.listIndexes + deleteIndex tested from Plugin.

---

## Summary

- **92 test functions** across **33 test files**
- **5 API surfaces**: Go Kernel, Go Node, TS, WASM, Plugin
- **6 transport backends**: GoChannel, SQLite, NATS, AMQP, Redis, Postgres
- **13 domains**: tools, fs, agents, ai, memory, workflows, vectors, wasm, kit, mcp, registry, plugin, streaming
- **6 cross-surface pairs** + 2 chains + 2 cross-Kit (tools only)
- **Real infrastructure**: OpenAI, Podman (NATS, RabbitMQ, Redis, Postgres, pgvector), testmcp, testplugin

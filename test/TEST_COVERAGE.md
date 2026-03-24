# Test Coverage Matrix

> **57 test functions** across **25 test files** + 2 test binaries.
> Real OpenAI API, real Podman containers (NATS, RabbitMQ, Redis, Postgres, pgvector). Zero mocks.
> All tests use pure async `sdk.Publish` + `sdk.SubscribeTo` pattern. Zero `PublishAwait`.
> This document is a FINAL STATE not a log.
>
> **Architecture note (2026-03-23):** AI/memory/workflow/vector domains removed from catalog.
> These are now .ts service territory — users write .ts bus services using AI SDK / Mastra directly.
> Surface tests (TS, WASM, Plugin) deleted — need rewriting for new module + bus.on pattern.

---

## Matrix 1: Infrastructure Domain × Surface

Which infrastructure operations are tested from which API surface.
Only infrastructure catalog commands remain — AI/memory/workflow/vector operations are now handled by .ts services.

| Domain | Operation | Go Kernel | Go Node |
|--------|-----------|:-:|:-:|
| **tools** | call | Y | Y |
| | resolve | Y | Y |
| | list | Y | Y |
| **fs** | read | Y | Y |
| | write | Y | Y |
| | list | Y | Y |
| | stat | Y | Y |
| | delete | Y | Y |
| | mkdir | Y | Y |
| **agents** | list | Y | Y |
| | discover | Y | Y |
| | get-status | Y | Y |
| | set-status | Y | Y |
| **kit** | deploy | Y | Y |
| | teardown | Y | Y |
| | redeploy | Y | Y |
| | list | Y | Y |
| **wasm** | compile | Y | — |
| | run | Y | — |
| | deploy | Y | — |
| | undeploy | Y | — |
| | describe | Y | — |
| | list | Y | — |
| | get | Y | — |
| | remove | Y | — |
| **mcp** | listTools | Y | — |
| | callTool | Y | — |
| **registry** | has | Y | — |
| | list | Y | — |
| | resolve | Y | — |

TS / WASM / Plugin surface tests: **DELETED** — need rewriting for new .ts service architecture.

---

## Matrix 2: Bus API Tests

New tests for the .ts service architecture bus layer.

| Test | What it verifies |
|------|-----------------|
| `JSPublishReturnsReplyTo` | `__go_brainkit_bus_publish` generates replyTo + correlationId |
| `JSEmitFireAndForget` | `__go_brainkit_bus_emit` publishes without replyTo |
| `JSReplyDoneFlag` | `__go_brainkit_bus_reply` with done=true/false metadata |
| `JSSubscribeReceivesMetadata` | JS subscribe handler gets payload + replyTo + correlationId + topic |
| `GoToJSRoundTrip` | Go → CustomMsg → JS handler → reply → Go receives |
| `DeployWithBusOn` | Deploy .ts → bus.on("greet") → send to ts.greeter.greet → reply |
| `StreamingChunks` | Deploy .ts → bus.on("stream") → msg.send chunks + msg.reply final |
| `KitRegisterAgentDiscovery` | kit.register("agent") → agents.list finds it → teardown removes it |

---

## Matrix 3: Domain × Backend

Infrastructure operations tested across transport backends.

| Domain | Operations | GoChannel | SQLite | NATS | AMQP | Redis | Postgres |
|--------|-----------|:-:|:-:|:-:|:-:|:-:|:-:|
| tools | call, list, resolve | Y | Y | P | P | P | P |
| fs | write, read, mkdir, list, stat, delete | Y | Y | P | P | P | P |
| agents | list | Y | Y | P | P | P | P |
| kit | deploy, teardown, redeploy | Y | Y | P | P | P | P |
| wasm | compile, run, deploy, undeploy, describe | Y | Y | P | P | P | P |
| registry | has, list | Y | Y | P | P | P | P |
| async | correlationID | Y | Y | P | P | P | P |

`Y` always runs | `P` runs when Podman available

---

## Matrix 4: Cross-Surface Tests

| Surface Pair | Status | File |
|--------------|--------|------|
| TS ↔ Go | needs rewrite | `cross_ts_go` |
| WASM ↔ Go | needs rewrite | `cross_wasm_go` |
| TS ↔ WASM | needs rewrite | `cross_ts_wasmmod` |
| Plugin ↔ Go | needs rewrite | `cross_plugin_go` |
| TS ↔ Plugin | needs rewrite | `cross_ts_plugin` |
| WASM ↔ Plugin | needs rewrite | `cross_wasmmod_plugin` |
| Go → TS → WASM chain | needs rewrite | `chain` |

Cross-surface tests exist but reference old WASM host function names. Need updating for `bus_publish`/`bus_emit`/`bus_on`.

---

## Matrix 5: Infrastructure

| Test | Infrastructure | File |
|------|----------------|---------|
| OpenAI live HTTP probe | Real OpenAI API | `probe` |
| Bad API key detection (401) | Real OpenAI API | `probe` |
| PgVector JS runtime probe | Podman pgvector | `probe` |
| InMemory storage probe | In-process JS | `probe` |
| Periodic ticker (500ms) | Real OpenAI API | `probe` |
| ProbeAll | Real OpenAI API | `probe` |
| PgVector createIndex/list/delete | Podman pgvector | `probe` |
| MCP listTools + callTool (Go) | testmcp binary | `go_direct_mcp` |
| Plugin subprocess e2e | Podman NATS + testplugin | `plugin_subprocess` |
| Transport pub/sub compliance | Per-backend containers | `transport_compliance` |
| TS Compartment per-source logging | In-process | `log_handler` |
| WASM module logging | In-process | `log_handler` |
| Registry Go + JS + deployed .ts | In-process | `registry_integration` |

---

## Matrix 6: E2E Scenarios

| Scenario | File |
|----------|------|
| Go registers tool → .ts deploys + kit.register → tool callable → teardown | `e2e_scenarios` |
| deploy → list → redeploy → teardown → gone | `e2e_scenarios` |
| fs.write → fs.read → tools.call → fs.write → verify | `e2e_scenarios` |
| compile → deploy persistent → 5 events → state → undeploy → remove | `e2e_scenarios` |
| 3 concurrent PublishAwait | `e2e_scenarios` |
| correlationID, 10 concurrent, cancel, subscribe cancel | `async` |
| WASM bus_publish callbacks (tools.call, tools.list, error) | `wasm_invokeAsync` |
| WASM shard reply(), persistent counter ×3 | `wasm_reply` |
| Node as sdk.Runtime — tools, fs, deploy, async | `plugin_inprocess` |

---

## Test File Index

| # | File | Test Functions | Backends | Infra |
|---|------|---------------|----------|-------|
| 1 | `go_direct_tools_test.go` | 1 (6 subtests ×2) | default | — |
| 2 | `go_direct_fs_test.go` | 1 (10 subtests ×2) | default | — |
| 3 | `go_direct_agents_test.go` | 1 (6 subtests ×2) | default | — |
| 4 | `go_direct_kit_test.go` | 1 (5 subtests ×2) | default | — |
| 5 | `go_direct_wasm_test.go` | 1 (9 subtests ×2) | default | — |
| 6 | `go_direct_mcp_test.go` | 1 (3 subtests) | default | testmcp |
| 7 | `bus_bridge_test.go` | 8 | default | — |
| 8 | `async_test.go` | 4 | default | — |
| 9 | `wasm_invokeAsync_test.go` | 3 | default | — |
| 10 | `wasm_reply_test.go` | 2 | default | — |
| 11 | `plugin_inprocess_test.go` | 1 (5 subtests) | memory | — |
| 12 | `plugin_subprocess_test.go` | 1 (4 subtests) | NATS | NATS + binary |
| 13 | `e2e_scenarios_test.go` | 5 | default | — |
| 14 | `transport_compliance_test.go` | 1 (3 subtests) | memory, SQLite | — |
| 15 | `cross_ts_go_test.go` | 1 | **all 6** | — |
| 16 | `cross_wasm_go_test.go` | 1 | **all 6** | — |
| 17 | `cross_ts_wasmmod_test.go` | 1 | **all 6** | — |
| 18 | `cross_plugin_go_test.go` | 1 | NATS | NATS + binary |
| 19 | `cross_ts_plugin_test.go` | 1 | NATS | NATS + binary |
| 20 | `cross_wasmmod_plugin_test.go` | 1 | NATS | NATS + binary |
| 21 | `chain_test.go` | 2 | **all 6** | — |
| 22 | `backend_matrix_test.go` | 1 (12 ops × backends) | **all 6** | Podman |
| 23 | `log_handler_test.go` | 4 | default | — |
| 24 | `registry_integration_test.go` | 6 | default | — |
| 25 | `probe_test.go` | 7 | default | OpenAI, pgvector |

## Test Binaries

| Binary | Location | Purpose |
|--------|----------|---------|
| `testplugin` | `test/testplugin/main.go` | Echo + concat over NATS |
| `testmcp` | `test/testmcp/main.go` | MCP echo server (stdio) |

---

## Deleted Tests (need rewriting for .ts service architecture)

| Old File | Reason | New Pattern |
|----------|--------|------------|
| `surface_ts_test.go` | Used old `ai.generate()` wrapper from "kit" | Deploy .ts with `bus.on` + new module imports |
| `surface_wasmmod_test.go` | Used `invokeAsync("ai.generate")` catalog cmd | WASM `bus_publish` to .ts services |
| `surface_plugin_test.go` | Used `sdk.Publish(AiGenerateMsg{})` | `sdk.Publish(CustomMsg{})` to .ts services |
| `crosskit_test.go` | Used removed AI/memory/workflow/vector types | Cross-Kit to .ts service mailboxes |
| `go_direct_ai_test.go` | AI catalog commands removed | .ts service handles AI |
| `go_direct_memory_test.go` | Memory catalog commands removed | .ts service handles memory |
| `go_direct_workflows_test.go` | Workflow catalog commands removed | .ts service handles workflows |
| `go_direct_vectors_test.go` | Vector catalog commands removed | .ts service handles vectors |
| `streaming_test.go` | StreamChunk type removed | .ts service streams via msg.send + msg.reply |

---

## Summary

- **57 test functions** across **25 test files**
- **4 API surfaces**: Go Kernel, Go Node, (TS/WASM/Plugin need rewrite)
- **6 transport backends**: GoChannel, SQLite, NATS, AMQP, Redis, Postgres
- **8 infrastructure domains**: tools, fs, agents (registry), kit, wasm, mcp, registry, bus
- **8 new bus API tests**: publish/emit/reply/subscribe/deploy+bus.on/streaming/kit.register
- **Real infrastructure**: OpenAI, Podman (NATS, RabbitMQ, Redis, Postgres, pgvector), testmcp, testplugin
- **9 test files deleted** — need rewriting for .ts service + module architecture

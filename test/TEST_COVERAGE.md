# Test Coverage Matrix

> **66 tests** across **7 suites** in **29 test files** + 2 test binaries.
> Real OpenAI API, real Podman containers (NATS, RabbitMQ, Redis, Postgres, pgvector). Zero mocks.
> All tests use pure async `sdk.Publish` + `sdk.SubscribeTo` pattern. Zero `PublishAwait`.
> Architecture: .ts service model with `bus.on` handlers that can `await` anything (generateText, fetch, tools.call).
> This document is a FINAL STATE not a log.

---

## Matrix 1: Infrastructure Domain x Surface

Which infrastructure operations are tested from which API surface.

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
| | deploy+list | Y | Y |
| **kit** | deploy | Y | Y |
| | teardown | Y | Y |
| | redeploy | Y | Y |
| | list | Y | Y |
| **wasm** | compile | Y | Y |
| | run | Y | Y |
| | deploy | Y | Y |
| | undeploy | Y | Y |
| | describe | Y | Y |
| | list | Y | Y |
| | get | Y | Y |
| | remove | Y | Y |
| **mcp** | listTools | Y | — |
| | callTool | Y | — |
| | callTool via registry | Y | — |
| **registry** | has | Y | — |
| | list | Y | — |
| | resolve | Y | — |

---

## Matrix 2: Bus API Tests

Tests for the .ts service architecture bus layer.

| Test | What it verifies | File |
|------|-----------------|------|
| `JSPublishReturnsReplyTo` | `__go_brainkit_bus_publish` generates replyTo + correlationId | `bus/api_test.go` |
| `JSEmitFireAndForget` | `__go_brainkit_bus_emit` publishes without replyTo | `bus/api_test.go` |
| `JSReplyDoneFlag` | `__go_brainkit_bus_reply` with done=true/false metadata | `bus/api_test.go` |
| `JSSubscribeReceivesMetadata` | JS subscribe handler gets payload + replyTo + correlationId + topic | `bus/api_test.go` |
| `GoToJSRoundTrip` | Go -> CustomMsg -> JS handler -> reply -> Go receives | `bus/api_test.go` |
| `DeployWithBusOn` | Deploy .ts -> bus.on("greet") -> send to ts.greeter.greet -> reply | `bus/api_test.go` |
| `StreamingChunks` | Deploy .ts -> bus.on("stream") -> msg.send chunks + msg.reply final | `bus/api_test.go` |
| `KitRegisterAgentDiscovery` | kit.register("agent") -> agents.list finds it -> teardown removes it | `bus/api_test.go` |
| `CorrelationIDFiltering` | Publish returns correlationID + replyTo, SubscribeTo receives response | `bus/async_test.go` |
| `MultipleInFlight` | 10 concurrent Publish calls each get their own ReplyTo topic | `bus/async_test.go` |
| `ContextCancellation` | Cancelled context doesn't hang Publish | `bus/async_test.go` |
| `SubscribeCancellation` | Calling unsub() stops message delivery | `bus/async_test.go` |

---

## Matrix 3: Surface Tests (TS)

Tests for the 4-module system (kit/ai/agent/compiler) and async bus.on with generateText/fetch.

| Test | What it verifies | Requires | File |
|------|-----------------|----------|------|
| `ModuleImports/KitModule` | bus, kit, model, tools, fs, mcp, output, registry endowments | — | `surface/ts_test.go` |
| `ModuleImports/AgentModule` | Agent, createTool, createWorkflow, createStep, Memory, etc. | — | `surface/ts_test.go` |
| `ModuleImports/AIModule` | generateText, streamText, generateObject, streamObject, embed, z | — | `surface/ts_test.go` |
| `DeployWithTool` | createTool + kit.register -> callable from Go | — | `surface/ts_test.go` |
| `DeployWithWorkflow` | createWorkflow + createStep chain -> run -> output | — | `surface/ts_test.go` |
| `DeployWithBusService` | bus.on("greet") -> mailbox topic -> async reply | — | `surface/ts_test.go` |
| `DeployWithStreaming` | bus.on -> msg.send() chunks -> msg.reply() final | — | `surface/ts_test.go` |
| `GenerateText_Real` | await generateText inside deployed .ts | OpenAI | `surface/ts_test.go` |
| `Agent_Real` | new Agent + generate inside deployed .ts | OpenAI | `surface/ts_test.go` |
| `AgentWithTool_Real` | Agent with createTool + generate | OpenAI | `surface/ts_test.go` |
| `BusServiceAsAIProxy` | bus.on -> await generateText -> msg.reply (canonical pattern) | OpenAI | `surface/ts_test.go` |
| `Diag_BusOn_AwaitPromiseResolve` | await Promise.resolve inside bus.on | — | `surface/async_diag_test.go` |
| `Diag_BusOn_AwaitSetTimeout` | await setTimeout inside bus.on (Go Schedule) | — | `surface/async_diag_test.go` |
| `Diag_BusOn_AwaitToolsCall` | await tools.call inside bus.on (bridgeRequestAsync) | — | `surface/async_diag_test.go` |
| `Diag_BusOn_AwaitFetch` | await fetch inside bus.on (real HTTP via Go goroutine) | — | `surface/async_diag_test.go` |
| `Diag_BusOn_AwaitGenerateText` | await generateText inside bus.on (full AI SDK + HTTP) | OpenAI | `surface/async_diag_test.go` |

---

## Matrix 4: Cross-Surface Tests

Every surface combination verified: TS, WASM, Go, Plugin.

| Test | Surfaces | Backends | File |
|------|----------|----------|------|
| `TS_deploys_tool_Go_calls_it` | TS -> Go | all 6 | `cross/ts_go_test.go` |
| `Go_registers_tool_TS_calls_via_deploy` | Go -> TS | all 6 | `cross/ts_go_test.go` |
| `WASM_calls_Go_tool_via_busPublish` | WASM -> Go | all 6 | `cross/wasm_go_test.go` |
| `Go_injects_event_WASM_shard_handles` | Go -> WASM | all 6 | `cross/wasm_go_test.go` |
| `WASM_calls_TS_registered_tool` | WASM -> TS | all 6 | `cross/ts_wasm_test.go` |
| `TS_deploys_WASM_shard_and_injects_event` | TS + WASM | all 6 | `cross/ts_wasm_test.go` |
| `Plugin_tool_called_from_Go` | Plugin -> Go | NATS | `cross/plugin_go_test.go` |
| `Go_tool_visible_in_list` | Go + Plugin | NATS | `cross/plugin_go_test.go` |
| `TS_calls_plugin_tool` | TS -> Plugin | NATS | `cross/ts_plugin_test.go` |
| `TS_deployed_tool_visible_alongside_plugin` | TS + Plugin | NATS | `cross/ts_plugin_test.go` |
| `WASM_calls_plugin_tool_via_busPublish` | WASM -> Plugin | NATS | `cross/wasm_plugin_test.go` |
| `WASM_and_plugin_tools_both_listed` | WASM + Plugin | NATS | `cross/wasm_plugin_test.go` |
| `Chain_Go_TS_WASM` | Go -> TS -> WASM | all 6 | `cross/chain_test.go` |
| `Chain_Go_TS_WASM_Reply` | Go -> WASM shard -> TS tool -> reply | all 6 | `cross/chain_test.go` |

---

## Matrix 5: Domain x Backend

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

## Matrix 6: E2E Scenarios

| Scenario | File |
|----------|------|
| Go registers tool -> .ts deploys + kit.register -> tool callable -> teardown | `e2e/scenarios_test.go` |
| deploy -> list -> redeploy -> teardown -> gone | `e2e/scenarios_test.go` |
| fs.write -> fs.read -> tools.call -> fs.write -> verify | `e2e/scenarios_test.go` |
| compile -> deploy persistent -> 5 events -> state -> undeploy -> remove | `e2e/scenarios_test.go` |
| 3 concurrent Publish tool calls | `e2e/scenarios_test.go` |

---

## Matrix 7: Infrastructure

| Test | Infrastructure | File |
|------|----------------|---------|
| OpenAI live HTTP probe | Real OpenAI API | `infra/probe_test.go` |
| Bad API key detection (401) | Real OpenAI API | `infra/probe_test.go` |
| Unregistered provider probe | In-process | `infra/probe_test.go` |
| InMemory storage probe | In-process JS | `infra/probe_test.go` |
| PgVector store probe | Podman pgvector | `infra/probe_test.go` |
| ProbeAll | Real OpenAI API | `infra/probe_test.go` |
| Periodic ticker (500ms) | Real OpenAI API | `infra/probe_test.go` |
| MCP listTools + callTool + via_registry (Go) | testmcp binary | `infra/mcp_test.go` |
| Plugin subprocess e2e | Podman NATS + testplugin | `plugin/subprocess_test.go` |
| Transport pub/sub compliance | Per-backend containers | `transport/compliance_test.go` |
| TS Compartment per-source logging | In-process | `infra/log_test.go` |
| TS Compartment multi-file logging | In-process | `infra/log_test.go` |
| WASM module logging | In-process | `infra/log_test.go` |
| Nil LogHandler default | In-process | `infra/log_test.go` |
| Registry Go + JS + deployed .ts | In-process | `infra/registry_test.go` |
| WASM bus_publish callbacks (tools.call, tools.list, error) | In-process | `infra/wasm_bus_test.go` |
| WASM shard reply() + persistent counter x3 | In-process | `infra/wasm_reply_test.go` |
| Node as sdk.Runtime (tools, fs, deploy, async) | In-process | `plugin/inprocess_test.go` |

---

## Test File Index

| # | Dir | File | Test Functions | Subtests | Backends | Infra |
|---|-----|------|:-:|----------|----------|-------|
| 1 | `infra/` | `tools_test.go` | 1 | 6 x2 surfaces | default | — |
| 2 | `infra/` | `fs_test.go` | 1 | 10 x2 surfaces | default | — |
| 3 | `infra/` | `agents_test.go` | 1 | 6 x2 surfaces | default | OpenAI (1 subtest) |
| 4 | `infra/` | `kit_test.go` | 1 | 5 x2 surfaces | default | — |
| 5 | `infra/` | `wasm_test.go` | 1 | 9 x2 surfaces | default | — |
| 6 | `infra/` | `mcp_test.go` | 1 | 3 x all backends | default | testmcp |
| 7 | `infra/` | `log_test.go` | 4 | — | default | — |
| 8 | `infra/` | `registry_test.go` | 6 | — | default | — |
| 9 | `infra/` | `probe_test.go` | 7 | — | default | OpenAI, pgvector |
| 10 | `infra/` | `wasm_bus_test.go` | 3 | — | default | — |
| 11 | `infra/` | `wasm_reply_test.go` | 2 | — | default | — |
| 12 | `bus/` | `api_test.go` | 8 | — | default | — |
| 13 | `bus/` | `async_test.go` | 4 | — | default | — |
| 14 | `surface/` | `ts_test.go` | 9 | 3 (ModuleImports) | default | OpenAI (4 tests) |
| 15 | `surface/` | `async_diag_test.go` | 5 | — | default | OpenAI (1 test) |
| 16 | `e2e/` | `scenarios_test.go` | 5 | — | default | — |
| 17 | `cross/` | `ts_go_test.go` | 1 | 2 x all backends | **all 6** | — |
| 18 | `cross/` | `wasm_go_test.go` | 1 | 2 x all backends | **all 6** | — |
| 19 | `cross/` | `ts_wasm_test.go` | 1 | 2 x all backends | **all 6** | — |
| 20 | `cross/` | `plugin_go_test.go` | 1 | 2 subtests | NATS | NATS + binary |
| 21 | `cross/` | `ts_plugin_test.go` | 1 | 2 subtests | NATS | NATS + binary |
| 22 | `cross/` | `wasm_plugin_test.go` | 1 | 2 subtests | NATS | NATS + binary |
| 23 | `cross/` | `chain_test.go` | 2 | — x all backends | **all 6** | — |
| 24 | `transport/` | `matrix_test.go` | 1 | 12 x all backends | **all 6** | Podman |
| 25 | `transport/` | `compliance_test.go` | 1 | 3 x all backends | **all 6** | Podman |
| 26 | `plugin/` | `inprocess_test.go` | 1 | 5 subtests | memory | — |
| 27 | `plugin/` | `subprocess_test.go` | 1 | 4 subtests | NATS | NATS + binary |

## Test Binaries

| Binary | Location | Purpose |
|--------|----------|---------|
| `testplugin` | `test/testplugin/main.go` | Echo + concat over NATS, plugin state |
| `testmcp` | `test/testmcp/main.go` | MCP echo server (stdio) |

---

## Summary

- **66 tests** across **29 test files** in **7 suites** (bus, infra, surface, e2e, cross, transport, plugin)
- **5 API surfaces**: Go Kernel, Go Node, TS (.ts services), WASM (AssemblyScript shards), Plugin (subprocess)
- **6 transport backends**: GoChannel, SQLite, NATS, AMQP, Redis, Postgres
- **8 infrastructure domains**: tools, fs, agents, kit, wasm, mcp, registry, bus
- **12 bus API tests**: publish/emit/reply/subscribe/deploy+bus.on/streaming/kit.register/correlationID/concurrency/cancel
- **16 surface tests**: 4-module imports, createTool, createWorkflow, bus.on services, streaming, generateText, Agent, async diagnostics (Promise.resolve -> setTimeout -> tools.call -> fetch -> generateText)
- **14 cross-surface tests**: TS<->Go, WASM<->Go, TS<->WASM, Plugin<->Go, Plugin<->TS, Plugin<->WASM, Go->TS->WASM chain
- **Real infrastructure**: OpenAI API, Podman (NATS, RabbitMQ, Redis, Postgres, pgvector), testmcp, testplugin
- **Shared helpers**: `internal/testutil/` (NewTestKernel, NewTestNode, NewTestKernelFull, AllBackends, BuildTestPlugin, BuildTestMCP)

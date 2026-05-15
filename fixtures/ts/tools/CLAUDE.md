# tools/ Fixtures

Tests the tool system: calling Go-registered tools from TypeScript, creating tools with Zod schemas via `createTool`, registering/listing/unregistering tools, and agent-driven tool invocation.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| abort-signal | no | none | Direct tool execution receives an already-aborted `AbortSignal` through execution context |
| agent-stream-data-persistence | yes | none | Live OpenAI-backed `agent.stream().fullStream` persists non-transient `data-*` tool chunks into real `Memory` + `InMemoryStore` |
| agent-stream-data-transient | yes | none | Live OpenAI-backed `agent.stream().fullStream` streams transient `data-*` chunks but persists only non-transient chunks into memory |
| agent-stream-subagent-writer | yes | none | Parent `agent.stream().fullStream` with live OpenAI-backed parent/sub-agents bubbles sub-agent tool `writer.custom()` chunks |
| agent-stream-writer | yes | none | `agent.stream().fullStream` with live OpenAI-backed tool use proves `writer.write()` emits wrapped `tool-output` chunks and `writer.custom()` bubbles direct `data-*` chunks |
| call-from-ts | no | none | Calls Go-registered "uppercase" tool via `tools.call()`; verifies text is uppercased |
| call-go-tool | no | none | Calls two Go-registered tools ("echo" and "add") via `tools.call()`; verifies echoed string and computed sum (42) |
| create-basic | no | none | `createTool` with Zod schema + `kit.register("tool", ...)` + `tools.call()` roundtrip; confirms sum is 42 |
| create-with-schema | yes | none | `createTool` with `outputSchema`; Agent uses the calculator tool to add 17+25 and confirms answer contains "42" |
| register-list | no | none | `kit.register("tool", ...)` then `tools.list()` confirms the tool appears by shortName |
| register-unregister | no | none | Register a tool, verify it appears in `tools.list()`, unregister it, verify it disappears |
| require-approval | no | none | `createTool` with `requireApproval`; direct execution can suspend and then approve |
| runtime-context | no | none | `RuntimeContext` alias and direct tool execution context access |
| with-output-schema | no | none | `createTool` with typed output schema and direct result validation |
| with-streaming | no | none | Direct tool execution receives a stream writer and can emit progress chunks |

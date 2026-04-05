# tools/ Fixtures

Tests the tool system: calling Go-registered tools from TypeScript, creating tools with Zod schemas via `createTool`, registering/listing/unregistering tools, and agent-driven tool invocation.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| call-from-ts | no | none | Calls Go-registered "uppercase" tool via `tools.call()`; verifies text is uppercased |
| call-go-tool | no | none | Calls two Go-registered tools ("echo" and "add") via `tools.call()`; verifies echoed string and computed sum (42) |
| create-basic | no | none | `createTool` with Zod schema + `kit.register("tool", ...)` + `tools.call()` roundtrip; confirms sum is 42 |
| create-with-schema | yes | none | `createTool` with `outputSchema`; Agent uses the calculator tool to add 17+25 and confirms answer contains "42" |
| register-list | no | none | `kit.register("tool", ...)` then `tools.list()` confirms the tool appears by shortName |
| register-unregister | no | none | Register a tool, verify it appears in `tools.list()`, unregister it, verify it disappears |

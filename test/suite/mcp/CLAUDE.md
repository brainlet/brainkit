# MCP Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests verify MCP server tool integration through bus messages. The test env must have an MCP server ("testmcp") configured with an "echo" tool. Tests use sdk.Publish + sdk.SubscribeTo pattern with typed message structs.

## Adding a test

1. Add function to tools.go
2. Register in run.go
3. Update TEST_MAP.md

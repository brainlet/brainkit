# mcp/ Fixtures

Tests the in-process MCP (Model Context Protocol) client: listing tools from connected MCP servers and calling MCP-provided tools from TypeScript.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| agent-with-mcp-tool | no | MCP server | `mcp.listTools()` returns tools from the test MCP server (expects "echo" tool); `mcp.callTool("test", "echo", ...)` invokes it |
| call-tool | no | MCP server | `mcp.listTools()` discovers available tools; `mcp.callTool("test", "echo", ...)` calls the echo tool and returns result |

# MCP Test Map

**Purpose:** Verifies the MCP (Model Context Protocol) tool listing and calling via bus messages, including registry integration
**Tests:** 3 functions across 1 file
**Entry point:** `mcp_test.go` → `Run(t, env)`
**Campaigns:** transport (all 5)

## Files

### tools.go — MCP tool operations

| Function | Purpose |
|----------|---------|
| testListTools | Publishes McpListToolsMsg and asserts the response contains a tool named "echo" from server "testmcp" |
| testCallTool | Publishes McpCallToolMsg targeting server="testmcp", tool="echo" with a message arg, asserts the result contains the echoed message and server name |
| testCallToolViaRegistry | Publishes a standard ToolCallMsg for "echo" (not MCP-specific), asserts the result contains the echoed message, verifying MCP tools are accessible through the unified registry |

## Cross-references

- **Campaigns:** `transport/{sqlite,nats,postgres,redis,amqp}_test.go`
- **Related domains:** tools (overlapping echo tool), registry (MCP tools registered in unified registry)
- **Fixtures:** none

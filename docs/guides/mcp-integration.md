# MCP Integration

brainkit connects to MCP (Model Context Protocol) servers and registers their tools in the shared tool registry. MCP tools become callable from Go, .ts, and plugins — same as any other tool.

## Configuration

MCP servers are configured on KernelConfig:

```go
k, err := kit.NewKernel(kit.KernelConfig{
    MCPServers: map[string]mcp.ServerConfig{
        "filesystem": {
            Command: "npx",
            Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
        },
        "remote-api": {
            URL: "http://localhost:8080/mcp",
        },
    },
})
```

Two transport types:

| Transport | Config | How it works |
|-----------|--------|-------------|
| Stdio | `Command` + `Args` + `Env` | Launches subprocess, communicates over stdin/stdout |
| HTTP | `URL` | Connects to HTTP/Streamable HTTP endpoint |

## What Happens at Init

During `NewKernel`, for each MCP server config:

1. Client connects (starts subprocess or opens HTTP connection)
2. MCP Initialize handshake (protocol version, capabilities)
3. `ListTools` fetches all tools the server provides
4. Each tool is registered in the shared tool registry as `mcp/<server>@1.0.0/<tool>`

```go
// kit/kernel.go — MCP setup
for name, serverCfg := range cfg.MCPServers {
    kernel.mcp.Connect(ctx, name, serverCfg)
    for _, tool := range kernel.mcp.ListToolsForServer(name) {
        fullName := toolreg.ComposeName("mcp", tool.ServerName, "1.0.0", tool.Name)
        kernel.Tools.Register(toolreg.RegisteredTool{
            Name:      fullName,
            ShortName: tool.Name,
            Executor:  &toolreg.GoFuncExecutor{Fn: func(ctx, callerID, input) {
                return kernel.mcp.CallTool(ctx, tool.ServerName, tool.Name, input)
            }},
        })
    }
}
```

After init, MCP tools appear in `tools.list` alongside Go-registered and .ts-registered tools.

## Calling MCP Tools

### From Go

```go
// Via the standard tool call mechanism
pr, _ := sdk.Publish(rt, ctx, messages.ToolCallMsg{
    Name:  "read_file",  // short name resolution finds mcp/filesystem@1.0.0/read_file
    Input: map[string]any{"path": "/tmp/test.txt"},
})
```

### From .ts

```typescript
// Via tools.call (resolves through the shared registry)
const result = await tools.call("read_file", { path: "/tmp/test.txt" });

// Via mcp.callTool (direct, specifying server name)
const result = await mcp.callTool("filesystem", "read_file", { path: "/tmp/test.txt" });
```

### Listing MCP Tools

```typescript
// All tools from all MCP servers
const allTools = mcp.listTools();

// Tools from a specific server
const fsTools = mcp.listTools("filesystem");
```

```go
pr, _ := sdk.Publish(rt, ctx, messages.McpListToolsMsg{Server: "filesystem"})
```

## Bus Topics

| Topic | Request | Response |
|-------|---------|----------|
| `mcp.listTools` | `{server?: string}` | `{tools: [{name, server, description}]}` |
| `mcp.callTool` | `{server: string, tool: string, args: any}` | `{result: any}` |

## The Test MCP Server

`test/testmcp/main.go` is a minimal MCP server used for testing:

```go
// test/testmcp/main.go
func main() {
    s := server.NewMCPServer("testmcp", "1.0.0")
    s.AddTool(mcp.Tool{
        Name:        "echo",
        Description: "echoes the input",
    }, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        return mcp.NewToolResultText(fmt.Sprintf("echo: %v", req.Params.Arguments)), nil
    })
    transport.ServeStdio(s)
}
```

Compiled and launched by the test helper:

```go
binary := testutil.BuildTestMCP(t)
k, _ := kit.NewKernel(kit.KernelConfig{
    MCPServers: map[string]mcp.ServerConfig{
        "test": {Command: binary},
    },
})
```

## Error Handling

```go
// No MCP servers configured
pr, _ := sdk.Publish(rt, ctx, messages.McpListToolsMsg{})
// Response error: ErrMCPNotConfigured ("mcp: no MCP servers configured")

// Server not connected
mcp.callTool("nonexistent", "tool", {})
// NotFoundError{Resource: "mcp-server", Name: "nonexistent"}
```

## Limitations

- MCP connections are established at Kernel init time. Runtime connect/disconnect requires Kernel restart.
- Only the tool primitive is supported. MCP resources, prompts, and sampling are not yet integrated.
- MCP server stdout (beyond the JSON-RPC protocol) is not captured.

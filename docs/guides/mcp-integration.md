# MCP Integration

`modules/mcp` wires external Model Context Protocol servers as
first-class bus tools. Once wired, every tool the MCP server
advertises becomes callable through the same `tools.call` topic as
Go- and plugin-registered tools.

Working end-to-end example:
[`examples/mcp/`](../../examples/mcp/).

## Wire an MCP server

```go
import (
    "github.com/brainlet/brainkit"
    mcpmod "github.com/brainlet/brainkit/modules/mcp"
)

kit, err := brainkit.New(brainkit.Config{
    Namespace: "mcp-demo",
    Transport: brainkit.Memory(),
    FSRoot:    "/tmp/mcp-demo",
    Modules: []brainkit.Module{
        mcpmod.New(map[string]mcpmod.ServerConfig{
            "fs": {
                Command: "npx",
                Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp/mcp-demo"},
            },
            "remote": {
                URL: "http://localhost:8080/mcp",
            },
        }),
    },
})
```

`mcpmod.New` (not `NewModule` — this one is spelled differently)
takes a `map[string]mcpmod.ServerConfig`. Both stdio subprocesses
and HTTP endpoints work:

| Transport | Config fields | Behaviour |
|---|---|---|
| Stdio | `Command`, `Args`, `Env` | Launches a subprocess and speaks JSON-RPC over stdin/stdout. |
| HTTP / Streamable HTTP | `URL`, `Headers` | Connects to the given endpoint. |

On Kit start the module dials every server, performs the MCP
`Initialize` handshake, fetches the tool catalog, and registers
each tool in the shared registry under the fully qualified name
`mcp/<server>@1.0.0/<tool>` with a short name of `<tool>`.

## List available tools

```go
list, err := brainkit.CallMcpListTools(kit, ctx,
    sdk.McpListToolsMsg{Server: "fs"},
    brainkit.WithCallTimeout(45*time.Second))
for _, t := range list.Tools {
    fmt.Printf("%s  %s\n", t.Name, t.Description)
}
```

Leave `Server` empty to list across every wired server.

## Call a tool

```go
res, err := brainkit.CallMcpCallTool(kit, ctx,
    sdk.McpCallToolMsg{
        Server: "fs",
        Tool:   "read_text_file",
        Args:   map[string]any{"path": "/tmp/mcp-demo/hello.txt"},
    },
    brainkit.WithCallTimeout(30*time.Second))
// res.Result is json.RawMessage containing the MCP tool output.
```

Or through the generic tool surface (short-name resolution finds
the MCP tool):

```go
resp, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{
    Name:  "read_text_file",
    Input: map[string]any{"path": "/tmp/mcp-demo/hello.txt"},
})
```

From `.ts`:

```ts
// Short name via the shared registry
const r = await bus.call("tools.call", {
    name:  "read_text_file",
    input: { path: "/tmp/mcp-demo/hello.txt" },
});

// Explicit server + tool via the mcp helper
const all = mcp.listTools();        // every MCP server
const fs  = mcp.listTools("fs");    // just the "fs" server
const r2  = await mcp.callTool("fs", "read_text_file", {
    path: "/tmp/mcp-demo/hello.txt",
});
```

## Use MCP tools in an agent

Because MCP tools appear in the registry, `tool(name)` picks them
up:

```ts
const agent = new Agent({
    name:         "fs-agent",
    model:        model("openai", "gpt-4o-mini"),
    instructions: "Use the MCP filesystem tools to answer.",
    tools: {
        read_text_file: tool("read_text_file"),
        list_directory: tool("list_directory"),
    },
});

const r = await agent.generate("What files are in /tmp/mcp-demo?");
```

## Bus commands

Both commands have generated Call wrappers:

| Command | Request | Response | Wrapper |
|---|---|---|---|
| `mcp.listTools` | `sdk.McpListToolsMsg` | `sdk.McpListToolsResp` | `brainkit.CallMcpListTools` |
| `mcp.callTool` | `sdk.McpCallToolMsg` | `sdk.McpCallToolResp` | `brainkit.CallMcpCallTool` |

## Errors

| Condition | Error |
|---|---|
| No MCP servers wired | `ErrMCPNotConfigured` |
| Unknown server name | `*sdk.NotFoundError{Resource: "mcp-server", Name: ...}` |
| Tool handshake fails during init | Kit construction returns an error. |

## Limitations

- MCP servers are wired at Kit start. Connect / disconnect at
  runtime requires a restart. (Multi-server graceful reconnection
  is on the roadmap.)
- Only the tool primitive is exposed. MCP resources, prompts, and
  sampling are not yet integrated.
- Stdio server output beyond JSON-RPC is not captured — configure
  your MCP server to log to stderr if you need it.

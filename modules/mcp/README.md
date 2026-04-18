# modules/mcp — stable

Wires external Model Context Protocol servers as first-class tools
inside a Kit. Each configured server is spawned (stdio) or
connected (URL) at Init; its tools register with the Kit's tool
registry.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/modules/mcp"
)

brainkit.New(brainkit.Config{
    Modules: []brainkit.Module{
        mcp.New(map[string]mcp.ServerConfig{
            "filesystem": {
                Command: "npx",
                Args:    []string{"@modelcontextprotocol/server-filesystem", "/"},
            },
            "remote": {URL: "https://mcp.example.com"},
        }),
    },
})
```

## Bus commands

- `mcp.list-tools` — list tools across all configured servers.
- `mcp.call-tool` — invoke a tool by fully-qualified name.
- `mcp.status` — server connection state snapshot.

Tools registered by MCP servers are addressable the same way as
in-process or plugin tools: `sdk.ToolCallMsg{Name: "filesystem/read-file", ...}`.

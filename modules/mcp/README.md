# modules/mcp — stable

Wires external Model Context Protocol servers as first-class tools
inside a Kit. Each configured server is spawned (stdio) or
connected (URL) at mount; its tools register with the Kit's tool
registry.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/module"
    "github.com/brainlet/brainkit/modules/mcp"
)

brainkit.New(brainkit.Config{
    Modules: []module.Module{
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

- `mcp.listTools` — list tools across all configured servers.
- `mcp.callTool` — invoke a tool by server and tool name.

The command surface mounts even when no servers are configured; in that shape
`mcp.listTools` returns an empty list and `mcp.callTool` returns a not-found
error for missing servers.

## Capabilities

- Requires: `tools` module for tool registration.
- Optional: `brainkit.core.lifecycle_debug_registry`.
- Provides: dynamically registered MCP-backed tools through the module tool host.

## Runtime resources

Owns `mcp.servers`, the external MCP server connections, and the tool leases
registered for discovered server tools.
When lifecycle debug is available, it registers a scoped `mcp` component with
closing state, configured server, connected server, and cached tool counts.

## Hot unmount

Unmounting closes all MCP server connections and unregisters MCP-owned tools and
`mcp.*` command handlers.
Close errors from external server transports are returned to the module scope;
connections that fail to close stay tracked so the close can be retried instead
of being silently dropped. Close is context-aware at the module boundary:
client closes that do not finish before the unmount context are reported through
the lifecycle debug `closingClients` count and remain tracked until their close
finishes or a later retry observes completion.

Tools registered by MCP servers are addressable the same way as
in-process or plugin tools: `toolmsg.ToolCallMsg{Name: "filesystem/read-file", ...}`.

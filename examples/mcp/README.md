# mcp

Wire the npx-published `@modelcontextprotocol/server-filesystem`
MCP server as tools on a brainkit Kit. The example seeds a
temp directory with a known file, lists tools the server
advertises, and reads the file through `mcp.callTool`.

## Prerequisites

`node` + `npx` on `PATH`. The first run may take a few seconds
while `npx` resolves and caches the MCP server package.

## Run

```sh
go run ./examples/mcp
```

Expected output (abridged):

```
MCP server 'fs' tool catalog:
  read_file  Read the complete contents of a file as text. DEPRECATED: …
  read_text_file  …
  list_directory  …
  …
  list_allowed_directories  …

mcp.callTool fs/read_text_file path=hello.txt:
"hello from an MCP-managed filesystem\n"
```

## What it shows

- `modules/mcp.New(map[string]ServerConfig{...})` spawns the
  configured servers at Kit init and registers their tools on
  the Kit's tool registry.
- `brainkit.CallMcpListTools(kit, ctx, {Server: "fs"})` queries
  a specific server's catalog over the bus.
- `brainkit.CallMcpCallTool(kit, ctx, {Server, Tool, Args})`
  invokes a tool. The example calls `read_text_file` with a
  full path to the seeded file.

## Adding more servers

Stack additional entries in the map passed to `mcpmod.New`:

```go
mcpmod.New(map[string]mcpmod.ServerConfig{
    "fs":     {Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "/data"}},
    "github": {URL: "https://mcp.github.internal"},
    "sql":    {Command: "./my-sql-mcp-server"},
})
```

Remote MCP servers authenticate via the URL itself; stdio
servers inherit the caller's environment.

## macOS gotcha

macOS `/var` is a symlink to `/private/var`; the MCP
filesystem server compares resolved paths against its allowed
directories list. The example calls `filepath.EvalSymlinks`
before handing the tempdir to the server to avoid the
`Access denied - path outside allowed directories` surprise.

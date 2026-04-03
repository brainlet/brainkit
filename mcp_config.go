package brainkit

import mcppkg "github.com/brainlet/brainkit/internal/mcp"

// MCPServerConfig defines an MCP server connection.
// Re-exported from internal/mcp for use by cmd/ and external consumers.
type MCPServerConfig = mcppkg.ServerConfig

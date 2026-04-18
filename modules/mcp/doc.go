// Package mcp wires external Model Context Protocol servers as
// first-class tools inside a Kit. Each configured MCP server
// (command/args/env for stdio, URL for remote) is spawned or
// connected at Init; its tools register with the Kit's
// ToolRegistry and route through mcp.* bus commands.
//
// Status: stable.
package mcp

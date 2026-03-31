package brainkit

import "fmt"

// ── Sentinel errors ──────────────────────────────────────────────────────────

// ErrNoWorkspace is returned when a filesystem operation is attempted
// but KernelConfig.FSRoot is not configured.
var ErrNoWorkspace = fmt.Errorf("workspace not configured")

// ErrMCPNotConfigured is returned when an MCP operation is attempted
// but no MCP servers were registered in KernelConfig.
var ErrMCPNotConfigured = fmt.Errorf("mcp: no MCP servers configured")

// ErrCommandTopic is returned when an event is emitted on a command topic.
var ErrCommandTopic = fmt.Errorf("brainkit: topic is a command topic, not an event topic")

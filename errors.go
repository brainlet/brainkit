package brainkit

import (
	"fmt"

	"github.com/brainlet/brainkit/internal/sdkerrors"
)

// ── Sentinel errors ──────────────────────────────────────────────────────────

// ErrNoWorkspace is returned when a filesystem operation is attempted
// but KernelConfig.FSRoot is not configured.
var ErrNoWorkspace error = &sdkerrors.NotConfiguredError{Feature: "workspace"}

// ErrMCPNotConfigured is returned when an MCP operation is attempted
// but no MCP servers were registered in KernelConfig.
var ErrMCPNotConfigured error = &sdkerrors.NotConfiguredError{Feature: "mcp"}

// ErrCommandTopic is returned when an event is emitted on a command topic.
var ErrCommandTopic = fmt.Errorf("brainkit: topic is a command topic, not an event topic")

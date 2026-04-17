package brainkit

import (
	"encoding/json"

	"github.com/brainlet/brainkit/internal/engine"
	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/types"
)

// ── Common types ─────────────────────────────────────────────────────────────

// LogEntry is a tagged log entry from a .ts Compartment or the runtime.
type LogEntry = types.LogEntry

// ResourceInfo describes a tracked resource (tool, agent, workflow, etc.).
type ResourceInfo = types.ResourceInfo

// Result is the return from sandbox eval.
type Result = types.Result

// RetryPolicy configures retry behavior for failed bus handlers.
type RetryPolicy = types.RetryPolicy

// PluginConfig configures a plugin subprocess.
type PluginConfig = types.PluginConfig

// ScheduleConfig configures a scheduled bus message.
type ScheduleConfig = types.ScheduleConfig

// MCPServerConfig configures an MCP tool server connection.
type MCPServerConfig = types.MCPServerConfig

// DiscoveryConfig configures cross-Kit peer discovery.
type DiscoveryConfig = types.DiscoveryConfig

// PeerConfig configures a known peer for static discovery.
type PeerConfig = types.PeerConfig

// (Module is now defined in module.go — keep the engine package imported
// for other aliases below.)

// Package describes a deployment unit. Build via PackageInline / PackageFromFile / PackageFromDir.
type Package struct {
	Name    string            `json:"name"`
	Version string            `json:"version,omitempty"`
	Entry   string            `json:"entry,omitempty"`
	Files   map[string]string `json:"files,omitempty"`

	// path is set by PackageFromDir / PackageFromFile to route the deploy
	// through the filesystem-bundling path. Internal.
	path string `json:"-"`
}

// ── Health & Metrics ────────────────────────────────────────────────────────

// HealthStatus is the full health report.
type HealthStatus = types.HealthStatus

// HealthCheck is a single health check result.
type HealthCheck = types.HealthCheck

// KernelMetrics is a point-in-time snapshot.
type KernelMetrics = types.KernelMetrics

// ── Tools ────────────────────────────────────────────────────────────────────

// TypedTool defines a tool with a typed Go struct for input.
type TypedTool[T any] = tools.TypedTool[T]

// RegisterTool registers a typed Go tool on a Kit runtime.
// Go-only: tool execution requires a Go function pointer, can't be a bus message.
func RegisterTool[T any](k *Kit, name string, tool TypedTool[T]) error {
	return engine.RegisterTool(k.kernel, name, tool)
}

// ── Client ───────────────────────────────────────────────────────────────────

// BusClient sends bus commands to a running brainkit instance over HTTP.
type BusClient = types.BusClient

// StreamEvent is one event from the NDJSON stream.
type StreamEvent = types.StreamEvent

// NewClient creates a BusClient that connects to a running instance over HTTP.
func NewClient(baseURL string) *BusClient {
	return types.NewClient(baseURL)
}

// ── Embedded .d.ts (for CLI scaffolding) ─────────────────────────────────────

var (
	KitDTS      = engine.KitDTS
	AiDTS       = engine.AiDTS
	AgentDTS    = engine.AgentDTS
	BrainkitDTS = engine.BrainkitDTS
	GlobalsDTS  = engine.GlobalsDTS
)

// ── Error types ──────────────────────────────────────────────────────────────

var (
	ErrCommandTopic     = types.ErrCommandTopic
)

// ── Encoding helper ──────────────────────────────────────────────────────────

// MustJSON marshals v to json.RawMessage, panics on error.
func MustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

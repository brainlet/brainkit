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

// RegistryConfig configures a plugin registry source.
type RegistryConfig = types.RegistryConfig

// Package describes a deployment unit for DeployPackage.
type Package struct {
	Name    string            `json:"name"`
	Version string            `json:"version,omitempty"`
	Entry   string            `json:"entry,omitempty"`
	Files   map[string]string `json:"files"`
}

// ── Health & Metrics ────────────────────────────────────────────────────────

// HealthStatus is the full health report.
type HealthStatus = types.HealthStatus

// HealthCheck is a single health check result.
type HealthCheck = types.HealthCheck

// KernelMetrics is a point-in-time snapshot.
type KernelMetrics = types.KernelMetrics

// ── Scaling ──────────────────────────────────────────────────────────────────

// InstanceManager manages pools of runtime instances.
type InstanceManager = engine.InstanceManager

// PoolConfig configures an instance pool.
type PoolConfig = engine.PoolConfig

// StaticStrategy maintains a fixed instance count.
type StaticStrategy = engine.StaticStrategy

// ScalingDecision describes a scaling action.
type ScalingDecision = types.ScalingDecision

// ScalingStrategy evaluates metrics and pool state.
type ScalingStrategy = types.ScalingStrategy

// PoolInfo describes pool state.
type PoolInfo = types.PoolInfo

// PoolMode controls how pool instances relate to each other.
type PoolMode = engine.PoolMode

const (
	// PoolSharded gives each instance a different namespace (workload isolation).
	PoolSharded = engine.PoolSharded
	// PoolReplicated gives all instances the same namespace (horizontal scaling).
	PoolReplicated = engine.PoolReplicated
)

var (
	NewInstanceManager = engine.NewInstanceManager
	NewStaticStrategy  = engine.NewStaticStrategy
)

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
	ErrMCPNotConfigured = types.ErrMCPNotConfigured
	ErrCommandTopic     = types.ErrCommandTopic
)

// DefaultRegistry is the official brainlet plugin registry.
var DefaultRegistry = types.DefaultRegistry

// ── Encoding helper ──────────────────────────────────────────────────────────

// MustJSON marshals v to json.RawMessage, panics on error.
func MustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

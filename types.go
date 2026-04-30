package brainkit

import (
	"encoding/json"

	"github.com/brainlet/brainkit/internal/types"
)

// ErrorContext provides operation / component / source fields for
// non-fatal errors routed to the internal error handler.
type ErrorContext = types.ErrorContext

// ── Common types ─────────────────────────────────────────────────────────────

// LogEntry is a tagged log entry from a .ts Compartment or the runtime.
type LogEntry = types.LogEntry

// ResourceInfo describes a tracked resource (tool, agent, workflow, etc.).
type ResourceInfo = types.ResourceInfo

// Result is the return from sandbox eval.
type Result = types.Result

// RetryPolicy configures retry behavior for failed bus handlers.
type RetryPolicy = types.RetryPolicy

// ── Health & Metrics ────────────────────────────────────────────────────────

// HealthStatus is the full health report.
type HealthStatus = types.HealthStatus

// HealthCheck is a single health check result.
type HealthCheck = types.HealthCheck

// KernelMetrics is a point-in-time snapshot.
type KernelMetrics = types.KernelMetrics

// ── Client ───────────────────────────────────────────────────────────────────

// BusClient sends bus commands to a running brainkit instance over HTTP.
type BusClient = types.BusClient

// StreamEvent is one event from the NDJSON stream.
type StreamEvent = types.StreamEvent

// NewClient creates a BusClient that connects to a running instance over HTTP.
func NewClient(baseURL string) *BusClient {
	return types.NewClient(baseURL)
}

// ── Error types ──────────────────────────────────────────────────────────────

var (
	ErrCommandTopic = types.ErrCommandTopic
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

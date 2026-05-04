package brainkit

import "github.com/brainlet/brainkit/internal/types"

// ── Common types ─────────────────────────────────────────────────────────────

// LogEntry is a tagged log entry from a .ts Compartment or the runtime.
type LogEntry = types.LogEntry

// RetryPolicy configures retry behavior for failed bus handlers.
type RetryPolicy = types.RetryPolicy

// ── Health & Metrics ────────────────────────────────────────────────────────

// HealthStatus is the full health report.
type HealthStatus = types.HealthStatus

// HealthCheck is a single health check result.
type HealthCheck = types.HealthCheck

// ── Error types ──────────────────────────────────────────────────────────────

var (
	ErrCommandTopic = types.ErrCommandTopic
)

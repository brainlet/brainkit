package types

import (
	"encoding/json"
	"log/slog"
	"time"
)

// Result is the generic return from sandbox Eval.
type Result struct {
	Value json.RawMessage
	Text  string
}

// ResourceInfo describes a tracked resource in the Kit.
type ResourceInfo struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	CreatedAt int64  `json:"createdAt"`
}

// LogEntry is a tagged log entry from a .ts Compartment or the Kernel.
type LogEntry struct {
	Source  string
	Level   string
	Message string
	Time    time.Time
}

// ErrorContext provides context about where a non-fatal error occurred.
type ErrorContext struct {
	Operation string
	Component string
	Source    string
}

// InvokeErrorHandler calls the handler if non-nil, otherwise logs with default format.
func InvokeErrorHandler(handler func(error, ErrorContext), err error, ctx ErrorContext) {
	if handler != nil {
		handler(err, ctx)
		return
	}
	defaultErrorHandler(err, ctx)
}

func defaultErrorHandler(err error, ctx ErrorContext) {
	attrs := []slog.Attr{
		slog.String("component", ctx.Component),
		slog.String("operation", ctx.Operation),
		slog.Any("error", err),
	}
	if ctx.Source != "" {
		attrs = append(attrs, slog.String("source", ctx.Source))
	}
	slog.LogAttrs(nil, slog.LevelError, "non-fatal error", attrs...)
}

// HealthStatus is the full health report from Kernel.Health().
type HealthStatus struct {
	Healthy bool          `json:"healthy"`
	Status  string        `json:"status"`
	Uptime  time.Duration `json:"uptime"`
	Checks  []HealthCheck `json:"checks"`
}

// HealthCheck is a single health check result.
type HealthCheck struct {
	Name    string        `json:"name"`
	Healthy bool          `json:"healthy"`
	Latency time.Duration `json:"latency,omitempty"`
	Error   string        `json:"error,omitempty"`
	Details any           `json:"details,omitempty"`
}

// KernelMetrics is a point-in-time snapshot of internal Kernel state.
type KernelMetrics struct {
	ActiveHandlers    int64         `json:"activeHandlers"`
	ActiveDeployments int           `json:"activeDeployments"`
	ActiveSchedules   int           `json:"activeSchedules"`
	PumpCycles        int64         `json:"pumpCycles"`
	Uptime            time.Duration `json:"uptime"`
}

// MCPServerConfig defines an MCP server connection.
type MCPServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// ── Deploy Options ───────────────────────────────────────────────────────────

// DeployOption configures a Deploy call.
type DeployOption func(*DeployConfig)

// DeployConfig holds deploy options.
type DeployConfig struct {
	Role        string
	PackageName string
	Restoring   bool
}

// WithRestoring marks this Deploy as a restore from persistence.
func WithRestoring() DeployOption {
	return func(c *DeployConfig) { c.Restoring = true }
}

// WithRole assigns an RBAC role to the deployment.
func WithRole(role string) DeployOption {
	return func(c *DeployConfig) { c.Role = role }
}

// WithPackageName tags the deployment as part of a package.
func WithPackageName(name string) DeployOption {
	return func(c *DeployConfig) { c.PackageName = name }
}

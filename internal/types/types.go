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

	// Plugin metrics (only populated when Node is present)
	ActivePlugins int             `json:"activePlugins"`
	Plugins       []PluginMetrics `json:"plugins,omitempty"`

	// Bus metrics per topic
	Bus *MetricsSnapshot `json:"bus,omitempty"`

	// Transport reports low-cardinality transport/router health gauges.
	Transport *TransportMetrics `json:"transport,omitempty"`
}

// MetricsSnapshot is a point-in-time copy of bus metrics data.
type MetricsSnapshot struct {
	Published           map[string]int           `json:"published"`
	Handled             map[string]int           `json:"handled"`
	Errors              map[string]int           `json:"errors"`
	HandleDurationTotal map[string]time.Duration `json:"handleDurationTotal,omitempty"`
	HandleDurationMax   map[string]time.Duration `json:"handleDurationMax,omitempty"`
	HandleDurationCount map[string]int           `json:"handleDurationCount,omitempty"`
}

// TransportMetrics is the stable low-cardinality transport metrics view.
type TransportMetrics struct {
	Kind                   string `json:"kind,omitempty"`
	ActiveSubscriptions    int64  `json:"activeSubscriptions"`
	RouterHandlers         int    `json:"routerHandlers"`
	RouterStartedHandlers  int    `json:"routerStartedHandlers"`
	RouterStoppedHandlers  int    `json:"routerStoppedHandlers"`
	ActiveStreamHeartbeats int    `json:"activeStreamHeartbeats"`
}

// PluginMetrics describes a single plugin's runtime state.
type PluginMetrics struct {
	Name       string        `json:"name"`
	Healthy    bool          `json:"healthy"`
	ToolCalls  int64         `json:"toolCalls"`
	ToolErrors int64         `json:"toolErrors"`
	Uptime     time.Duration `json:"uptime"`
	LastPong   time.Time     `json:"lastPong,omitempty"`
}

// MCPServerConfig defines an MCP server connection.
type MCPServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// ── Deploy Options ───────────────────────────────────────────────────────────

// DeployArtifactKind describes the shape of code handed to the runtime.
type DeployArtifactKind string

const (
	// DeployArtifactSource is direct source code that the runtime may prepare
	// before evaluation.
	DeployArtifactSource DeployArtifactKind = "source"
	// DeployArtifactNormalizedJS is already-bundled JavaScript ready for the
	// runtime Compartment. The runtime must not transpile or bundle it again.
	DeployArtifactNormalizedJS DeployArtifactKind = "normalized_js"
)

// DeployOption configures a Deploy call.
type DeployOption func(*DeployConfig)

// DeployConfig holds deploy options.
type DeployConfig struct {
	PackageName  string
	Restoring    bool
	ArtifactKind DeployArtifactKind
}

// EffectiveArtifactKind returns the explicit artifact kind, defaulting to
// source for callers that do not opt into normalized artifacts.
func (c DeployConfig) EffectiveArtifactKind() DeployArtifactKind {
	if c.ArtifactKind != "" {
		return c.ArtifactKind
	}
	return DeployArtifactSource
}

// WithRestoring marks this Deploy as a restore from persistence.
func WithRestoring() DeployOption {
	return func(c *DeployConfig) { c.Restoring = true }
}

// WithPackageName tags the deployment as part of a package.
func WithPackageName(name string) DeployOption {
	return func(c *DeployConfig) { c.PackageName = name }
}

// WithNormalizedJS marks code as an already-bundled JavaScript artifact. The
// runtime may still use a .ts logical source name for addressing, but it must
// not run the package artifact back through TypeScript preparation.
func WithNormalizedJS() DeployOption {
	return func(c *DeployConfig) { c.ArtifactKind = DeployArtifactNormalizedJS }
}

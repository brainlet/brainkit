package audit

import (
	"encoding/json"
	"time"
)

// Verbosity controls which events are recorded.
type Verbosity int

const (
	// VerbosityNormal records lifecycle events, failures, security events, and tool calls.
	VerbosityNormal Verbosity = iota
	// VerbosityVerbose also records every bus command completion and periodic metric snapshots.
	VerbosityVerbose
)

// Recorder provides typed convenience methods for recording audit events.
// All methods are safe to call on a nil Recorder (no-op).
type Recorder struct {
	store     Store
	runtimeID string
	namespace string
	verbosity Verbosity
}

// RecorderConfig configures the Recorder.
type RecorderConfig struct {
	Store     Store
	RuntimeID string
	Namespace string
	Verbosity Verbosity
}

// NewRecorder creates a Recorder that writes to the given store.
func NewRecorder(store Store, runtimeID, namespace string) *Recorder {
	return &Recorder{store: store, runtimeID: runtimeID, namespace: namespace, verbosity: VerbosityNormal}
}

// NewRecorderWithConfig creates a Recorder with full configuration.
func NewRecorderWithConfig(cfg RecorderConfig) *Recorder {
	return &Recorder{store: cfg.Store, runtimeID: cfg.RuntimeID, namespace: cfg.Namespace, verbosity: cfg.Verbosity}
}

func (r *Recorder) record(category, typ, source string, data any, duration time.Duration, errMsg string) {
	if r == nil || r.store == nil {
		return
	}
	var payload json.RawMessage
	if data != nil {
		payload, _ = json.Marshal(data)
	}
	r.store.Record(Event{
		Timestamp: time.Now(),
		Category:  category,
		Type:      typ,
		Source:    source,
		RuntimeID: r.runtimeID,
		Namespace: r.namespace,
		Data:      payload,
		Duration:  duration,
		Error:     errMsg,
	})
}

// --- Plugin events ---

func (r *Recorder) PluginRegistered(name, owner, version string, toolCount int) {
	r.record("plugin", "plugin.registered", name, map[string]any{
		"owner": owner, "version": version, "tools": toolCount,
	}, 0, "")
}

func (r *Recorder) PluginStarted(name string, pid int) {
	r.record("plugin", "plugin.started", name, map[string]any{"pid": pid}, 0, "")
}

func (r *Recorder) PluginStopped(name, reason string) {
	r.record("plugin", "plugin.stopped", name, map[string]any{"reason": reason}, 0, "")
}

func (r *Recorder) PluginCrashed(name string, exitCode int, restarts int) {
	r.record("plugin", "plugin.crashed", name, map[string]any{
		"exitCode": exitCode, "restarts": restarts,
	}, 0, "plugin process exited unexpectedly")
}

func (r *Recorder) PluginHealthChanged(name, status string) {
	r.record("plugin", "plugin.health.changed", name, map[string]any{"status": status}, 0, "")
}

// --- Tool call events ---

func (r *Recorder) ToolCallCompleted(toolName, callerID string, duration time.Duration) {
	r.record("tools", "tools.call.completed", toolName, map[string]any{
		"caller": callerID,
	}, duration, "")
}

func (r *Recorder) ToolCallFailed(toolName, callerID string, duration time.Duration, err error) {
	r.record("tools", "tools.call.failed", toolName, map[string]any{
		"caller": callerID,
	}, duration, err.Error())
}

func (r *Recorder) ToolCallDenied(toolName, callerRuntimeID, reason string) {
	r.record("security", "tools.call.denied", toolName, map[string]any{
		"callerRuntimeID": callerRuntimeID, "reason": reason,
	}, 0, reason)
}

// --- Security events ---

// --- Secret events ---

func (r *Recorder) SecretSet(name, callerID string) {
	r.record("secrets", "secrets.set", name, map[string]any{"caller": callerID}, 0, "")
}

func (r *Recorder) SecretDeleted(name, callerID string) {
	r.record("secrets", "secrets.deleted", name, map[string]any{"caller": callerID}, 0, "")
}

func (r *Recorder) SecretRotated(name, callerID string) {
	r.record("secrets", "secrets.rotated", name, map[string]any{"caller": callerID}, 0, "")
}

// --- Deployment events ---

func (r *Recorder) Deployed(source string, resources int) {
	r.record("deploy", "kit.deployed", source, map[string]any{
		"resources": resources,
	}, 0, "")
}

func (r *Recorder) Teardown(source string) {
	r.record("deploy", "kit.teardown", source, nil, 0, "")
}

func (r *Recorder) DeployFailed(source string, err error) {
	r.record("deploy", "kit.deploy.failed", source, nil, 0, err.Error())
}

// --- Bus events ---

func (r *Recorder) BusHandlerFailed(topic string, err error) {
	r.record("bus", "bus.handler.failed", topic, nil, 0, err.Error())
}

func (r *Recorder) BusHandlerExhausted(topic string, attempts int) {
	r.record("bus", "bus.handler.exhausted", topic, map[string]any{
		"attempts": attempts,
	}, 0, "retry attempts exhausted")
}

// --- Health events ---

func (r *Recorder) HealthChanged(component, status string, healthy bool) {
	r.record("health", "health.changed", component, map[string]any{
		"status": status, "healthy": healthy,
	}, 0, "")
}

// --- Verbose-only events (bus command completions, metric snapshots) ---
// These only record when Verbosity >= VerbosityVerbose.

// BusCommandCompleted records a bus command that completed successfully.
// Only recorded in verbose mode — normal mode skips routine command completions.
func (r *Recorder) BusCommandCompleted(topic, callerID string, duration time.Duration) {
	if r == nil || r.verbosity < VerbosityVerbose {
		return
	}
	r.record("bus", "bus.command.completed", topic, map[string]any{
		"caller": callerID,
	}, duration, "")
}

// MetricsSnapshot records a periodic metrics snapshot.
// Only recorded in verbose mode.
func (r *Recorder) MetricsSnapshot(data any) {
	if r == nil || r.verbosity < VerbosityVerbose {
		return
	}
	r.record("metrics", "metrics.snapshot", "kernel", data, 0, "")
}

// IsVerbose returns true if verbose audit recording is enabled.
func (r *Recorder) IsVerbose() bool {
	return r != nil && r.verbosity >= VerbosityVerbose
}

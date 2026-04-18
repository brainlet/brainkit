package audit

import (
	"encoding/json"
	stdErrors "errors"
	"time"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
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

// SetStore attaches (or detaches) the underlying store. Safe on a nil
// Recorder; pass nil to make subsequent Record calls no-ops. The audit
// module calls this at Init/Close; no other caller should mutate the
// store mid-run, so no synchronization is used beyond the happens-before
// ordering provided by brainkit.New's module init phase.
func (r *Recorder) SetStore(s Store) {
	if r == nil {
		return
	}
	r.store = s
}

// SetVerbosity flips the recorder's verbosity tier. Same lifecycle
// expectations as SetStore — called once from the audit module's Init.
func (r *Recorder) SetVerbosity(v Verbosity) {
	if r == nil {
		return
	}
	r.verbosity = v
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

// recordErr records an audit event whose failure is a Go error. When err
// implements BrainkitError, its Code and Details are merged into the event
// Data under `errorCode` and `errorDetails` so the log remains
// machine-queryable. Plain errors collapse to INTERNAL_ERROR.
func (r *Recorder) recordErr(category, typ, source string, data map[string]any, duration time.Duration, err error) {
	if r == nil || r.store == nil || err == nil {
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	var bk sdkerrors.BrainkitError
	if stdErrors.As(err, &bk) {
		data["errorCode"] = bk.Code()
		if d := bk.Details(); len(d) > 0 {
			data["errorDetails"] = d
		}
	} else {
		data["errorCode"] = "INTERNAL_ERROR"
	}
	r.record(category, typ, source, data, duration, err.Error())
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
	r.recordErr("tools", "tools.call.failed", toolName, map[string]any{
		"caller": callerID,
	}, duration, err)
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
	r.recordErr("deploy", "kit.deploy.failed", source, nil, 0, err)
}

// --- Bus events ---

func (r *Recorder) BusHandlerFailed(topic string, err error) {
	r.recordErr("bus", "bus.handler.failed", topic, nil, 0, err)
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

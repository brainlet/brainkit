// Ported from: packages/core/src/observability/types/logging.ts
package types

import "time"

// ============================================================================
// Log Level
// ============================================================================

// LogLevel represents log severity levels.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// ============================================================================
// LoggerContext (API Interface)
// ============================================================================

// LoggerContext is the API interface for emitting structured logs.
// Logs are automatically correlated with the current span's trace/span IDs.
type LoggerContext interface {
	Debug(message string, data ...map[string]any)
	Info(message string, data ...map[string]any)
	Warn(message string, data ...map[string]any)
	Error(message string, data ...map[string]any)
	Fatal(message string, data ...map[string]any)
}

// ============================================================================
// ExportedLog (Event Bus Transport)
// ============================================================================

// ExportedLog is log data transported via the event bus.
// Must be JSON-serializable.
type ExportedLog struct {
	// Timestamp is when the log was emitted.
	Timestamp time.Time `json:"timestamp"`
	// Level is the log severity level.
	Level LogLevel `json:"level"`
	// Message is the human-readable log message.
	Message string `json:"message"`
	// Data is structured data associated with this log.
	Data map[string]any `json:"data,omitempty"`
	// TraceID for correlation (from current span).
	TraceID string `json:"traceId,omitempty"`
	// SpanID for correlation (from current span).
	SpanID string `json:"spanId,omitempty"`
	// Tags for filtering/categorization.
	Tags []string `json:"tags,omitempty"`
	// Metadata is user-defined metadata. Context fields are stored here.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ============================================================================
// LogEvent (Event Bus Event)
// ============================================================================

// LogEvent is a log event emitted to the ObservabilityBus.
type LogEvent struct {
	Type string      `json:"type"` // always "log"
	Log  ExportedLog `json:"log"`
}

// NewLogEvent creates a new LogEvent with the type set to "log".
func NewLogEvent(log ExportedLog) LogEvent {
	return LogEvent{
		Type: "log",
		Log:  log,
	}
}

// Ported from: packages/core/src/logger/noop-logger.ts
package logger

import (
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
)

// noopLoggerImpl is a logger that discards all log messages.
type noopLoggerImpl struct{}

// NoopLogger is a package-level singleton that implements IMastraLogger
// with all no-op methods. Mirrors the TS noopLogger export.
var NoopLogger IMastraLogger = &noopLoggerImpl{}

func (n *noopLoggerImpl) Debug(message string, args ...any)  {}
func (n *noopLoggerImpl) Info(message string, args ...any)   {}
func (n *noopLoggerImpl) Warn(message string, args ...any)   {}
func (n *noopLoggerImpl) Error(message string, args ...any)  {}
func (n *noopLoggerImpl) TrackException(err *mastraerror.MastraBaseError) {}

func (n *noopLoggerImpl) GetTransports() map[string]LoggerTransport {
	return make(map[string]LoggerTransport)
}

func (n *noopLoggerImpl) ListLogs(transportID string, params *ListLogsParams) (LogResult, error) {
	return LogResult{
		Logs:    []BaseLogMessage{},
		Total:   0,
		Page:    1,
		PerPage: 100,
		HasMore: false,
	}, nil
}

func (n *noopLoggerImpl) ListLogsByRunID(args *ListLogsByRunIDFullArgs) (LogResult, error) {
	return LogResult{
		Logs:    []BaseLogMessage{},
		Total:   0,
		Page:    1,
		PerPage: 100,
		HasMore: false,
	}, nil
}

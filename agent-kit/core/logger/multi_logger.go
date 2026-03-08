// Ported from: packages/core/src/logger/multi-logger.ts
package logger

import (
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
)

// MultiLogger composites multiple IMastraLogger instances, delegating
// all log calls to each underlying logger.
type MultiLogger struct {
	loggers []IMastraLogger
}

// NewMultiLogger creates a new MultiLogger from the given loggers.
func NewMultiLogger(loggers []IMastraLogger) *MultiLogger {
	return &MultiLogger{
		loggers: loggers,
	}
}

// Debug delegates to all underlying loggers.
func (m *MultiLogger) Debug(message string, args ...any) {
	for _, l := range m.loggers {
		l.Debug(message, args...)
	}
}

// Info delegates to all underlying loggers.
func (m *MultiLogger) Info(message string, args ...any) {
	for _, l := range m.loggers {
		l.Info(message, args...)
	}
}

// Warn delegates to all underlying loggers.
func (m *MultiLogger) Warn(message string, args ...any) {
	for _, l := range m.loggers {
		l.Warn(message, args...)
	}
}

// Error delegates to all underlying loggers.
func (m *MultiLogger) Error(message string, args ...any) {
	for _, l := range m.loggers {
		l.Error(message, args...)
	}
}

// TrackException delegates to all underlying loggers.
func (m *MultiLogger) TrackException(err *mastraerror.MastraBaseError) {
	for _, l := range m.loggers {
		l.TrackException(err)
	}
}

// GetTransports merges transports from all underlying loggers into a single map.
func (m *MultiLogger) GetTransports() map[string]LoggerTransport {
	transports := make(map[string]LoggerTransport)
	for _, l := range m.loggers {
		for k, v := range l.GetTransports() {
			transports[k] = v
		}
	}
	return transports
}

// ListLogs iterates through all loggers and returns the first non-empty result.
// If no logger has logs, returns an empty result.
func (m *MultiLogger) ListLogs(transportID string, params *ListLogsParams) (LogResult, error) {
	for _, l := range m.loggers {
		result, err := l.ListLogs(transportID, params)
		if err != nil {
			return result, err
		}
		if result.Total > 0 {
			return result, nil
		}
	}

	var page, perPage *int
	if params != nil {
		page = params.Page
		perPage = params.PerPage
	}
	return emptyLogResult(page, perPage), nil
}

// ListLogsByRunID iterates through all loggers and returns the first non-empty result.
// If no logger has logs, returns an empty result.
func (m *MultiLogger) ListLogsByRunID(args *ListLogsByRunIDFullArgs) (LogResult, error) {
	for _, l := range m.loggers {
		result, err := l.ListLogsByRunID(args)
		if err != nil {
			return result, err
		}
		if result.Total > 0 {
			return result, nil
		}
	}

	var page, perPage *int
	if args != nil {
		page = args.Page
		perPage = args.PerPage
	}
	return emptyLogResult(page, perPage), nil
}

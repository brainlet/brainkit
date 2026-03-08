// Ported from: packages/core/src/logger/default-logger.ts
package logger

import (
	"fmt"
	"os"
)

// ConsoleLogger is a logger that writes to stdout/stderr based on log level.
// It extends MastraLoggerBase (embedding the base struct) and implements IMastraLogger.
type ConsoleLogger struct {
	MastraLoggerBase
}

// ConsoleLoggerOptions holds configuration for creating a ConsoleLogger.
type ConsoleLoggerOptions struct {
	Name  string
	Level LogLevel
}

// NewConsoleLogger creates a new ConsoleLogger with the given options.
func NewConsoleLogger(opts *ConsoleLoggerOptions) *ConsoleLogger {
	var mastraOpts *MastraLoggerOptions
	if opts != nil {
		mastraOpts = &MastraLoggerOptions{
			Name:  opts.Name,
			Level: opts.Level,
		}
	}
	return &ConsoleLogger{
		MastraLoggerBase: NewMastraLoggerBase(mastraOpts),
	}
}

// Debug logs a debug message. Only outputs if level <= DEBUG.
func (c *ConsoleLogger) Debug(message string, args ...any) {
	if c.Level <= LogLevelDebug {
		fmt.Println(formatMessage(message, args...))
	}
}

// Info logs an info message. Only outputs if level <= INFO.
func (c *ConsoleLogger) Info(message string, args ...any) {
	if c.Level <= LogLevelInfo {
		fmt.Println(formatMessage(message, args...))
	}
}

// Warn logs a warning message. Only outputs if level <= WARN.
func (c *ConsoleLogger) Warn(message string, args ...any) {
	if c.Level <= LogLevelWarn {
		fmt.Println(formatMessage(message, args...))
	}
}

// Error logs an error message. Only outputs if level <= ERROR.
// Writes to stderr, matching the TS console.error behavior.
func (c *ConsoleLogger) Error(message string, args ...any) {
	if c.Level <= LogLevelError {
		fmt.Fprintln(os.Stderr, formatMessage(message, args...))
	}
}

// ListLogs returns an empty result set. ConsoleLogger does not store logs.
// This overrides MastraLoggerBase.ListLogs to match the TS ConsoleLogger
// which always returns empty regardless of transportId.
func (c *ConsoleLogger) ListLogs(transportID string, params *ListLogsParams) (LogResult, error) {
	var page, perPage *int
	if params != nil {
		page = params.Page
		perPage = params.PerPage
	}
	return emptyLogResult(page, perPage), nil
}

// ListLogsByRunID returns an empty result set. ConsoleLogger does not store logs.
func (c *ConsoleLogger) ListLogsByRunID(args *ListLogsByRunIDFullArgs) (LogResult, error) {
	var page, perPage *int
	if args != nil {
		page = args.Page
		perPage = args.PerPage
	}
	return emptyLogResult(page, perPage), nil
}

// formatMessage formats a log message with optional args, similar to how
// console.info(message, ...args) works in JS.
func formatMessage(message string, args ...any) string {
	if len(args) == 0 {
		return message
	}
	return fmt.Sprintf("%s %v", message, args)
}

// CreateLogger is a deprecated factory function that creates a ConsoleLogger.
// Deprecated: Use NewConsoleLogger directly instead.
func CreateLogger(opts *ConsoleLoggerOptions) *ConsoleLogger {
	logger := NewConsoleLogger(opts)
	logger.Warn(`CreateLogger is deprecated. Please use "NewConsoleLogger()" instead.`)
	return logger
}

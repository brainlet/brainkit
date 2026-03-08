// Ported from: packages/core/src/logger/transport.ts
package logger

import (
	"io"
	"time"
)

// BaseLogMessage represents a single log entry.
type BaseLogMessage struct {
	RunID    string   `json:"runId,omitempty"`
	Msg      string   `json:"msg"`
	Level    LogLevel `json:"level"`
	Time     time.Time `json:"time"`
	PID      int      `json:"pid"`
	Hostname string   `json:"hostname"`
	Name     string   `json:"name"`
}

// ListLogsParams holds parameters for listing logs.
type ListLogsParams struct {
	FromDate *time.Time
	ToDate   *time.Time
	LogLevel *LogLevel
	Filters  map[string]any
	ReturnPaginationResults *bool
	Page    *int
	PerPage *int
}

// ListLogsByRunIDArgs holds parameters for listing logs by run ID.
type ListLogsByRunIDArgs struct {
	RunID    string
	FromDate *time.Time
	ToDate   *time.Time
	LogLevel *LogLevel
	Filters  map[string]any
	Page     *int
	PerPage  *int
}

// LogResult holds the paginated result of a log query.
type LogResult struct {
	Logs    []BaseLogMessage `json:"logs"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"perPage"`
	HasMore bool             `json:"hasMore"`
}

// emptyLogResult returns a LogResult with empty logs and default pagination.
func emptyLogResult(page, perPage *int) LogResult {
	p := 1
	pp := 100
	if page != nil {
		p = *page
	}
	if perPage != nil {
		pp = *perPage
	}
	return LogResult{
		Logs:    []BaseLogMessage{},
		Total:   0,
		Page:    p,
		PerPage: pp,
		HasMore: false,
	}
}

// LoggerTransport is the interface for log transport backends.
// In the TS source, LoggerTransport extends Node.js Transform (objectMode).
// In Go, we replace that with io.Writer plus the list methods.
type LoggerTransport interface {
	io.Writer
	ListLogs(params *ListLogsParams) (LogResult, error)
	ListLogsByRunID(args *ListLogsByRunIDArgs) (LogResult, error)
}

// BaseTransport provides default implementations for LoggerTransport.
// Embed this in custom transports to get default no-op list methods.
type BaseTransport struct{}

// ListLogs returns an empty result set (default implementation).
func (b *BaseTransport) ListLogs(params *ListLogsParams) (LogResult, error) {
	var page, perPage *int
	if params != nil {
		page = params.Page
		perPage = params.PerPage
	}
	return emptyLogResult(page, perPage), nil
}

// ListLogsByRunID returns an empty result set (default implementation).
func (b *BaseTransport) ListLogsByRunID(args *ListLogsByRunIDArgs) (LogResult, error) {
	var page, perPage *int
	if args != nil {
		page = args.Page
		perPage = args.PerPage
	}
	return emptyLogResult(page, perPage), nil
}

// CustomTransport wraps an io.Writer with optional list function overrides.
type CustomTransport struct {
	writer          io.Writer
	listLogsFunc    func(params *ListLogsParams) (LogResult, error)
	listByRunIDFunc func(args *ListLogsByRunIDArgs) (LogResult, error)
}

// Write delegates to the underlying writer.
func (c *CustomTransport) Write(p []byte) (n int, err error) {
	return c.writer.Write(p)
}

// ListLogs calls the custom function if provided, otherwise returns empty result.
func (c *CustomTransport) ListLogs(params *ListLogsParams) (LogResult, error) {
	if c.listLogsFunc != nil {
		return c.listLogsFunc(params)
	}
	var page, perPage *int
	if params != nil {
		page = params.Page
		perPage = params.PerPage
	}
	return emptyLogResult(page, perPage), nil
}

// ListLogsByRunID calls the custom function if provided, otherwise returns empty result.
func (c *CustomTransport) ListLogsByRunID(args *ListLogsByRunIDArgs) (LogResult, error) {
	if c.listByRunIDFunc != nil {
		return c.listByRunIDFunc(args)
	}
	var page, perPage *int
	if args != nil {
		page = args.Page
		perPage = args.PerPage
	}
	return emptyLogResult(page, perPage), nil
}

// CreateCustomTransport creates a LoggerTransport from an io.Writer with optional
// list function overrides. Mirrors the TS createCustomTransport factory.
func CreateCustomTransport(
	writer io.Writer,
	listLogs func(params *ListLogsParams) (LogResult, error),
	listByRunID func(args *ListLogsByRunIDArgs) (LogResult, error),
) LoggerTransport {
	return &CustomTransport{
		writer:          writer,
		listLogsFunc:    listLogs,
		listByRunIDFunc: listByRunID,
	}
}

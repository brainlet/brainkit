// Ported from: packages/core/src/logger/logger.ts
package logger

import (
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
)

// ListLogsByRunIDFullArgs extends ListLogsByRunIDArgs with a TransportID field,
// matching the TS interface where transportId is part of the args object passed
// to IMastraLogger.listLogsByRunId.
type ListLogsByRunIDFullArgs struct {
	TransportID string
	ListLogsByRunIDArgs
}

// IMastraLogger defines the interface for all Mastra loggers.
type IMastraLogger interface {
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
	TrackException(err *mastraerror.MastraBaseError)
	GetTransports() map[string]LoggerTransport
	ListLogs(transportID string, params *ListLogsParams) (LogResult, error)
	ListLogsByRunID(args *ListLogsByRunIDFullArgs) (LogResult, error)
}

// MastraLoggerOptions holds configuration for creating a MastraLogger.
type MastraLoggerOptions struct {
	Name       string
	Level      LogLevel
	Transports map[string]LoggerTransport
}

// MastraLoggerBase provides the shared state and default method implementations
// for the abstract MastraLogger class from TypeScript. Concrete loggers embed
// this struct and implement Debug/Info/Warn/Error themselves.
type MastraLoggerBase struct {
	Name       string
	Level      LogLevel
	Transports map[string]LoggerTransport
}

// NewMastraLoggerBase creates a new MastraLoggerBase with the given options.
// Defaults: Name="Mastra", Level=LogLevelError, Transports=empty map.
func NewMastraLoggerBase(opts *MastraLoggerOptions) MastraLoggerBase {
	name := "Mastra"
	level := LogLevelError
	transports := make(map[string]LoggerTransport)

	if opts != nil {
		if opts.Name != "" {
			name = opts.Name
		}
		level = opts.Level
		if opts.Transports != nil {
			for k, v := range opts.Transports {
				transports[k] = v
			}
		}
	}

	return MastraLoggerBase{
		Name:       name,
		Level:      level,
		Transports: transports,
	}
}

// GetTransports returns the map of registered transports.
func (b *MastraLoggerBase) GetTransports() map[string]LoggerTransport {
	return b.Transports
}

// TrackException is a no-op default implementation.
func (b *MastraLoggerBase) TrackException(err *mastraerror.MastraBaseError) {}

// ListLogs delegates to the named transport's ListLogs method.
// Returns an empty result if the transport is not found.
func (b *MastraLoggerBase) ListLogs(transportID string, params *ListLogsParams) (LogResult, error) {
	if transportID == "" {
		var page, perPage *int
		if params != nil {
			page = params.Page
			perPage = params.PerPage
		}
		return emptyLogResult(page, perPage), nil
	}

	transport, ok := b.Transports[transportID]
	if !ok {
		var page, perPage *int
		if params != nil {
			page = params.Page
			perPage = params.PerPage
		}
		return emptyLogResult(page, perPage), nil
	}

	return transport.ListLogs(params)
}

// ListLogsByRunID delegates to the named transport's ListLogsByRunID method.
// TransportID and RunID are extracted from args. Returns an empty result if
// the transport is not found or RunID is empty.
func (b *MastraLoggerBase) ListLogsByRunID(args *ListLogsByRunIDFullArgs) (LogResult, error) {
	if args == nil || args.TransportID == "" || args.RunID == "" {
		var page, perPage *int
		if args != nil {
			page = args.Page
			perPage = args.PerPage
		}
		return emptyLogResult(page, perPage), nil
	}

	transport, ok := b.Transports[args.TransportID]
	if !ok {
		return emptyLogResult(args.Page, args.PerPage), nil
	}

	return transport.ListLogsByRunID(&args.ListLogsByRunIDArgs)
}

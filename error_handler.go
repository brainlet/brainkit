package brainkit

import "log/slog"

// ErrorContext provides context about where a non-fatal error occurred.
type ErrorContext struct {
	Operation string // "LoadDeployments", "SaveSchedule", "RestorePlugin"
	Component string // "kernel", "node", "plugin", "workflow", "persistence"
	Source    string // deployment source or plugin name, "" if N/A
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

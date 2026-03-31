package brainkit

import "log"

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
	if ctx.Source != "" {
		log.Printf("[brainkit] [%s] %s %s: %v", ctx.Component, ctx.Operation, ctx.Source, err)
	} else {
		log.Printf("[brainkit] [%s] %s: %v", ctx.Component, ctx.Operation, err)
	}
}

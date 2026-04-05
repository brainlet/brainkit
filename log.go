package brainkit

import (
	"log/slog"
	"time"
)

// LogEntry is a tagged log entry from a .ts Compartment or the Kernel.
// Source identifies the origin: "myagent.ts", "kernel".
// Level is one of: "log", "warn", "error", "debug", "info".
//
// Concurrency: LogHandler is called from multiple goroutines concurrently.
// brainkit does NOT serialize calls. The consumer's LogHandler MUST be goroutine-safe.
type LogEntry struct {
	Source  string
	Level   string
	Message string
	Time    time.Time
}

// defaultLogHandler logs .ts compartment entries via slog.
func defaultLogHandler(e LogEntry) {
	level := slog.LevelInfo
	switch e.Level {
	case "error":
		level = slog.LevelError
	case "warn":
		level = slog.LevelWarn
	case "debug":
		level = slog.LevelDebug
	}
	slog.LogAttrs(nil, level, e.Message,
		slog.String("source", e.Source),
	)
}

// emitLog sends a log entry through the Kernel's LogHandler.
func (k *Kernel) emitLog(source, level, message string) {
	handler := k.config.LogHandler
	if handler == nil {
		handler = defaultLogHandler
	}
	handler(LogEntry{
		Source:  source,
		Level:   level,
		Message: message,
		Time:    time.Now(),
	})
}

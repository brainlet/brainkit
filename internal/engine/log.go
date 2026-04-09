package engine

import (
	"log/slog"
	"time"

	"github.com/brainlet/brainkit/internal/types"
)

// types.LogEntry alias for use within engine.

// defaultLogHandler logs .ts compartment entries via slog.
func defaultLogHandler(e types.LogEntry) {
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
	handler(types.LogEntry{
		Source:  source,
		Level:   level,
		Message: message,
		Time:    time.Now(),
	})
}

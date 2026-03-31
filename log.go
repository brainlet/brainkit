package brainkit

import (
	"log"
	"time"
)

// LogEntry is a tagged log entry from a .ts Compartment, WASM module, or the Kernel.
// Source identifies the origin: "myagent.ts", "wasm:counter-shard", "kernel".
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

// defaultLogHandler prints to stdout via log.Printf. Used when KernelConfig.LogHandler is nil.
func defaultLogHandler(e LogEntry) {
	log.Printf("[%s] [%s] %s", e.Source, e.Level, e.Message)
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

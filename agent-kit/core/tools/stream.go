// Ported from: packages/core/src/tools/stream.ts
package tools

import (
	"fmt"
	"io"
	"sync"
)

// ToolStream wraps an io.Writer and enriches each written chunk with metadata
// (toolCallId, toolName, runId) before passing it to the underlying OutputWriter.
//
// In TypeScript this extends WritableStream<unknown>. In Go we use an io.Writer
// interface and provide explicit Write/Custom methods for structured output.
type ToolStream struct {
	mu       sync.Mutex
	prefix   string
	callID   string
	name     string
	runID    string
	writeFn  OutputWriter
	writer   io.Writer // optional raw writer for io.Writer compatibility
}

// ToolStreamConfig holds the configuration for creating a new ToolStream.
type ToolStreamConfig struct {
	Prefix string
	CallID string
	Name   string
	RunID  string
}

// NewToolStream creates a new ToolStream with the given configuration and optional OutputWriter.
//
// The OutputWriter is invoked for each Write/Custom call with structured event data.
// If writeFn is nil, writes are silently discarded.
func NewToolStream(cfg ToolStreamConfig, writeFn OutputWriter) *ToolStream {
	ts := &ToolStream{
		prefix:  cfg.Prefix,
		callID:  cfg.CallID,
		name:    cfg.Name,
		runID:   cfg.RunID,
		writeFn: writeFn,
	}
	return ts
}

// writeInternal writes a chunk with metadata wrapping through the OutputWriter.
// This is the Go equivalent of the private _write method in TypeScript.
func (ts *ToolStream) writeInternal(data any) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.writeFn == nil {
		return nil
	}

	payload := map[string]any{
		"output": data,
	}

	if ts.prefix == "workflow-step" {
		payload["runId"] = ts.runID
		payload["stepName"] = ts.name
	} else {
		payload[fmt.Sprintf("%sCallId", ts.prefix)] = ts.callID
		payload[fmt.Sprintf("%sName", ts.prefix)] = ts.name
	}

	event := map[string]any{
		"type":    fmt.Sprintf("%s-output", ts.prefix),
		"runId":   ts.runID,
		"from":    "USER",
		"payload": payload,
	}

	return ts.writeFn(event)
}

// WriteData writes a data chunk through the ToolStream, wrapping it with metadata.
// This is the Go equivalent of the public write() method in TypeScript.
func (ts *ToolStream) WriteData(data any) error {
	return ts.writeInternal(data)
}

// Custom writes a custom event directly through the OutputWriter without metadata wrapping.
// This is the Go equivalent of the custom<T>() method in TypeScript.
//
// The data parameter should be a struct or map with a "type" field.
// In TypeScript, the type was constrained to { type: `data-${string}` } via generics;
// in Go, callers are responsible for providing correctly typed data.
func (ts *ToolStream) Custom(data any) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.writeFn == nil {
		return nil
	}
	return ts.writeFn(data)
}

// Write implements io.Writer for compatibility with Go's standard streaming interfaces.
// It wraps the raw bytes as a string chunk through the ToolStream's metadata pipeline.
func (ts *ToolStream) Write(p []byte) (n int, err error) {
	if err := ts.writeInternal(string(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Ensure ToolStream implements io.Writer at compile time.
var _ io.Writer = (*ToolStream)(nil)

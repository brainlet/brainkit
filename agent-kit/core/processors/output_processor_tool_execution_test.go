// Ported from: packages/core/src/processors/output-processor-tool-execution.test.ts
package processors

import (
	"testing"
)

// These tests verify that output processors receive tool-result chunks and that
// state persists across tool execution steps. In TS they use a full Agent with
// a MockLanguageModelV2, createTool, and stream(). Those high-level orchestration
// constructs (Agent, stream, createTool, MockLanguageModelV2) are not yet ported
// to Go, so every test that requires them is compiled but skipped.

func TestOutputProcessorToolResultChunks(t *testing.T) {
	t.Run("should receive tool-result chunks in processOutputStream", func(t *testing.T) {
		t.Skip("not yet implemented: requires Agent, createTool, MockLanguageModelV2, and stream() which are not ported")

		// TS test creates a ToolResultTrackingProcessor that captures chunk types
		// from processOutputStream, creates an echoTool via createTool, wires up a
		// MockLanguageModelV2 that first emits a tool-call then (after tool execution)
		// emits text, and asserts that the processor received 'tool-result' chunks.
		//
		// TODO: Port once Agent + stream() are available in Go.
	})
}

func TestOutputProcessorStatePersistenceAcrossToolExecution(t *testing.T) {
	t.Run("should filter intermediate finish chunks and maintain state during tool execution", func(t *testing.T) {
		t.Skip("not yet implemented: requires Agent, createTool, MockLanguageModelV2, and stream() which are not ported")

		// TS test creates a StateTrackingProcessor that captures each chunk's type
		// and the accumulated streamParts types. It verifies:
		//   - Only 1 finish chunk reaches the output processor (the final one)
		//   - tool-call is the second chunk (after response-metadata)
		//   - State accumulation works (each chunk sees prior chunks)
		//
		// TODO: Port once Agent + stream() are available in Go.
	})
}

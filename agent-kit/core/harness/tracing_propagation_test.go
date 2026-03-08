// Ported from: packages/core/src/harness/tracing-propagation.test.ts
package harness

import (
	"testing"
)

func TestHarnessTracingPropagation(t *testing.T) {
	t.Skip("not yet implemented - requires sendMessage with tracingContext/tracingOptions forwarding to agent.stream()")

	// The TS tests verify:
	// 1. should forward tracingContext to agent.stream() when provided
	//    - Creates a harness with an agent, spies on agent.stream
	//    - Calls harness.sendMessage({content: "hello", tracingContext: {currentSpan: ...}})
	//    - Verifies agent.stream received the tracingContext option
	//
	// 2. should forward tracingOptions to agent.stream() when provided
	//    - Calls harness.sendMessage({content: "hello", tracingOptions: {traceId, parentSpanId, metadata}})
	//    - Verifies agent.stream received the tracingOptions option
	//
	// 3. should not include tracingContext/tracingOptions when not provided
	//    - Calls harness.sendMessage({content: "hello"})
	//    - Verifies agent.stream options do NOT have tracingContext or tracingOptions
	//
	// These require:
	// - Full sendMessage implementation that calls agent.stream()
	// - TracingContext and TracingOptions types from observability package
	// - Agent interface with Stream() method
}

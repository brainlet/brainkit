// Ported from: packages/core/src/harness/om-failure-abort.test.ts
package harness

import (
	"testing"
)

func TestHarnessOMFailureAbortBehavior(t *testing.T) {
	t.Skip("not yet implemented - requires processStream method with OM buffering/observation failure handling and abort logic")

	// The TS tests verify:
	// 1. Aborts stream and emits error when OM buffering fails
	//    - Sends a data-om-buffering-failed stream chunk
	//    - Expects om_buffering_failed event emitted
	//    - Expects error event with message containing "Observational memory observation buffering failed: Bad Request"
	//    - Expects abortRequested to be true
	//    - Expects no message_start event (stream aborted before text processing)
	//
	// 2. Aborts stream and emits error when OM observation run fails
	//    - Sends a data-om-observation-failed stream chunk
	//    - Expects om_reflection_failed event emitted
	//    - Expects error event with message containing "Observational memory reflection run failed: Model unavailable"
	//    - Expects abortRequested to be true
	//    - Expects no message_start event
	//
	// These require:
	// - processStream() method on Harness
	// - Stream chunk type handling for data-om-buffering-failed and data-om-observation-failed
	// - AbortController/abort logic integration
}

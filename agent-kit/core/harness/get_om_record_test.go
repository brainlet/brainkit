// Ported from: packages/core/src/harness/get-om-record.test.ts
package harness

import (
	"testing"
)

func TestHarnessGetObservationalMemoryRecord(t *testing.T) {
	t.Skip("not yet implemented - requires getObservationalMemoryRecord method, InMemoryStore, and observational memory storage integration")

	// The TS tests verify:
	// 1. Returns null when no thread is selected
	// 2. Returns null when no OM record exists for the thread
	// 3. Returns the OM record with activeObservations when one exists
	//    - Creates a thread, seeds OM via storage, verifies record fields
	// 4. Returns record for the current thread after switching threads
	//    - Creates two threads with different OM data
	//    - Verifies correct OM data per thread after switchThread
	//
	// These require:
	// - Harness.getObservationalMemoryRecord() method
	// - InMemoryStore with memory store supporting:
	//   - initializeObservationalMemory({threadId, resourceId, scope, config})
	//   - updateActiveObservations({id, observations, tokenCount, lastObservedAt})
	// - Storage integration in Harness (createThread persisting to storage, switchThread, etc.)
}

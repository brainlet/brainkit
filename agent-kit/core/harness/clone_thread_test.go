// Ported from: packages/core/src/harness/clone-thread.test.ts
package harness

import (
	"testing"
)

func TestHarnessCloneThread(t *testing.T) {
	t.Skip("not yet implemented - requires cloneThread method, dynamic memory factory resolution, and MastraMemory integration")

	// The TS tests verify:
	// 1. cloneThread resolves dynamic memory factory before cloning
	//    - Creates a harness with a memory factory function
	//    - Calls harness.cloneThread({sourceThreadId, title, resourceId})
	//    - Verifies factory was called once
	//    - Verifies cloneThread on memory was called with correct args
	//    - Verifies returned thread has correct id and resourceId
	//
	// 2. cloneThread throws when dynamic memory factory returns empty value
	//    - Creates harness with a factory that returns undefined
	//    - Expects cloneThread to reject with "Dynamic memory factory returned empty value"
	//
	// These require:
	// - Harness.cloneThread() method
	// - Dynamic memory factory support (memory as a function in config)
	// - MastraMemory.cloneThread() on the memory interface
}

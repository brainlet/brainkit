package lrucache

// warn_missing_ac_test.go — Placeholder for node-lru-cache test/warn-missing-ac.ts (110 lines).
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/warn-missing-ac.ts
//
// SKIPPED: The Go port does not implement fetchMethod or AbortController integration.
// This test verifies that using fetchMethod without AbortController/AbortSignal
// produces appropriate warnings.
//
// The TS test covers:
//   - Warning when fetchMethod is used without AbortController polyfill
//   - LRU_CACHE_IGNORE_AC_WARNING environment variable
//   - process.emitWarning vs console.error fallback
//
// If fetchMethod and signal support are added to the Go port, this file should
// be replaced with a full 1:1 port of test/warn-missing-ac.ts.

import "testing"

func TestWarnMissingAC_NotImplemented(t *testing.T) {
	// test/warn-missing-ac.ts — 110 lines
	t.Skip("SKIP: fetchMethod and AbortController are not implemented in the Go port. " +
		"See test/warn-missing-ac.ts in node-lru-cache for the original 110-line test suite.")
}

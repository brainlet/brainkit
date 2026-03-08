package lrucache

// memo_test.go — Placeholder for node-lru-cache test/memo.ts (70 lines).
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/memo.ts
//
// SKIPPED: The Go port does not implement memoMethod.
// memoMethod provides synchronous memoization with automatic cache population.
//
// The TS test covers:
//   - Basic memoization: memo(key) calls memoMethod on miss, caches result
//   - Memoization with context parameter
//   - Type validation: memoMethod must be a function
//   - memo() without memoMethod throws
//
// If memoMethod is added to the Go port in the future, this file should be
// replaced with a full 1:1 port of test/memo.ts.

import "testing"

func TestMemo_NotImplemented(t *testing.T) {
	// test/memo.ts — 70 lines of memoization testing
	t.Skip("SKIP: memoMethod is not implemented in the Go port. " +
		"See test/memo.ts in node-lru-cache for the original 70-line test suite.")
}

package lrucache

// fetch_test.go — Placeholder for node-lru-cache test/fetch.ts (851 lines).
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/fetch.ts
//
// SKIPPED: The Go port does not implement fetchMethod (async background fetching).
// fetchMethod relies on JavaScript's Promise/async-await concurrency model
// which has no direct equivalent in Go's goroutine-based concurrency.
//
// The TS test covers:
//   - Asynchronous fetching with stale-while-revalidate
//   - AbortController signal handling
//   - Background fetch lifecycle (pending, resolved, rejected)
//   - Fetch options: forceRefresh, allowStaleOnFetchRejection, allowStaleOnFetchAbort
//   - Fetch with context parameter
//   - Concurrent fetch deduplication
//   - Fetch + TTL interaction
//   - Fetch + size tracking interaction
//   - ignoreFetchAbort option
//
// If fetchMethod is added to the Go port in the future, this file should be
// replaced with a full 1:1 port of test/fetch.ts.

import "testing"

func TestFetch_NotImplemented(t *testing.T) {
	// test/fetch.ts — 851 lines of async fetch testing
	t.Skip("SKIP: fetchMethod is not implemented in the Go port. " +
		"See test/fetch.ts in node-lru-cache for the original 851-line test suite.")
}

package lrucache

// lost_background_fetch_test.go — Placeholder for node-lru-cache test/lost-background-fetch.ts (50 lines).
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/lost-background-fetch.ts
//
// SKIPPED: The Go port does not implement fetchMethod (async background fetching).
// This test verifies edge cases where background fetches are aborted and
// re-fetched with ignoreFetchAbort and allowStaleOnFetchAbort options.
//
// Issue reference: https://github.com/isaacs/node-lru-cache/issues/389
//
// The TS test covers:
//   - Fetch with ignoreFetchAbort: true and allowStaleOnFetchAbort: true
//   - AbortController.abort() during pending fetch
//   - Stale value availability after abort
//   - Fetch resolution after abort
//
// If fetchMethod is added to the Go port, this file should be replaced with
// a full 1:1 port of test/lost-background-fetch.ts.

import "testing"

func TestLostBackgroundFetch_NotImplemented(t *testing.T) {
	// test/lost-background-fetch.ts — 50 lines
	t.Skip("SKIP: fetchMethod is not implemented in the Go port. " +
		"See test/lost-background-fetch.ts in node-lru-cache for the original 50-line test suite.")
}

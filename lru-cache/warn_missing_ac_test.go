package lrucache

// warn_missing_ac_test.go documents the one upstream test that is intentionally
// not ported 1:1. node-lru-cache warns when AbortController is missing because
// the JS runtime may not provide it. The Go port uses context.Context for fetch
// cancellation, so there is no equivalent runtime capability check or warning.

import "testing"

func TestWarnMissingAC_NotApplicable(t *testing.T) {
	t.Skip("AbortController polyfill warnings are JS-specific; the Go port uses context.Context for fetch cancellation.")
}

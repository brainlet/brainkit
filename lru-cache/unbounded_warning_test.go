package lrucache

// unbounded_warning_test.go — Faithful 1:1 port of test/unbounded-warning.ts from node-lru-cache.
// TS source: test/unbounded-warning.ts (68 lines)
//
// ADAPTATION NOTES:
//
// In the TS version, creating an LRUCache with only TTL (no max, no maxSize,
// no ttlAutopurge) emits a warning via process.emitWarning() or console.error().
// This is because such a cache can grow without bound — items are only removed
// when their TTL expires, but if items are added faster than they expire, memory
// grows indefinitely.
//
// In the Go port, New() panics for invalid configurations instead of emitting
// warnings. The equivalent Go behavior is:
//   - TTL only (no max, no maxSize, no ttlAutopurge) → panic
//   - TTL + ttlAutopurge (no max, no maxSize) → OK (autopurge prevents unbounded growth)
//   - TTL + max → OK
//   - TTL + maxSize → OK
//
// These tests verify that the Go constructor correctly panics for unbounded
// configurations and correctly allows bounded ones.

import "testing"

// TestUnboundedWarning_TTLOnlyPanics verifies that creating a cache with only TTL
// (no max, no maxSize, no ttlAutopurge) panics in Go.
// TS source: test/unbounded-warning.ts:4-28 ("emits warning")
// In TS, this emits an "UnboundedCacheWarning" with code "LRU_CACHE_UNBOUNDED".
// In Go, this is a hard panic because unbounded caches are a programming error.
func TestUnboundedWarning_TTLOnlyPanics(t *testing.T) {
	// TS source: test/unbounded-warning.ts:18-20
	// new LRUCache({ ttl: 100 }) → emits warning
	// Go equivalent: New(Options{ TTL: 100 }) → panics
	assertPanics(t, func() {
		New[int, int](Options[int, int]{
			TTL: 100,
		})
	}, "TTL without max, maxSize, or ttlAutopurge should panic")
}

// TestUnboundedWarning_TTLOnlyNoPrint verifies the same panic happens regardless of
// how the cache is constructed (the TS test had a separate case for when
// process.emitWarning is undefined and it falls back to console.error).
// TS source: test/unbounded-warning.ts:31-68 ("prints to stderr if no process.emitWarning")
// In Go, the behavior is always the same: panic. There's no fallback path.
func TestUnboundedWarning_TTLOnlyAlwaysPanics(t *testing.T) {
	// TS source: test/unbounded-warning.ts:55-57 and 59-61
	// Two separate `new LRU({ ttl: 100 })` calls, both produce stderr output.
	// In TS, the warning is only emitted once (deduplicated by code).
	// In Go, every call panics independently.

	// First construction — panics
	// TS source: test/unbounded-warning.ts:55-57
	assertPanics(t, func() {
		New[int, int](Options[int, int]{
			TTL: 100,
		})
	}, "first construction with TTL-only should panic")

	// Second construction — also panics (no deduplication needed in Go)
	// TS source: test/unbounded-warning.ts:59-61
	assertPanics(t, func() {
		New[string, string](Options[string, string]{
			TTL: 100,
		})
	}, "second construction with TTL-only should also panic")
}

// TestUnboundedWarning_TTLWithAutopurgeOK verifies that TTL + ttlAutopurge
// does NOT panic, because autopurge prevents unbounded memory growth.
// This is the "fix" for the unbounded cache warning — adding ttlAutopurge
// ensures expired items are cleaned up automatically.
func TestUnboundedWarning_TTLWithAutopurgeOK(t *testing.T) {
	// No TS equivalent — in TS, ttlAutopurge suppresses the warning.
	// In Go, it prevents the panic.
	c := New[int, int](Options[int, int]{
		TTL:          100,
		TTLAutopurge: true,
	})
	if c == nil {
		t.Fatal("TTL + ttlAutopurge should create a valid cache")
	}
}

// TestUnboundedWarning_TTLWithMaxOK verifies that TTL + max does NOT panic.
func TestUnboundedWarning_TTLWithMaxOK(t *testing.T) {
	c := New[int, int](Options[int, int]{
		TTL: 100,
		Max: 10,
	})
	if c == nil {
		t.Fatal("TTL + max should create a valid cache")
	}
}

// TestUnboundedWarning_TTLWithMaxSizeOK verifies that TTL + maxSize does NOT panic.
func TestUnboundedWarning_TTLWithMaxSizeOK(t *testing.T) {
	c := New[int, int](Options[int, int]{
		TTL:     100,
		MaxSize: 1000,
	})
	if c == nil {
		t.Fatal("TTL + maxSize should create a valid cache")
	}
}

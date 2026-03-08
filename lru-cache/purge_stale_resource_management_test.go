package lrucache

// purge_stale_resource_management_test.go — Faithful 1:1 port of node-lru-cache
// test/purge-stale-resource-management.ts (137 lines).
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/purge-stale-resource-management.ts
//
// Every test case includes a comment with the original source file and line number.
// Uses test helpers from helpers_test.go.
//
// ADAPTATION NOTES:
//
//   The TS version monkey-patches global.setTimeout and global.clearTimeout to count
//   how many timers are created and cleared. It verifies that autopurge timer
//   management doesn't leak resources (e.g., creating N timers for N overwrites of
//   the same key, without clearing the old ones).
//
//   TS counting strategy (lines 16-38):
//     let timeouts = 0
//     const origST = global.setTimeout
//     global.setTimeout = (...args) => { ++timeouts; return origST.apply(...) }
//     let clears = 0
//     const origCT = global.clearTimeout
//     global.clearTimeout = (...args) => { ++clears; return origCT.apply(...) }
//
//   In Go, we cannot monkey-patch time.AfterFunc. Instead, we:
//     1. Expose the internal autopurgeTimers slice to count active (non-nil) timers.
//     2. Verify the functional behavior: items are actually autopurged after TTL.
//     3. Verify resource cleanup: after clear/delete/evict, timers are nil.
//
//   The TS test also uses t.clock (fake timers from the tap test framework):
//     const clock = t.clock
//     clock.advance(1)
//   This ensures setTimeout is intercepted by the fake timer system. In Go,
//   autopurge uses real time.AfterFunc timers (not affected by NowFn), so
//   behavioral tests use short TTLs and time.Sleep where needed.
//
//   The TS test structure is preserved as closely as possible, with comments
//   explaining the original setTimeout/clearTimeout counting assertions.

import (
	"testing"
	"time"
)

// exposeAutopurgeTimers returns the internal autopurge timer slice for white-box testing.
// TS equivalent: not directly exposed in TS; the TS test counts setTimeout/clearTimeout calls.
func exposeAutopurgeTimers[K comparable, V any](c *LRUCache[K, V]) []*time.Timer {
	return c.autopurgeTimers
}

// countActiveTimers counts non-nil timers in the autopurge timer slice.
// TS equivalent: the TS test tracks this via the timeouts/clears counters.
// In Go, a non-nil timer means a timer was created and not yet cleared.
func countActiveTimers[K comparable, V any](c *LRUCache[K, V]) int {
	timers := exposeAutopurgeTimers(c)
	if timers == nil {
		return 0
	}
	count := 0
	for _, t := range timers {
		if t != nil {
			count++
		}
	}
	return count
}

// TestPurgeStaleResourceManagement_HotKeyOverwrite ports the first TS subtest.
// TS source: test/purge-stale-resource-management.ts lines 42-78
// "a cache that overwrites a hot key many times"
//
// Original TS assertions using setTimeout/clearTimeout counters:
//   N sets of same key: t.equal(timeouts, N), t.equal(clears, N-1)
//   set with ttl:0:     t.equal(timeoutsAfterSetTTL0, 0), t.equal(clearsAfterSetTTL0, 1)
//   set with default:   t.equal(clearsAfterSetTTLDef, 0), t.equal(timeoutsAfterSetTTLDef, 1)
//   delete:             t.equal(clearsAfterDelete, 1), t.equal(timeoutsAfterDelete, 0)
//
// Go adaptation: Instead of counting create/clear calls, we verify the resulting
// active timer count after each operation. The invariant is the same: at most 1
// active timer per key, and timers are properly cleaned up on overwrite/delete.
func TestPurgeStaleResourceManagement_HotKeyOverwrite(t *testing.T) {
	// test/purge-stale-resource-management.ts lines 43-46:
	// const cache = new LRU<string, number>({ ttl: 10, ttlAutopurge: true })
	cache := New[string, int](Options[string, int]{
		TTL:          10,
		TTLAutopurge: true,
	})

	// test/purge-stale-resource-management.ts lines 48-51:
	// const N = 10 //_000
	// for (let i = 0; i < N; i++) { cache.set('hot-key', i) }
	N := 10
	for i := 0; i < N; i++ {
		cache.Set("hot-key", i)
	}

	// test/purge-stale-resource-management.ts lines 52-53:
	// t.equal(timeouts, N)   — N timers created (one per set)
	// t.equal(clears, N - 1) — N-1 old timers cleared (each overwrite clears prev)
	//
	// In Go: after N overwrites of the same key, exactly 1 timer should be active
	// (the latest one). The previous N-1 timers were stopped and set to nil in
	// setItemTTL when the old timer is cancelled before creating the new one.
	assertEqual(t, countActiveTimers(cache), 1,
		"after N overwrites: exactly 1 active timer (latest)")

	// test/purge-stale-resource-management.ts lines 55-62:
	// timeouts = 0; clears = 0
	// cache.set('hot-key', 99, { ttl: 0 })
	// const clearsAfterSetTTL0 = clears
	// const timeoutsAfterSetTTL0 = timeouts
	// t.equal(timeoutsAfterSetTTL0, 0) — no new timer (ttl:0 = no TTL)
	// t.equal(clearsAfterSetTTL0, 1)   — old timer cleared
	cache.Set("hot-key", 99, SetOptions[string, int]{TTL: Int64(0)})

	// In Go: setting with ttl:0 clears the existing timer and creates none.
	// The setItemTTL method stops the old timer and only creates a new one if ttl > 0.
	assertEqual(t, countActiveTimers(cache), 0,
		"after set with ttl:0: no active timers")

	// test/purge-stale-resource-management.ts lines 64-70:
	// timeouts = 0; clears = 0
	// cache.set('hot-key', 100)
	// const clearsAfterSetTTLDef = clears
	// const timeoutsAfterSetTTLDef = timeouts
	// t.equal(clearsAfterSetTTLDef, 0)  — no old timer to clear (ttl:0 removed it)
	// t.equal(timeoutsAfterSetTTLDef, 1) — new timer created (default ttl:10)
	cache.Set("hot-key", 100)

	// In Go: re-setting with default TTL creates a new timer.
	// There was no previous timer to clear (ttl:0 had already cleared it).
	assertEqual(t, countActiveTimers(cache), 1,
		"after set with default ttl: 1 active timer")

	// test/purge-stale-resource-management.ts lines 72-78:
	// timeouts = 0; clears = 0
	// cache.delete('hot-key')
	// const clearsAfterDelete = clears
	// const timeoutsAfterDelete = timeouts
	// t.equal(clearsAfterDelete, 1)   — timer cleared on delete
	// t.equal(timeoutsAfterDelete, 0) — no new timer created
	cache.Delete("hot-key")

	// In Go: internalDelete cancels the timer via autopurgeTimers[index].Stop().
	assertEqual(t, countActiveTimers(cache), 0,
		"after delete: no active timers")
}

// TestPurgeStaleResourceManagement_EvictClearsTimer ports the second TS subtest.
// TS source: test/purge-stale-resource-management.ts lines 81-108
// "evicting an item means no need for autopurge"
//
// Original TS assertions:
//   after set 'a':     t.equal(clearsAfterSet, 0), t.equal(timeoutsAfterSet, 1)
//   after eviction:    t.equal(clearsAfterEvict, 1), t.equal(timeoutsAfterEvict, 0)
func TestPurgeStaleResourceManagement_EvictClearsTimer(t *testing.T) {
	// test/purge-stale-resource-management.ts lines 82-86:
	// const cache = new LRU<string, number>({ ttl: 10, max: 5, ttlAutopurge: true })
	cache := New[string, int](Options[string, int]{
		TTL:          10,
		Max:          5,
		TTLAutopurge: true,
	})

	// test/purge-stale-resource-management.ts lines 88-94:
	// timeouts = 0; clears = 0
	// cache.set('a', 1)
	// const clearsAfterSet = clears
	// const timeoutsAfterSet = timeouts
	// t.equal(clearsAfterSet, 0)   — no old timer to clear
	// t.equal(timeoutsAfterSet, 1) — 1 new timer created for 'a'
	cache.Set("a", 1)
	assertEqual(t, countActiveTimers(cache), 1,
		"after set 'a': 1 active timer")

	// test/purge-stale-resource-management.ts lines 96-102:
	// timeouts = 0; clears = 0
	// cache.set('b', 1, { ttl: 0 })
	// cache.set('c', 1, { ttl: 0 })
	// cache.set('d', 1, { ttl: 0 })
	// cache.set('e', 1, { ttl: 0 })
	// cache.set('f', 1, { ttl: 0 })  <- this evicts 'a' because max:5
	cache.Set("b", 1, SetOptions[string, int]{TTL: Int64(0)})
	cache.Set("c", 1, SetOptions[string, int]{TTL: Int64(0)})
	cache.Set("d", 1, SetOptions[string, int]{TTL: Int64(0)})
	cache.Set("e", 1, SetOptions[string, int]{TTL: Int64(0)})
	cache.Set("f", 1, SetOptions[string, int]{TTL: Int64(0)})

	// test/purge-stale-resource-management.ts lines 104-107:
	// const clearsAfterEvict = clears
	// const timeoutsAfterEvict = timeouts
	// t.equal(clearsAfterEvict, 1)   — 'a's timer cleared on eviction
	// t.equal(timeoutsAfterEvict, 0) — no new timers (all have ttl:0)
	//
	// In Go: 'a' was evicted by 'f' (max:5, 6th item triggers eviction of LRU).
	// The evictHead method stops 'a's autopurge timer. Items b/c/d/e/f all
	// have ttl:0 so no timers were created for them.
	assertEqual(t, countActiveTimers(cache), 0,
		"after evicting 'a': no active timers (all remaining have ttl:0)")
}

// TestPurgeStaleResourceManagement_ClearClearsTimers ports the third TS subtest.
// TS source: test/purge-stale-resource-management.ts lines 110-136
// "clearing list clears autopurge timers"
//
// Original TS assertions:
//   after 4 sets: t.equal(clearsAfterSet, 0), t.equal(timeoutsAfterSet, 4)
//   after clear:  t.equal(clearsAfterClear, 4), t.equal(timeoutsAfterClear, 0)
func TestPurgeStaleResourceManagement_ClearClearsTimers(t *testing.T) {
	// test/purge-stale-resource-management.ts lines 111-115:
	// const cache = new LRU<string, number>({ ttl: 10, max: 5, ttlAutopurge: true })
	cache := New[string, int](Options[string, int]{
		TTL:          10,
		Max:          5,
		TTLAutopurge: true,
	})

	// test/purge-stale-resource-management.ts lines 117-126:
	// timeouts = 0; clears = 0
	// cache.set('a', 1)
	// cache.set('b', 1)
	// cache.set('c', 1)
	// cache.set('d', 1)
	// const clearsAfterSet = clears
	// const timeoutsAfterSet = timeouts
	// t.equal(clearsAfterSet, 0)   — no old timers to clear (all fresh keys)
	// t.equal(timeoutsAfterSet, 4) — 4 new timers created
	cache.Set("a", 1)
	cache.Set("b", 1)
	cache.Set("c", 1)
	cache.Set("d", 1)
	assertEqual(t, countActiveTimers(cache), 4,
		"after 4 sets: 4 active timers")

	// test/purge-stale-resource-management.ts lines 128-135:
	// timeouts = 0; clears = 0
	// cache.clear()
	// const clearsAfterClear = clears
	// const timeoutsAfterClear = timeouts
	// t.equal(clearsAfterClear, 4)  — all 4 timers cleared
	// t.equal(timeoutsAfterClear, 0) — no new timers created
	cache.Clear()

	// In Go: the internalClear method iterates autopurgeTimers and calls
	// Stop() + nil on each timer (lrucache.go lines 908-914).
	assertEqual(t, countActiveTimers(cache), 0,
		"after clear: no active timers")
	assertEqual(t, cache.Size(), 0,
		"after clear: size is 0")
}

// TestPurgeStaleResourceManagement_AutopurgeActuallyPurges verifies the
// behavioral outcome of TTLAutopurge using real timers.
//
// This is a Go-specific supplemental test. The TS test only counts timer
// creation/cancellation via monkey-patched setTimeout/clearTimeout. In Go
// we cannot intercept time.AfterFunc, so we also verify the functional
// outcome: items are actually deleted when their autopurge timer fires.
//
// Since autopurge uses time.AfterFunc (real goroutine timers), not NowFn,
// we use short TTLs and time.Sleep for behavioral verification.
func TestPurgeStaleResourceManagement_AutopurgeActuallyPurges(t *testing.T) {
	cache := New[string, int](Options[string, int]{
		TTL:          50, // 50ms TTL — short enough for fast tests
		TTLAutopurge: true,
	})

	cache.Set("ephemeral", 42)
	assertEqual(t, cache.Size(), 1, "size after set")

	// Wait for the autopurge timer to fire.
	// The timer calls internalDelete with DisposeExpire after TTL elapses.
	// We sleep 2x the TTL to account for timer scheduling jitter.
	time.Sleep(120 * time.Millisecond)

	// The item should have been auto-deleted by the time.AfterFunc callback.
	assertEqual(t, cache.Size(), 0, "size after autopurge timer fires")

	_, ok := cache.Get("ephemeral")
	assertFalse(t, ok, "item should be gone after autopurge")
}

// TestPurgeStaleResourceManagement_OverwriteDoesNotDoublePurge verifies that
// overwriting a key cancels the old autopurge timer and only the latest TTL applies.
//
// Go-specific supplemental test: since we cannot count timer creates/cancels,
// we verify the behavioral invariant — only the latest TTL determines when
// the item is purged, and old timer firings don't corrupt state.
func TestPurgeStaleResourceManagement_OverwriteDoesNotDoublePurge(t *testing.T) {
	cache := New[string, int](Options[string, int]{
		TTL:          30, // 30ms default TTL
		TTLAutopurge: true,
	})

	// Set with 30ms TTL, then overwrite with a much longer TTL.
	// The first timer (30ms) should be cancelled and not fire.
	cache.Set("key", 1)
	cache.Set("key", 2, SetOptions[string, int]{TTL: Int64(200)})

	// Wait past the original 30ms TTL — the item should still exist
	// because the overwrite cancelled the 30ms timer and set a 200ms one.
	time.Sleep(60 * time.Millisecond)

	v, ok := cache.Get("key")
	assertTrue(t, ok, "item should still exist after original TTL expires")
	assertEqual(t, v, 2, "item should have the overwritten value")

	// Now wait for the 200ms timer to fire.
	time.Sleep(200 * time.Millisecond)

	_, ok = cache.Get("key")
	assertFalse(t, ok, "item should be purged after new TTL expires")
	assertEqual(t, cache.Size(), 0, "cache should be empty after autopurge")
}

package lrucache

// helpers_test.go — Shared test infrastructure for lru-cache Go port.
// Ported from: test/fixtures/expose.ts + tap testing patterns.
//
// The node-lru-cache tests use the "tap" testing framework which provides:
//   - t.clock — a mock clock with advance(), enter(), exit(), now()
//   - t.matchSnapshot — snapshot testing (we skip these, verify values directly)
//   - t.strictSame — deep equality (we use assertSliceEqual / assertEntriesEqual)
//   - t.rejects — promise rejection testing (not applicable in Go)
//   - expose() — exposes LRUCache internals for white-box testing
//
// Go equivalents:
//   - testClock — mock clock with advance() and nowFn()
//   - assertEqual / assertTrue / assertFalse — basic assertions
//   - assertSliceEqual — ordered slice comparison
//   - assertPanics — panic assertion (equivalent to t.throws)
//   - exposeInternals — exposes internal state for white-box testing

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test clock for deterministic TTL testing
// TS source: test/ttl.ts uses t.clock with clock.advance()
//
// In the TS tests, t.clock provides:
//   - clock.advance(ms) — move time forward
//   - clock.now() — current fake time
//   - clock.enter() / clock.exit() — activate/deactivate fake time
//   - clock.setTimeout / clock.clearTimeout — fake timers
//
// Our testClock provides the nowFn() function to inject into LRUCache options.
// ---------------------------------------------------------------------------

type testClock struct {
	now int64
}

func newTestClock(startMs int64) *testClock {
	return &testClock{now: startMs}
}

func (tc *testClock) advance(ms int64) {
	tc.now += ms
}

func (tc *testClock) nowFn() int64 {
	return tc.now
}

// ---------------------------------------------------------------------------
// Assertion helpers
// Ported from: tap framework assertion patterns used across all test files.
// ---------------------------------------------------------------------------

// assertEqual checks that got == want.
// TS equivalent: t.equal(got, want, msg)
func assertEqual[T comparable](t *testing.T, got, want T, msg ...string) {
	t.Helper()
	if got != want {
		prefix := ""
		if len(msg) > 0 {
			prefix = msg[0] + ": "
		}
		t.Errorf("%sgot %v, want %v", prefix, got, want)
	}
}

// assertTrue checks that got is true.
// TS equivalent: t.ok(got, msg) or t.equal(got, true, msg)
func assertTrue(t *testing.T, got bool, msg ...string) {
	t.Helper()
	if !got {
		prefix := ""
		if len(msg) > 0 {
			prefix = msg[0] + ": "
		}
		t.Errorf("%sexpected true, got false", prefix)
	}
}

// assertFalse checks that got is false.
// TS equivalent: t.notOk(got, msg) or t.equal(got, false, msg)
func assertFalse(t *testing.T, got bool, msg ...string) {
	t.Helper()
	if got {
		prefix := ""
		if len(msg) > 0 {
			prefix = msg[0] + ": "
		}
		t.Errorf("%sexpected false, got true", prefix)
	}
}

// assertPanics checks that fn panics.
// TS equivalent: t.throws(() => ...)
func assertPanics(t *testing.T, fn func(), msg ...string) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			prefix := ""
			if len(msg) > 0 {
				prefix = msg[0] + ": "
			}
			t.Errorf("%sexpected panic but did not get one", prefix)
		}
	}()
	fn()
}

// assertSliceEqual checks two slices for deep equality.
// TS equivalent: t.strictSame(got, want)
func assertSliceEqual[T comparable](t *testing.T, got, want []T, msg ...string) {
	t.Helper()
	prefix := ""
	if len(msg) > 0 {
		prefix = msg[0] + ": "
	}
	if len(got) != len(want) {
		t.Errorf("%sslice length mismatch: got %d, want %d\n  got:  %v\n  want: %v",
			prefix, len(got), len(want), got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%sslice[%d]: got %v, want %v\n  got:  %v\n  want: %v",
				prefix, i, got[i], want[i], got, want)
			return
		}
	}
}

// assertEntryPairsEqual checks [][2]any entries for equality.
// TS equivalent: t.strictSame([...c.entries()], expected)
func assertEntryPairsEqual[K comparable, V comparable](t *testing.T, got [][2]any, want [][2]any, msg ...string) {
	t.Helper()
	prefix := ""
	if len(msg) > 0 {
		prefix = msg[0] + ": "
	}
	if len(got) != len(want) {
		t.Errorf("%sentries length mismatch: got %d, want %d\n  got:  %v\n  want: %v",
			prefix, len(got), len(want), got, want)
		return
	}
	for i := range want {
		if fmt.Sprint(got[i]) != fmt.Sprint(want[i]) {
			t.Errorf("%sentries[%d]: got %v, want %v",
				prefix, i, got[i], want[i])
			return
		}
	}
}

// assertDisposals checks a slice of disposal records against expected values.
// TS equivalent: t.strictSame(disposed, [[v, k, r], ...])
type disposal[K comparable, V any] struct {
	value  V
	key    K
	reason DisposeReason
}

func assertDisposals[K comparable, V comparable](t *testing.T, got []disposal[K, V], want []disposal[K, V], msg ...string) {
	t.Helper()
	prefix := ""
	if len(msg) > 0 {
		prefix = msg[0] + ": "
	}
	if len(got) != len(want) {
		t.Errorf("%sdisposals length mismatch: got %d, want %d\n  got:  %v\n  want: %v",
			prefix, len(got), len(want), got, want)
		return
	}
	for i := range want {
		if got[i].value != want[i].value || got[i].key != want[i].key || got[i].reason != want[i].reason {
			t.Errorf("%sdisposals[%d]: got {%v, %v, %v}, want {%v, %v, %v}",
				prefix, i, got[i].value, got[i].key, got[i].reason,
				want[i].value, want[i].key, want[i].reason)
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Internal state exposure for white-box testing
// TS source: test/fixtures/expose.ts
//
// expose.ts does:
//   export const expose = (cache, LRU = LRUCache) => {
//     return Object.assign(LRU.unsafeExposeInternals(cache), cache)
//   }
//
// This exposes: keyMap, valList, keyList, sizes, starts, ttls,
//   head, tail, next, prev, free, isStale(), indexes(), rindexes(),
//   isBackgroundFetch(), backgroundFetch(), moveToTail()
//
// In Go we access internals directly since tests are in the same package.
// ---------------------------------------------------------------------------

// exposeValList returns the internal value list for white-box testing.
// TS equivalent: expose(c).valList
func exposeValList[K comparable, V any](c *LRUCache[K, V]) []*V {
	return c.valList
}

// exposeKeyMap returns the internal key→index map.
// TS equivalent: expose(c).keyMap
func exposeKeyMap[K comparable, V any](c *LRUCache[K, V]) map[K]int {
	return c.keyMap
}

// exposeSizes returns the internal sizes slice.
// TS equivalent: expose(c).sizes
func exposeSizes[K comparable, V any](c *LRUCache[K, V]) []int {
	return c.sizes
}

// exposeStarts returns the internal TTL start times.
// TS equivalent: expose(c).starts
func exposeStarts[K comparable, V any](c *LRUCache[K, V]) []int64 {
	return c.starts
}

// exposeTTLs returns the internal TTL values.
// TS equivalent: expose(c).ttls
func exposeTTLs[K comparable, V any](c *LRUCache[K, V]) []int64 {
	return c.ttls
}

// exposeHead returns the head (LRU) index.
// TS equivalent: expose(c).head
func exposeHead[K comparable, V any](c *LRUCache[K, V]) int {
	return c.head
}

// exposeTail returns the tail (MRU) index.
// TS equivalent: expose(c).tail
func exposeTail[K comparable, V any](c *LRUCache[K, V]) int {
	return c.tail
}

// exposeNext returns the forward linked list.
// TS equivalent: expose(c).next
func exposeNext[K comparable, V any](c *LRUCache[K, V]) []int {
	return c.next
}

// exposePrev returns the backward linked list.
// TS equivalent: expose(c).prev
func exposePrev[K comparable, V any](c *LRUCache[K, V]) []int {
	return c.prev
}

// exposeFree returns the free index stack.
// TS equivalent: expose(c).free
func exposeFree[K comparable, V any](c *LRUCache[K, V]) []int {
	return c.free
}

// exposeIsStale checks if an item at the given index is stale.
// TS equivalent: expose(c).isStale(index)
func exposeIsStale[K comparable, V any](c *LRUCache[K, V], index int) bool {
	return c.isStale(index)
}

// exposeMoveToTail moves the item at index to the tail (MRU) position.
// TS equivalent: expose(c).moveToTail(index)
func exposeMoveToTail[K comparable, V any](c *LRUCache[K, V], index int) {
	c.moveToTail(index)
}

// exposeIndexes returns indexes in MRU→LRU order, optionally including stale.
// TS equivalent: [...expose(c).indexes({ allowStale })]
func exposeIndexes[K comparable, V any](c *LRUCache[K, V], allowStale bool) []int {
	var result []int
	c.forEachIndex(allowStale, func(index int) bool {
		result = append(result, index)
		return true
	})
	return result
}

// exposeRIndexes returns indexes in LRU→MRU order, optionally including stale.
// TS equivalent: [...expose(c).rindexes({ allowStale })]
func exposeRIndexes[K comparable, V any](c *LRUCache[K, V], allowStale bool) []int {
	var result []int
	c.forEachRIndex(allowStale, func(index int) bool {
		result = append(result, index)
		return true
	})
	return result
}

// ---------------------------------------------------------------------------
// Unused import guards
// ---------------------------------------------------------------------------

var (
	_ = sync.Mutex{}
	_ = time.Now
)

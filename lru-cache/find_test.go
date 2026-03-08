package lrucache

// Tests ported from node-lru-cache test/find.ts
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/find.ts
//
// The original test file uses fetchMethod and Promises extensively.
// Only the synchronous Find() parts are ported here; all async fetch
// parts are skipped since fetchMethod is a JS-only pattern.
//
// Test helpers (assertEqual, assertTrue, assertFalse, etc.)
// are defined in helpers_test.go — shared across all test files.

import (
	"testing"
)

// ===========================================================================
// Find tests
// TS source: test/find.ts
// ===========================================================================

// valueHolder is the Go equivalent of the TS { value: number } objects
// used in find.ts. We use a struct so we can match by value field.
type valueHolder struct {
	value int
}

func TestFindSync(t *testing.T) {
	// TS source: test/find.ts lines 3-18 (top-level setup)
	// The TS version creates a cache with max=5, ttl=1, fetchMethod, allowStale,
	// and noDeleteOnStaleGet. We skip fetchMethod/ttl/allowStale/noDeleteOnStaleGet
	// since those are tied to the async fetch pattern.
	//
	// TS sets items 0..8 into a max=5 cache, so only items 4..8 remain.
	// Then it tests Find() with various predicates.

	c := New[int, valueHolder](Options[int, valueHolder]{Max: 5})

	// TS source: test/find.ts lines 18-20
	// for (let i = 0; i < 9; i++) { c.set(i, { value: i }) }
	// With max=5, items 0..3 are evicted, leaving 4,5,6,7,8 in cache.
	for i := 0; i < 9; i++ {
		c.Set(i, valueHolder{value: i})
	}

	// TS source: test/find.ts lines 22-26
	// The TS test does: const p = c.fetch(8, { forceRefresh: true })
	// SKIP: forceRefresh fetch is a fetchMethod feature, not applicable in Go.
	// Without the fetch, item 8 is already in the cache with { value: 8 }.

	t.Run("find existing value matches get", func(t *testing.T) {
		// TS source: test/find.ts lines 24-27
		// t.equal(c.find(o => o.value === 4), c.get(4))
		// Find an item whose value field is 4 — should match what Get(4) returns.
		found, foundOk := c.Find(func(v valueHolder, k int) bool {
			return v.value == 4
		})
		got, gotOk := c.Get(4)
		assertTrue(t, foundOk, "find should locate value 4")
		assertTrue(t, gotOk, "get(4) should succeed")
		assertEqual(t, found.value, got.value, "find result should match get result")
	})

	t.Run("find non-existing value returns false", func(t *testing.T) {
		// TS source: test/find.ts lines 29-32
		// t.equal(c.find(o => o.value === 9), undefined)
		// Value 9 was never set (we set 0..8), so find should fail.
		_, ok := c.Find(func(v valueHolder, k int) bool {
			return v.value == 9
		})
		assertFalse(t, ok, "find should not locate value 9")
	})

	t.Run("find value 8 returns correct object", func(t *testing.T) {
		// TS source: test/find.ts lines 34-37
		// t.same(c.find(o => o.value === 8), { value: 8 })
		// Value 8 should be in the cache.
		found, ok := c.Find(func(v valueHolder, k int) bool {
			return v.value == 8
		})
		assertTrue(t, ok, "find should locate value 8")
		assertEqual(t, found.value, 8, "found value should be 8")
	})

	// TS source: test/find.ts lines 39-47
	// SKIP: resolves[8]?.({ value: 10 }) and Promise-based assertions
	// (testing that after a fetch resolves, find returns the updated value).
	// This requires fetchMethod + async resolution, not applicable in Go.

	// TS source: test/find.ts lines 49-66
	// SKIP: c.fetch(99) and subsequent find tests for pending/resolved fetches.
	// All of these rely on fetchMethod and Promise resolution.
}

func TestFindAfterUpdate(t *testing.T) {
	// Additional sync test: verify Find works after updating a value.
	// Inspired by the TS pattern of resolves[8]?.({ value: 10 }) which updates
	// key 8's value, then finds by the new value.
	c := New[int, valueHolder](Options[int, valueHolder]{Max: 5})
	for i := 0; i < 5; i++ {
		c.Set(i, valueHolder{value: i})
	}

	// Update key 3 with a new value
	c.Set(3, valueHolder{value: 30})

	// Find by old value should fail
	_, ok := c.Find(func(v valueHolder, k int) bool {
		return v.value == 3
	})
	assertFalse(t, ok, "find old value 3 should fail after update")

	// Find by new value should succeed
	found, ok := c.Find(func(v valueHolder, k int) bool {
		return v.value == 30
	})
	assertTrue(t, ok, "find new value 30 should succeed")
	assertEqual(t, found.value, 30, "found value should be 30")

	// Verify it matches Get(3)
	got, gotOk := c.Get(3)
	assertTrue(t, gotOk, "get(3) should succeed")
	assertEqual(t, got.value, found.value, "find result should match get result")
}

func TestFindUpdatesRecency(t *testing.T) {
	// Verify that Find() updates item recency (via internal Get call).
	// This matches the TS behavior where find returns c.get(key),
	// which moves the item to the MRU position.
	c := New[int, int](Options[int, int]{Max: 3})
	c.Set(1, 10)
	c.Set(2, 20)
	c.Set(3, 30)
	// LRU order: 1, 2, 3

	// Find value 10 (key 1) — should move key 1 to MRU
	found, ok := c.Find(func(v int, k int) bool {
		return v == 10
	})
	assertTrue(t, ok, "find should locate value 10")
	assertEqual(t, found, 10, "found value should be 10")

	// Now add a new item — should evict key 2 (the new LRU), not key 1
	c.Set(4, 40)
	assertEqual(t, c.Size(), 3, "size should be 3")

	_, ok = c.Get(2)
	assertFalse(t, ok, "key 2 should have been evicted (was LRU)")

	_, ok = c.Get(1)
	assertTrue(t, ok, "key 1 should still exist (find updated its recency)")
}

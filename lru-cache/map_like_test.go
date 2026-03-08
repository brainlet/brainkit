package lrucache

// Tests ported from node-lru-cache test/map-like.ts
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/map-like.ts
//
// The original test file uses fetchMethod, clock.enter/exit, matchSnapshot,
// and expose() extensively. This port covers the concrete iteration behavior:
//   - Keys(), Values(), Entries() on empty cache
//   - Pending/resolved fetch placeholders in iteration views
//   - Fill cache, test Keys/Values/Entries/RKeys/RValues/REntries
//   - ForEach and RForEach
//   - Stale entries excluded from normal iteration but retained in Dump()
//   - ForEach/RForEach on empty cache doesn't call fn
//   - Skip only the JS-specific `this` binding behavior
//
// Test helpers (assertEqual, assertTrue, assertFalse, assertSliceEqual, etc.)
// are defined in helpers_test.go — shared across all test files.

import (
	"testing"
)

// ===========================================================================
// Map-like iteration tests
// TS source: test/map-like.ts
// ===========================================================================

func TestMapLikeIterationEmpty(t *testing.T) {
	// TS source: test/map-like.ts lines 33-41 — empty cache iteration
	// t.matchSnapshot(c.keys(), 'empty, keys')
	// t.matchSnapshot(c.values(), 'empty, values')
	// t.matchSnapshot(c.entries(), 'empty, entries')
	// t.matchSnapshot(entriesFromForeach(c), 'empty, foreach')
	// t.matchSnapshot(c.rkeys(), 'empty, rkeys')
	// t.matchSnapshot(c.rvalues(), 'empty, rvalues')
	// t.matchSnapshot(c.rentries(), 'empty, rentries')
	// t.matchSnapshot(entriesFromRForeach(c), 'empty, rforeach')
	//
	// NOTE: We skip matchSnapshot and directly assert empty results.
	// Also skipping maxSize/sizeCalculation/fetchMethod from the TS setup
	// since those are for the fetch tests we're skipping.

	c := New[int, string](Options[int, string]{Max: 5})

	// TS source: test/map-like.ts line 33 — empty, keys
	keys := c.Keys()
	assertEqual(t, len(keys), 0, "empty cache keys should be empty")

	// TS source: test/map-like.ts line 34 — empty, values
	values := c.Values()
	assertEqual(t, len(values), 0, "empty cache values should be empty")

	// TS source: test/map-like.ts line 35 — empty, entries
	entries := c.Entries()
	assertEqual(t, len(entries), 0, "empty cache entries should be empty")

	// TS source: test/map-like.ts line 37 — empty, rkeys
	rkeys := c.RKeys()
	assertEqual(t, len(rkeys), 0, "empty cache rkeys should be empty")

	// TS source: test/map-like.ts line 38 — empty, rvalues
	rvalues := c.RValues()
	assertEqual(t, len(rvalues), 0, "empty cache rvalues should be empty")

	// TS source: test/map-like.ts line 39 — empty, rentries
	rentries := c.REntries()
	assertEqual(t, len(rentries), 0, "empty cache rentries should be empty")

	// TS source: test/map-like.ts lines 36, 40 — empty, foreach / rforeach
	// Verify ForEach and RForEach on empty cache don't call the function.
	foreachCalled := false
	c.ForEach(func(v string, k int) {
		foreachCalled = true
	})
	assertFalse(t, foreachCalled, "ForEach on empty cache should not call fn")

	rforeachCalled := false
	c.RForEach(func(v string, k int) {
		rforeachCalled = true
	})
	assertFalse(t, rforeachCalled, "RForEach on empty cache should not call fn")
}

func TestMapLikeIterationFilled(t *testing.T) {
	// TS source: test/map-like.ts lines 56-58 — fill cache with items 0..2
	// for (let i = 0; i < 3; i++) { c.set(i, String(i)) }
	//
	// Then lines 73-78 — fill more items 3..7, causing evictions.
	// With max=5, after setting 0..7, only items 3,4,5,6,7 remain.
	//
	// NOTE: We skip the fetch-related steps (p99, p123, resolves) and
	// test only the sync Set/iteration behavior.

	c := New[int, string](Options[int, string]{Max: 5})

	// Fill with items 0..7 (max=5, so 0..2 get evicted)
	for i := 0; i < 8; i++ {
		c.Set(i, intToStr(i))
	}

	// TS source: test/map-like.ts lines 80-86
	// After setting 0..7 into max=5 cache:
	// MRU→LRU order: 7, 6, 5, 4, 3
	// Keys() returns MRU→LRU: [7, 6, 5, 4, 3]
	t.Run("Keys", func(t *testing.T) {
		// TS source: test/map-like.ts line 80 — t.matchSnapshot(c.keys(), 'keys')
		keys := c.Keys()
		assertSliceEqual(t, keys, []int{7, 6, 5, 4, 3}, "keys MRU→LRU")
	})

	t.Run("Values", func(t *testing.T) {
		// TS source: test/map-like.ts line 81 — t.matchSnapshot(c.values(), 'values')
		values := c.Values()
		assertSliceEqual(t, values, []string{"7", "6", "5", "4", "3"}, "values MRU→LRU")
	})

	t.Run("Entries", func(t *testing.T) {
		// TS source: test/map-like.ts line 82 — t.matchSnapshot(c.entries(), 'entries')
		entries := c.Entries()
		expected := [][2]any{
			{7, "7"},
			{6, "6"},
			{5, "5"},
			{4, "4"},
			{3, "3"},
		}
		assertEntryPairsEqual[int, string](t, entries, expected, "entries MRU→LRU")
	})

	t.Run("RKeys", func(t *testing.T) {
		// TS source: test/map-like.ts line 83 — t.matchSnapshot(c.rkeys(), 'rkeys')
		rkeys := c.RKeys()
		assertSliceEqual(t, rkeys, []int{3, 4, 5, 6, 7}, "rkeys LRU→MRU")
	})

	t.Run("RValues", func(t *testing.T) {
		// TS source: test/map-like.ts line 84 — t.matchSnapshot(c.rvalues(), 'rvalues')
		rvalues := c.RValues()
		assertSliceEqual(t, rvalues, []string{"3", "4", "5", "6", "7"}, "rvalues LRU→MRU")
	})

	t.Run("REntries", func(t *testing.T) {
		// TS source: test/map-like.ts line 85 — t.matchSnapshot(c.rentries(), 'rentries')
		rentries := c.REntries()
		expected := [][2]any{
			{3, "3"},
			{4, "4"},
			{5, "5"},
			{6, "6"},
			{7, "7"},
		}
		assertEntryPairsEqual[int, string](t, rentries, expected, "rentries LRU→MRU")
	})
}

func TestMapLikeIterationWithUpdate(t *testing.T) {
	// TS source: test/map-like.ts lines 88-95 — c.set(4, 'new value 4')
	// After updating key 4, it moves to MRU position.

	c := New[int, string](Options[int, string]{Max: 5})

	// Fill 3..7 (simulating the state after evictions)
	for i := 3; i < 8; i++ {
		c.Set(i, intToStr(i))
	}
	// LRU→MRU: 3, 4, 5, 6, 7

	// TS source: test/map-like.ts line 88 — c.set(4, 'new value 4')
	c.Set(4, "new value 4")
	// Now LRU→MRU: 3, 5, 6, 7, 4

	t.Run("Keys after update", func(t *testing.T) {
		// TS source: test/map-like.ts line 89 — t.matchSnapshot(c.keys(), 'keys, new value 4')
		// MRU→LRU: 4, 7, 6, 5, 3
		keys := c.Keys()
		assertSliceEqual(t, keys, []int{4, 7, 6, 5, 3}, "keys after update")
	})

	t.Run("Values after update", func(t *testing.T) {
		// TS source: test/map-like.ts line 90 — t.matchSnapshot(c.values(), 'values, new value 4')
		values := c.Values()
		assertSliceEqual(t, values, []string{"new value 4", "7", "6", "5", "3"}, "values after update")
	})

	t.Run("Entries after update", func(t *testing.T) {
		// TS source: test/map-like.ts line 91 — t.matchSnapshot(c.entries(), 'entries, new value 4')
		entries := c.Entries()
		expected := [][2]any{
			{4, "new value 4"},
			{7, "7"},
			{6, "6"},
			{5, "5"},
			{3, "3"},
		}
		assertEntryPairsEqual[int, string](t, entries, expected, "entries after update")
	})

	t.Run("RKeys after update", func(t *testing.T) {
		// TS source: test/map-like.ts line 92 — t.matchSnapshot(c.rkeys(), 'rkeys, new value 4')
		// LRU→MRU: 3, 5, 6, 7, 4
		rkeys := c.RKeys()
		assertSliceEqual(t, rkeys, []int{3, 5, 6, 7, 4}, "rkeys after update")
	})

	t.Run("RValues after update", func(t *testing.T) {
		// TS source: test/map-like.ts line 93 — t.matchSnapshot(c.rvalues(), 'rvalues, new value 4')
		rvalues := c.RValues()
		assertSliceEqual(t, rvalues, []string{"3", "5", "6", "7", "new value 4"}, "rvalues after update")
	})

	t.Run("REntries after update", func(t *testing.T) {
		// TS source: test/map-like.ts line 94 — t.matchSnapshot(c.rentries(), 'rentries, new value 4')
		rentries := c.REntries()
		expected := [][2]any{
			{3, "3"},
			{5, "5"},
			{6, "6"},
			{7, "7"},
			{4, "new value 4"},
		}
		assertEntryPairsEqual[int, string](t, rentries, expected, "rentries after update")
	})
}

func TestMapLikeForEach(t *testing.T) {
	// TS source: test/map-like.ts lines 114-120
	// const feArr: any[] = []
	// c.forEach((value, key) => feArr.push([value, key]))
	// t.matchSnapshot(feArr, 'forEach, no thisp')
	//
	// NOTE: Go's ForEach callback signature is fn(v V, k K), not fn(v, k, cache).
	// The TS `thisp` parameter has no Go equivalent (no `this` binding), so we
	// skip the thisp tests.

	c := New[int, string](Options[int, string]{Max: 5})
	for i := 3; i < 8; i++ {
		c.Set(i, intToStr(i))
	}
	// LRU→MRU: 3, 4, 5, 6, 7

	t.Run("ForEach MRU to LRU", func(t *testing.T) {
		// TS source: test/map-like.ts lines 114-116 — forEach collects [value, key] pairs
		type pair struct {
			value string
			key   int
		}
		var feArr []pair
		c.ForEach(func(v string, k int) {
			feArr = append(feArr, pair{v, k})
		})

		// ForEach iterates MRU→LRU (same as Keys/Values order)
		// MRU→LRU: 7, 6, 5, 4, 3
		expectedKeys := []int{7, 6, 5, 4, 3}
		expectedVals := []string{"7", "6", "5", "4", "3"}
		if len(feArr) != len(expectedKeys) {
			t.Fatalf("ForEach length mismatch: got %d, want %d", len(feArr), len(expectedKeys))
		}
		for i, e := range feArr {
			assertEqual(t, e.key, expectedKeys[i], "ForEach key")
			assertEqual(t, e.value, expectedVals[i], "ForEach value")
		}
	})

	t.Run("RForEach LRU to MRU", func(t *testing.T) {
		// TS source: test/map-like.ts lines 117-119 — rforEach collects [value, key] pairs
		type pair struct {
			value string
			key   int
		}
		var rfeArr []pair
		c.RForEach(func(v string, k int) {
			rfeArr = append(rfeArr, pair{v, k})
		})

		// RForEach iterates LRU→MRU
		// LRU→MRU: 3, 4, 5, 6, 7
		expectedKeys := []int{3, 4, 5, 6, 7}
		expectedVals := []string{"3", "4", "5", "6", "7"}
		if len(rfeArr) != len(expectedKeys) {
			t.Fatalf("RForEach length mismatch: got %d, want %d", len(rfeArr), len(expectedKeys))
		}
		for i, e := range rfeArr {
			assertEqual(t, e.key, expectedKeys[i], "RForEach key")
			assertEqual(t, e.value, expectedVals[i], "RForEach value")
		}
	})

	// TS source: test/map-like.ts lines 121-128 — forEach/rforEach with thisp
	// SKIP: Go has no `this` binding concept. The TS test verifies that
	// forEach(fn, thisp) calls fn with `this` set to thisp. This is a
	// JavaScript-specific feature with no Go equivalent.
}

func TestMapLikeForEachEmpty(t *testing.T) {
	// TS source: test/map-like.ts lines 131-136
	// const empty = new LRU({ max: 10 })
	// empty.forEach(() => { throw new Error('fail empty forEach') })
	// empty.rforEach(() => { throw new Error('fail empty rforEach') })
	//
	// Verifies that ForEach/RForEach on an empty cache does NOT call the callback.

	empty := New[int, string](Options[int, string]{Max: 10})

	// TS source: test/map-like.ts line 133 — forEach on empty should not call fn
	empty.ForEach(func(v string, k int) {
		t.Fatal("ForEach callback should not be called on empty cache")
	})

	// TS source: test/map-like.ts line 136 — rforEach on empty should not call fn
	empty.RForEach(func(v string, k int) {
		t.Fatal("RForEach callback should not be called on empty cache")
	})
}

func TestMapLikePendingFetch(t *testing.T) {
	started := make(chan int, 1)
	replies := make(chan fetchReply[string], 1)
	c := New[int, string](Options[int, string]{
		Max:     5,
		MaxSize: 5,
		SizeCalculation: func(v string, k int) int {
			return 1
		},
		FetchMethod: func(key int, stale *string, _ FetcherOptions[int, string]) (string, bool, error) {
			started <- key
			reply := <-replies
			return reply.value, reply.ok, reply.err
		},
	})

	pending := startAsyncFetch(c, 99)
	<-started

	assertEqual(t, len(c.Keys()), 0, "pending fetch keys")
	assertEqual(t, len(c.Values()), 0, "pending fetch values")
	assertEqual(t, len(c.Entries()), 0, "pending fetch entries")
	assertEqual(t, len(c.RKeys()), 0, "pending fetch rkeys")
	assertEqual(t, len(c.RValues()), 0, "pending fetch rvalues")
	assertEqual(t, len(c.REntries()), 0, "pending fetch rentries")
	assertEqual(t, len(c.Dump()), 0, "pending fetch dump")

	foreachCalled := false
	c.ForEach(func(v string, k int) { foreachCalled = true })
	assertFalse(t, foreachCalled, "pending fetch should not appear in ForEach")
	rforeachCalled := false
	c.RForEach(func(v string, k int) { rforeachCalled = true })
	assertFalse(t, rforeachCalled, "pending fetch should not appear in RForEach")

	assertTrue(t, exposeIsBackgroundFetch(c, exposeKeyMap(c)[99]), "pending slot should be background fetch")
	c.Delete(99)
	out := awaitFetchResult(t, pending)
	if out.err == nil {
		t.Fatal("expected pending fetch to fail when deleted")
	}
}

func TestMapLikeResolvedFetch(t *testing.T) {
	started := make(chan int, 1)
	replies := make(chan fetchReply[string], 1)
	c := New[int, string](Options[int, string]{
		Max:     5,
		MaxSize: 5,
		SizeCalculation: func(v string, k int) int {
			return 1
		},
		FetchMethod: func(key int, stale *string, _ FetcherOptions[int, string]) (string, bool, error) {
			started <- key
			reply := <-replies
			return reply.value, reply.ok, reply.err
		},
	})

	resolved := startAsyncFetch(c, 123)
	<-started
	replies <- fetchReply[string]{value: "123", ok: true}
	out := awaitFetchResult(t, resolved)
	assertEqual(t, out.err, error(nil), "resolved fetch error")
	assertTrue(t, out.ok, "resolved fetch ok")
	assertEqual(t, out.value, "123", "resolved fetch value")

	assertSliceEqual(t, c.Keys(), []int{123}, "resolved fetch keys")
	assertSliceEqual(t, c.Values(), []string{"123"}, "resolved fetch values")
	assertEntryPairsEqual[int, string](t, c.Entries(), [][2]any{{123, "123"}}, "resolved fetch entries")
}

func TestMapLikeStaleEntry(t *testing.T) {
	clock := newTestClock(1)
	c := New[int, string](Options[int, string]{
		Max:   5,
		TTL:   1,
		NowFn: clock.nowFn,
	})
	for i := 0; i < 3; i++ {
		c.Set(i, intToStr(i))
	}
	clock.advance(10)

	assertEqual(t, len(c.Keys()), 0, "stale keys should be hidden")
	assertEqual(t, len(c.Values()), 0, "stale values should be hidden")
	assertEqual(t, len(c.Entries()), 0, "stale entries should be hidden")
	assertEqual(t, len(c.RKeys()), 0, "stale rkeys should be hidden")
	assertEqual(t, len(c.RValues()), 0, "stale rvalues should be hidden")
	assertEqual(t, len(c.REntries()), 0, "stale rentries should be hidden")
	assertEqual(t, len(c.Dump()), 3, "dump should retain stale entries")
}

// ---------------------------------------------------------------------------
// Helper: intToStr converts an int to its string representation.
// Used to match the TS pattern: c.set(i, String(i))
// ---------------------------------------------------------------------------
func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + intToStr(-i)
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}

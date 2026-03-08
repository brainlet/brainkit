package lrucache

// purge_stale_exhaustive_test.go — Faithful 1:1 port of node-lru-cache
// test/purge-stale-exhaustive.ts.
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/purge-stale-exhaustive.ts
//
// Every test case includes a comment with the original source file and line number.
// Uses test helpers from helpers_test.go: exposeRIndexes, testClock.
//
// This is a brutal exhaustive test. For each permutation of indexes (5! = 120 orderings),
// it generates every possible arrangement of staleness (2^5 = 32 configurations) and
// verifies that purgeStale produces correct results every time.
//
// Adaptation notes:
//   - t.clock → testClock with NowFn injection
//   - clock.enter()/exit() → testClock is always active when injected via NowFn
//   - assert.deepEqual → assertSliceEqual
//   - expose(c).rindexes() → exposeRIndexes(c, allowStale)

import (
	"fmt"
	"testing"
)

// boolOpts generates all 2^n binary combinations as slices of 0s and 1s.
// TS source: test/purge-stale-exhaustive.ts lines 15-28
func boolOpts(n int) [][]int {
	// test/purge-stale-exhaustive.ts lines 16-27
	mask := 1 << n // Math.pow(2, n)
	arr := make([][]int, 0, mask)
	for i := 0; i < mask; i++ {
		// Convert (mask + i) to binary string, slice off leading 1, split into digits
		bits := make([]int, n)
		for j := 0; j < n; j++ {
			if (i>>(n-1-j))&1 == 1 {
				bits[j] = 1
			}
		}
		arr = append(arr, bits)
	}
	return arr
}

// permute generates all permutations of a slice.
// TS source: test/purge-stale-exhaustive.ts lines 30-45
func permute(arr []int) [][]int {
	// test/purge-stale-exhaustive.ts lines 35-36
	if len(arr) == 1 {
		return [][]int{{arr[0]}}
	}
	var permutations [][]int
	// test/purge-stale-exhaustive.ts lines 39-43
	for i := 0; i < len(arr); i++ {
		items := make([]int, len(arr))
		copy(items, arr)
		item := items[i]
		items = append(items[:i], items[i+1:]...)
		for _, perm := range permute(items) {
			permutations = append(permutations, append([]int{item}, perm...))
		}
	}
	return permutations
}

// permuteN generates all permutations of [0, 1, ..., n-1].
// TS source: test/purge-stale-exhaustive.ts lines 31-33
func permuteN(n int) [][]int {
	arr := make([]int, n)
	for i := 0; i < n; i++ {
		arr[i] = i
	}
	return permute(arr)
}

// runTestStep runs a single test step for the given ordering and staleness config.
// TS source: test/purge-stale-exhaustive.ts lines 47-97
func runTestStep(t *testing.T, order []int, stales []int, length int) {
	t.Helper()

	// test/purge-stale-exhaustive.ts lines 65-66: clock.enter() + new LRU
	clock := newTestClock(1) // clock.advance(1) at TS line 13 sets initial time to 1
	c := New[int, int](Options[int, int]{Max: length, TTL: 100, NowFn: clock.nowFn})

	// test/purge-stale-exhaustive.ts lines 69-75: fill array with index matching k/v
	for i := 0; i < length; i++ {
		if stales[i] == 1 {
			// test/purge-stale-exhaustive.ts line 71: c.set(i, i, { ttl: 1 })
			c.Set(i, i, SetOptions[int, int]{TTL: Int64(1)})
		} else {
			// test/purge-stale-exhaustive.ts line 73: c.set(i, i)
			c.Set(i, i)
		}
	}

	// test/purge-stale-exhaustive.ts lines 78-80: get() items to reorder
	for _, index := range order {
		c.Get(index)
	}

	// test/purge-stale-exhaustive.ts line 82: assert.deepEqual([...e.rindexes()], order)
	// Verify that rindexes matches the expected order
	rindexes := exposeRIndexes(c, false)
	assertSliceEqual(t, rindexes, order, "expected ordering after gets")

	// test/purge-stale-exhaustive.ts lines 85-86: clock.advance(10); c.purgeStale()
	// Advance clock so items with ttl:1 go stale (10 > 1)
	clock.advance(10)
	c.PurgeStale()

	// test/purge-stale-exhaustive.ts lines 87-90:
	// assert.deepEqual([...e.rindexes()], [...e.rindexes({ allowStale: true })])
	// After purging stale items, non-stale rindexes should equal rindexes with allowStale
	// (because all stale items have been removed)
	rindexesNoStale := exposeRIndexes(c, false)
	rindexesWithStale := exposeRIndexes(c, true)
	assertSliceEqual(t, rindexesNoStale, rindexesWithStale,
		"after partial purge: rindexes should equal rindexes(allowStale)")

	// test/purge-stale-exhaustive.ts lines 92-94:
	// clock.advance(100); c.purgeStale()
	// Make all items go stale (100 > remaining TTL for all)
	clock.advance(100)
	c.PurgeStale()

	// assert.deepEqual([...e.rindexes({ allowStale: true })], [])
	rindexesAfterFull := exposeRIndexes(c, true)
	assertSliceEqual(t, rindexesAfterFull, []int{},
		"after full purge: rindexes should be empty")
}

func TestPurgeStaleExhaustive(t *testing.T) {
	// test/purge-stale-exhaustive.ts lines 99-114: "exhaustive tests"
	//
	// This is a brutal test.
	// Generate every possible ordering of indexes.
	// Then for each ordering, generate every possible arrangement of staleness.
	// Verify that purgeStale produces the correct result every time.

	const length = 5

	// test/purge-stale-exhaustive.ts line 105: for (const order of permute(len))
	for _, order := range permuteN(length) {
		order := order // capture loop variable

		// test/purge-stale-exhaustive.ts line 106: name = `order=${order.join('')}`
		name := fmt.Sprintf("order=%d%d%d%d%d", order[0], order[1], order[2], order[3], order[4])

		t.Run(name, func(t *testing.T) {
			// test/purge-stale-exhaustive.ts lines 58-63:
			// When stales === -1, generate all 2^len stale configurations
			for _, stales := range boolOpts(length) {
				runTestStep(t, order, stales, length)
			}
		})
	}
}

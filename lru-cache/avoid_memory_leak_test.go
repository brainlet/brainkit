package lrucache

// avoid_memory_leak_test.go — Faithful 1:1 port of test/avoid-memory-leak.ts from node-lru-cache.
// TS source: test/avoid-memory-leak.ts (139 lines)
// GitHub issue: https://github.com/isaacs/node-lru-cache/issues/227
//
// ADAPTATION NOTES:
//
// The TS version requires --expose-gc and the v8 module to:
//   1. Force garbage collection between profiling steps
//   2. Read v8.getHeapStatistics() to detect memory growth
//   3. Verify valList.length and free.length stay bounded
//   4. Verify no detached contexts (V8-specific)
//
// The Go adaptation:
//   - Uses runtime.GC() instead of global gc()
//   - Uses runtime.MemStats instead of v8.getHeapStatistics()
//   - Uses exposeValList/exposeFree from helpers_test.go for internal inspection
//   - Runs 100K iterations (not 1M) to keep test fast while still detecting leaks
//   - Skips V8-specific checks (number_of_native_contexts, number_of_detached_contexts)
//   - Tests the same 3 configurations as TS: maxSize+max, maxSize only, max only
//
// Key invariant being tested:
//   After the cache is fully populated, valList should never grow beyond (max+1)
//   and the free stack should have at most 1 entry. If either grows unboundedly,
//   we have a memory leak in the internal data structures.

import (
	"runtime"
	"testing"
)

const (
	// TS source: test/avoid-memory-leak.ts:8-11
	// const maxSize = 100_000
	// const itemSize = 1_000
	// const profEvery = 10_000
	// const n = 1_000_000
	//
	// Go adaptation: reduced iteration count for faster testing while still
	// being sufficient to detect unbounded growth patterns.
	avoidLeakMaxSize   = 100_000
	avoidLeakItemSize  = 1_000
	avoidLeakProfEvery = 10_000
	avoidLeakN         = 100_000 // 100K vs 1M in TS — sufficient to detect leaks
)

// profile captures a snapshot of cache internals and memory stats.
// TS source: test/avoid-memory-leak.ts:39-51 (prof function)
type profile struct {
	i              int
	valListLength  int
	freeLength     int
	heapAllocBytes uint64
}

// takeProfile captures the current state for leak detection.
// TS source: test/avoid-memory-leak.ts:39-51
func takeProfile[K comparable, V any](i int, cache *LRUCache[K, V]) profile {
	// TS source: test/avoid-memory-leak.ts:44 — gc()
	// Force GC so we measure actual retained memory, not lazy GC behavior.
	runtime.GC()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return profile{
		i:              i,
		valListLength:  len(exposeValList(cache)),
		freeLength:     len(exposeFree(cache)),
		heapAllocBytes: ms.HeapAlloc,
	}
}

// nextPow2 returns the smallest power of 2 >= n (minimum 8).
// Go adaptation: Go slices grow in power-of-2 steps, so when max is not set
// (dynamic growth mode), valList length will be the next power-of-2 above the
// actual item count, not the exact item count as in TS's typed arrays.
func nextPow2(n int) int {
	if n <= 8 {
		return 8
	}
	p := 1
	for p < n {
		p *= 2
	}
	return p
}

// runAvoidLeakTest runs the memory leak detection test for a given cache configuration.
// TS source: test/avoid-memory-leak.ts:53-115 (runTest function)
func runAvoidLeakTest(t *testing.T, cache *LRUCache[int, []byte], expectItemCount int, max int) {
	t.Helper()

	// Go adaptation: When max is not explicitly set (cache.max == 0), the Go
	// implementation grows internal arrays using power-of-2 allocation (e.g.,
	// 8→16→32→64→128). The TS version uses JS arrays whose .length equals the
	// highest assigned index + 1. We adjust the bound to account for Go's
	// allocation strategy while still detecting unbounded growth.
	valListBound := max
	if cache.max == 0 {
		valListBound = nextPow2(max)
	}

	// TS source: test/avoid-memory-leak.ts:55-57
	// First, fill to expected size
	for i := 0; i < expectItemCount; i++ {
		cache.Set(i, make([]byte, avoidLeakItemSize))
	}

	keyRange := expectItemCount * 2

	// TS source: test/avoid-memory-leak.ts:60-89
	// Now start the setting and profiling
	var profiles []profile
	for i := 0; i < avoidLeakN; i++ {
		if i%avoidLeakProfEvery == 0 {
			p := takeProfile(i, cache)

			// TS source: test/avoid-memory-leak.ts:64-68
			// t.ok(profile.valListLength <= max, ...)
			// Go adaptation: use valListBound to account for power-of-2 allocation.
			if p.valListLength > valListBound {
				t.Errorf("iteration %d: valList length %d exceeds bound %d (max=%d)",
					i, p.valListLength, valListBound, max)
			}

			// TS source: test/avoid-memory-leak.ts:69-73
			// t.ok(profile.freeLength <= 1, ...)
			if p.freeLength > 1 {
				t.Errorf("iteration %d: free stack length %d exceeds 1",
					i, p.freeLength)
			}

			// Note: V8-specific checks skipped:
			// - number_of_native_contexts (TS line 74-78)
			// - number_of_detached_contexts (TS line 79-83)

			profiles = append(profiles, p)
		}

		// TS source: test/avoid-memory-leak.ts:87-88
		// const item = makeItem()
		// cache.set(i % keyRange, item)
		item := make([]byte, avoidLeakItemSize)
		cache.Set(i%keyRange, item)
	}

	// TS source: test/avoid-memory-leak.ts:91-92
	// Final profile
	finalProfile := takeProfile(avoidLeakN, cache)
	profiles = append(profiles, finalProfile)

	// TS source: test/avoid-memory-leak.ts:104-114
	// Check that memory growth is bounded after initial warmup.
	// The TS test uses total_heap_size and checks for < 2x growth.
	// We use HeapAlloc which is more precise for Go.
	//
	// Warning: kludgey inexact test! (same caveat as TS version)
	// Memory leaks can be hard to catch deterministically.
	// After the initial warmup period, heap should be relatively stable.
	// The original bug showed 10x growth; 2x threshold is aggressive
	// without risking false positives.
	start := len(profiles) / 2
	initial := profiles[start]
	for i := start; i < len(profiles); i++ {
		current := profiles[i]
		if initial.heapAllocBytes > 0 {
			delta := float64(current.heapAllocBytes) / float64(initial.heapAllocBytes)
			if delta >= 2.0 {
				t.Errorf("memory growth should not be unbounded: delta=%.2f at i=%d (current=%d, initial=%d)",
					delta, current.i, current.heapAllocBytes, initial.heapAllocBytes)
			}
		}
	}
}

// TestAvoidMemoryLeak_BothMaxAndMaxSize tests memory stability with both max and maxSize.
// TS source: test/avoid-memory-leak.ts:117-126 ("both max and maxSize")
func TestAvoidMemoryLeak_BothMaxAndMaxSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	// TS source: test/avoid-memory-leak.ts:32-34
	// const expectItemCount = Math.ceil(maxSize / itemSize)
	// const max = expectItemCount + 1
	expectItemCount := (avoidLeakMaxSize + avoidLeakItemSize - 1) / avoidLeakItemSize // ceil division
	max := expectItemCount + 1

	// TS source: test/avoid-memory-leak.ts:118-125
	cache := New[int, []byte](Options[int, []byte]{
		MaxSize: avoidLeakMaxSize,
		SizeCalculation: func(v []byte, k int) int {
			return len(v)
		},
		Max: max,
	})

	runAvoidLeakTest(t, cache, expectItemCount, max)
}

// TestAvoidMemoryLeak_NoMaxOnlyMaxSize tests memory stability with maxSize only (no max).
// TS source: test/avoid-memory-leak.ts:128-135 ("no max, only maxSize")
func TestAvoidMemoryLeak_NoMaxOnlyMaxSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	expectItemCount := (avoidLeakMaxSize + avoidLeakItemSize - 1) / avoidLeakItemSize
	max := expectItemCount + 1

	// TS source: test/avoid-memory-leak.ts:129-134
	cache := New[int, []byte](Options[int, []byte]{
		MaxSize: avoidLeakMaxSize,
		SizeCalculation: func(v []byte, k int) int {
			return len(v)
		},
	})

	runAvoidLeakTest(t, cache, expectItemCount, max)
}

// TestAvoidMemoryLeak_OnlyMaxNoMaxSize tests memory stability with max only (no maxSize).
// TS source: test/avoid-memory-leak.ts:138 ("only max, no maxSize")
func TestAvoidMemoryLeak_OnlyMaxNoMaxSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	expectItemCount := (avoidLeakMaxSize + avoidLeakItemSize - 1) / avoidLeakItemSize
	max := expectItemCount + 1

	// TS source: test/avoid-memory-leak.ts:138
	cache := New[int, []byte](Options[int, []byte]{
		Max: max,
	})

	runAvoidLeakTest(t, cache, expectItemCount, max)
}

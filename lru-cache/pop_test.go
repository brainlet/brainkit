package lrucache

// Tests ported from node-lru-cache test/pop.ts
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/pop.ts
//
// Test helpers (assertEqual, assertTrue, assertSliceEqual, etc.)
// are defined in helpers_test.go — shared across all test files.

import (
	"testing"
)

// ===========================================================================
// Pop tests
// TS source: test/pop.ts
// ===========================================================================

func TestPopLRUOrder(t *testing.T) {
	// TS source: test/pop.ts lines 4-14 (top-level test, not inside t.test)
	// Creates cache with max=5, fills with 0..4, gets(2) to move it to MRU,
	// then pops all items. Expected LRU→MRU order: [0, 1, 3, 4, 2, undefined].
	// In Go, Pop() returns (V, bool) — the last pop returns (0, false).
	cache := New[int, int](Options[int, int]{Max: 5})
	for i := 0; i < 5; i++ {
		cache.Set(i, i)
	}
	cache.Get(2) // moves 2 to MRU position

	// Pop all items — they should come out in LRU order.
	// After Get(2), LRU order is: 0, 1, 3, 4, 2
	// TS: const popped = []; do { p = cache.pop(); popped.push(p) } while (p !== undefined)
	// TS expected: [0, 1, 3, 4, 2, undefined]
	type popResult struct {
		value int
		ok    bool
	}
	var popped []popResult
	for {
		v, ok := cache.Pop()
		popped = append(popped, popResult{v, ok})
		if !ok {
			break
		}
	}

	// TS source: test/pop.ts line 14 — t.same(popped, [0, 1, 3, 4, 2, undefined])
	expected := []popResult{
		{0, true},
		{1, true},
		{3, true},
		{4, true},
		{2, true},
		{0, false}, // undefined → zero value, false
	}
	if len(popped) != len(expected) {
		t.Fatalf("popped length mismatch: got %d, want %d", len(popped), len(expected))
	}
	for i, e := range expected {
		assertEqual(t, popped[i].value, e.value, "popped value")
		assertEqual(t, popped[i].ok, e.ok, "popped ok")
	}
}

func TestPopWithBackgroundFetches(t *testing.T) {
	clock := newTestClock(1)
	started := make(chan int, 4)

	fetcher := func(key int, stale *int, opts FetcherOptions[int, int]) (int, bool, error) {
		started <- key
		<-opts.Signal.Done()
		return 0, false, nil
	}

	f := New[int, int](Options[int, int]{
		Max:         5,
		TTL:         10,
		NowFn:       clock.nowFn,
		FetchMethod: fetcher,
	})

	f.Set(0, 0, SetOptions[int, int]{TTL: Int64(0)})
	pf := startAsyncFetch(f, 1)
	<-started
	f.Set(2, 2, SetOptions[int, int]{TTL: Int64(0)})

	v, ok := f.Pop()
	assertTrue(t, ok, "first pop should succeed")
	assertEqual(t, v, 0, "first pop should return 0")
	assertEqual(t, f.Size(), 2, "size after first pop")

	v, ok = f.Pop()
	assertTrue(t, ok, "second pop should succeed")
	assertEqual(t, v, 2, "second pop should return 2 after skipping inflight fetch")
	assertEqual(t, f.Size(), 0, "size after second pop should be 0")

	out := awaitFetchResult(t, pf)
	if out.err == nil {
		t.Fatal("expected aborted fetch to return an error after pop")
	}

	f.Set(0, 0, SetOptions[int, int]{TTL: Int64(0)})
	f.Set(1, 111)
	clock.advance(20)
	pf = startAsyncFetch(f, 1)
	<-started
	f.Set(2, 2, SetOptions[int, int]{TTL: Int64(0)})

	v, ok = f.Pop()
	assertTrue(t, ok, "third pop should succeed")
	assertEqual(t, v, 0, "third pop should return 0")
	assertEqual(t, f.Size(), 2, "size after third pop")

	v, ok = f.Pop()
	assertTrue(t, ok, "fourth pop should succeed")
	assertEqual(t, v, 111, "fourth pop should return stale value from inflight fetch")
	assertEqual(t, f.Size(), 1, "size after fourth pop")

	v, ok = f.Pop()
	assertTrue(t, ok, "fifth pop should succeed")
	assertEqual(t, v, 2, "fifth pop should return 2")
	assertEqual(t, f.Size(), 0, "size after fifth pop")

	out = awaitFetchResult(t, pf)
	if out.err == nil {
		t.Fatal("expected stale inflight fetch to be aborted by pop")
	}
}

func TestPopDisposeAndDisposeAfter(t *testing.T) {
	// TS source: test/pop.ts lines 54-69 — t.test('pop calls dispose and disposeAfter')
	// Verifies that Pop() triggers both dispose and disposeAfter callbacks
	// for each popped item.
	disposeCalled := 0
	disposeAfterCalled := 0

	c := New[int, int](Options[int, int]{
		Max: 5,
		// TS source: test/pop.ts line 57 — dispose: () => disposeCalled++
		Dispose: func(value int, key int, reason DisposeReason) {
			disposeCalled++
		},
		// TS source: test/pop.ts line 58 — disposeAfter: () => disposeAfterCalled++
		DisposeAfter: func(value int, key int, reason DisposeReason) {
			disposeAfterCalled++
		},
	})

	// TS source: test/pop.ts lines 60-62
	c.Set(0, 0)
	c.Set(1, 1)
	c.Set(2, 2)

	// TS source: test/pop.ts line 63 — t.equal(c.pop(), 0)
	v, ok := c.Pop()
	assertTrue(t, ok, "pop 0 should succeed")
	assertEqual(t, v, 0, "first pop should be 0 (LRU)")

	// TS source: test/pop.ts line 64 — t.equal(c.pop(), 1)
	v, ok = c.Pop()
	assertTrue(t, ok, "pop 1 should succeed")
	assertEqual(t, v, 1, "second pop should be 1")

	// TS source: test/pop.ts line 65 — t.equal(c.pop(), 2)
	v, ok = c.Pop()
	assertTrue(t, ok, "pop 2 should succeed")
	assertEqual(t, v, 2, "third pop should be 2")

	// TS source: test/pop.ts line 66 — t.equal(c.pop(), undefined)
	v, ok = c.Pop()
	assertFalse(t, ok, "pop on empty cache should return false")
	assertEqual(t, v, 0, "pop on empty cache should return zero value")

	// TS source: test/pop.ts line 67 — t.equal(c.size, 0)
	assertEqual(t, c.Size(), 0, "size should be 0 after popping all")

	// TS source: test/pop.ts line 68 — t.equal(disposeCalled, 3)
	assertEqual(t, disposeCalled, 3, "dispose should be called 3 times")

	// TS source: test/pop.ts line 69 — t.equal(disposeAfterCalled, 3)
	assertEqual(t, disposeAfterCalled, 3, "disposeAfter should be called 3 times")
}

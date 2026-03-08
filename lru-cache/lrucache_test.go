package lrucache

// Tests ported from node-lru-cache test suite.
// TS source: test/basic.ts, test/ttl.ts, test/dispose.ts, test/size-calculation.ts

import (
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test clock for deterministic TTL testing
// TS source: test/ttl.ts uses t.clock with clock.advance()
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
// Helpers
// ---------------------------------------------------------------------------

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

// ===========================================================================
// Basic operation tests
// TS source: test/basic.ts
// ===========================================================================

func TestBasicOperation(t *testing.T) {
	// TS source: test/basic.ts "basic operation" (line 22)
	c := New[int, int](Options[int, int]{Max: 10})

	// Set 5 items
	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}

	// Get all 5
	for i := 0; i < 5; i++ {
		v, ok := c.Get(i)
		assertTrue(t, ok, "get should find item")
		assertEqual(t, v, i, "get should return correct value")
	}
	assertEqual(t, c.Size(), 5, "size after 5 sets")

	// Set 5 more (total 10 = max)
	for i := 5; i < 10; i++ {
		c.Set(i, i)
	}
	assertEqual(t, c.Size(), 10, "size after 10 sets")

	// Update items 5-9 (same value)
	for i := 5; i < 10; i++ {
		c.Set(i, i)
	}
	assertEqual(t, c.Size(), 10, "size after updates")

	// Get items 0-4 (updates their recency)
	for i := 0; i < 5; i++ {
		v, ok := c.Get(i, GetOptions[int]{UpdateAgeOnGet: Bool(true)})
		assertTrue(t, ok)
		assertEqual(t, v, i)
	}
	assertEqual(t, c.Size(), 10)

	// Get items 5-9
	for i := 5; i < 10; i++ {
		v, ok := c.Get(i)
		assertTrue(t, ok)
		assertEqual(t, v, i)
	}

	// Set 10-14, should evict 0-4 (they were accessed, but 5-9 were accessed after)
	// Wait, with the Get(0-4, updateAge) then Get(5-9), the order is:
	// LRU: 0,1,2,3,4 (oldest) ... 5,6,7,8,9 (newest)
	// Actually after Get(0-4, updateAge), 0-4 are moved to MRU end.
	// Then Get(5-9), 5-9 are moved to MRU end.
	// So LRU order is: 0,1,2,3,4,5,6,7,8,9 (0 is LRU again)
	for i := 10; i < 15; i++ {
		c.Set(i, i)
	}
	assertEqual(t, c.Size(), 10, "size after eviction")

	// Set 15-19
	for i := 15; i < 20; i++ {
		c.Set(i, i)
	}
	assertEqual(t, c.Size(), 10)

	// Items 0-9 should all be evicted
	for i := 0; i < 10; i++ {
		_, ok := c.Get(i)
		assertFalse(t, ok, "evicted items should not be found")
	}

	// Delete
	c.Clear()
	for i := 0; i < 10; i++ {
		c.Set(i, i)
	}
	assertEqual(t, c.Size(), 10)
	assertTrue(t, c.Delete(9), "delete existing key")
	assertFalse(t, c.Delete(9), "delete non-existing key")
	assertEqual(t, c.Size(), 9)

	// Clear
	c.Clear()
	assertEqual(t, c.Size(), 0, "size after clear")

	// Has
	for i := 0; i < 10; i++ {
		c.Set(i, i)
	}
	assertTrue(t, c.Has(0))
	assertTrue(t, c.Has(9))
	assertFalse(t, c.Has(10))
}

func TestBadMaxValues(t *testing.T) {
	// TS source: test/basic.ts "bad max values" (line 115)

	// Negative max
	assertPanics(t, func() {
		New[int, int](Options[int, int]{Max: -123})
	}, "negative max")

	// Zero max without other limits
	assertPanics(t, func() {
		New[int, int](Options[int, int]{Max: 0})
	}, "zero max without maxSize or ttl")

	// OK to have max=0 if maxSize is set
	sizeOnly := New[string, string](Options[string, string]{MaxSize: 100})
	if sizeOnly == nil {
		t.Fatal("should create cache with maxSize only")
	}

	// Setting size without tracking
	assertPanics(t, func() {
		ttlOnly := New[string, string](Options[string, string]{TTL: 1000, TTLAutopurge: true})
		ttlOnly.Set("foo", "bar", SetOptions[string, string]{Size: 1})
	}, "size without maxSize")

	// OK with both maxSize and ttl
	_ = New[int, int](Options[int, int]{MaxSize: 100, TTL: 1000})
}

func TestDeleteFromMiddle(t *testing.T) {
	// TS source: test/basic.ts "delete from middle, reuses that index" (line 200)
	c := New[int, int](Options[int, int]{Max: 5})

	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}

	// Delete from middle
	c.Delete(2)
	assertEqual(t, c.Size(), 4)

	// Set a new item - should reuse the freed index
	c.Set(5, 5)
	assertEqual(t, c.Size(), 5)

	// Verify all expected items exist
	for _, k := range []int{0, 1, 3, 4, 5} {
		v, ok := c.Get(k)
		assertTrue(t, ok, "should find key")
		assertEqual(t, v, k, "should have correct value")
	}

	// Key 2 should not exist
	_, ok := c.Get(2)
	assertFalse(t, ok, "deleted key should not exist")
}

func TestSetUndefined(t *testing.T) {
	// In TS, set(k, undefined) is an alias for delete(k).
	// In Go, we can't set "undefined" since V has a zero value.
	// This test verifies that Delete works correctly.
	c := New[string, string](Options[string, string]{Max: 5})
	c.Set("key", "value")
	assertTrue(t, c.Has("key"))
	c.Delete("key")
	assertFalse(t, c.Has("key"))
}

// ===========================================================================
// Status tracking tests
// TS source: test/basic.ts (status tracking throughout)
// ===========================================================================

func TestStatusTracking(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 10})

	// Set - add
	s1 := &Status[int]{}
	c.Set(1, 100, SetOptions[int, int]{Status: s1})
	assertEqual(t, s1.Set, "add", "status: first set should be add")

	// Set - update (same value)
	s2 := &Status[int]{}
	c.Set(1, 100, SetOptions[int, int]{Status: s2})
	assertEqual(t, s2.Set, "update", "status: same value should be update")

	// Set - replace (different value)
	s3 := &Status[int]{}
	c.Set(1, 200, SetOptions[int, int]{Status: s3})
	assertEqual(t, s3.Set, "replace", "status: different value should be replace")
	if s3.OldValue == nil {
		t.Error("status: replace should have OldValue")
	} else {
		assertEqual(t, *s3.OldValue, 100, "status: OldValue should be previous value")
	}

	// Get - hit
	s4 := &Status[int]{}
	v, ok := c.Get(1, GetOptions[int]{Status: s4})
	assertTrue(t, ok)
	assertEqual(t, v, 200)
	assertEqual(t, s4.Get, "hit", "status: get existing should be hit")

	// Get - miss
	s5 := &Status[int]{}
	_, ok = c.Get(999, GetOptions[int]{Status: s5})
	assertFalse(t, ok)
	assertEqual(t, s5.Get, "miss", "status: get missing should be miss")

	// Has - hit
	s6 := &Status[int]{}
	assertTrue(t, c.Has(1, HasOptions[int]{Status: s6}))
	assertEqual(t, s6.Has, "hit", "status: has existing should be hit")

	// Has - miss
	s7 := &Status[int]{}
	assertFalse(t, c.Has(999, HasOptions[int]{Status: s7}))
	assertEqual(t, s7.Has, "miss", "status: has missing should be miss")
}

// ===========================================================================
// TTL tests
// TS source: test/ttl.ts
// ===========================================================================

func TestTTLBasic(t *testing.T) {
	// TS source: test/ttl.ts "ttl tests defaults" (line 25)
	clock := newTestClock(1) // Start at 1 to avoid zero-time issues

	c := New[int, int](Options[int, int]{
		Max:           5,
		TTL:           10,
		TTLResolution: 0, // Exact staleness checks
		NowFn:         clock.nowFn,
	})

	c.Set(1, 1)
	v, ok := c.Get(1)
	assertTrue(t, ok, "1 get not stale at t=1")
	assertEqual(t, v, 1)

	clock.advance(5)
	v, ok = c.Get(1)
	assertTrue(t, ok, "1 get not stale at t=6")
	assertEqual(t, v, 1)

	remaining := c.GetRemainingTTL(1)
	assertEqual(t, remaining, int64(5), "5ms left to live")

	clock.advance(5) // t=11, TTL=10, start=1 → age=10, not yet stale (age <= ttl)
	v, ok = c.Get(1)
	assertTrue(t, ok, "1 get not stale at t=11 (age == ttl, not > ttl)")
	assertEqual(t, v, 1)

	clock.advance(1) // t=12, age=11 > ttl=10 → stale
	_, ok = c.Get(1)
	assertFalse(t, ok, "1 should be stale at t=12")
	assertEqual(t, c.Size(), 0, "stale item should be deleted")
}

func TestTTLPerItemOverride(t *testing.T) {
	// TS source: test/ttl.ts "ttl tests defaults" (line 63) - per-item TTL override
	clock := newTestClock(1)

	c := New[int, int](Options[int, int]{
		Max:           5,
		TTL:           10,
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})

	// Set item with TTL=100 (overriding cache TTL=10)
	c.Set(2, 2, SetOptions[int, int]{TTL: Int64(100)})
	clock.advance(50)
	assertTrue(t, c.Has(2), "should not be stale at t=51 (TTL=100)")
	v, ok := c.Get(2)
	assertTrue(t, ok)
	assertEqual(t, v, 2)

	clock.advance(51) // t=102, age=101 > ttl=100
	assertFalse(t, c.Has(2), "should be stale at t=102")
	_, ok = c.Get(2)
	assertFalse(t, ok)
}

func TestTTLImmortalItem(t *testing.T) {
	// TS source: test/ttl.ts "ttl tests defaults" (line 82) - immortal item (TTL=0)
	clock := newTestClock(1)

	c := New[string, bool](Options[string, bool]{
		Max:           5,
		TTL:           10,
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})

	// Set immortal item (TTL=0 means no TTL)
	c.Set("immortal", true, SetOptions[string, bool]{TTL: Int64(0)})
	clock.advance(100)
	v, ok := c.Get("immortal")
	assertTrue(t, ok, "immortal item should survive")
	assertTrue(t, v, "immortal item should be true")

	clock.advance(100)
	v, ok = c.Get("immortal")
	assertTrue(t, ok, "immortal item should survive forever")
	assertTrue(t, v)
}

func TestTTLAllowStale(t *testing.T) {
	clock := newTestClock(1)

	c := New[int, int](Options[int, int]{
		Max:           5,
		TTL:           10,
		TTLResolution: 0,
		AllowStale:    true,
		NowFn:         clock.nowFn,
	})

	c.Set(1, 100)
	clock.advance(20) // Well past TTL

	// With allowStale, stale items should still be returned
	v, ok := c.Get(1)
	assertTrue(t, ok, "allowStale should return stale item")
	assertEqual(t, v, 100, "should return the stale value")
}

func TestTTLNoDeleteOnStaleGet(t *testing.T) {
	clock := newTestClock(1)

	c := New[int, int](Options[int, int]{
		Max:                5,
		TTL:                10,
		TTLResolution:      0,
		NoDeleteOnStaleGet: true,
		NowFn:              clock.nowFn,
	})

	c.Set(1, 100)
	clock.advance(20)

	// Without allowStale, get returns nothing but item is preserved
	_, ok := c.Get(1)
	assertFalse(t, ok, "stale item not returned without allowStale")

	// But the item is still in the cache (noDeleteOnStaleGet)
	assertEqual(t, c.Size(), 1, "item should still be in cache")
}

func TestTTLUpdateAgeOnGet(t *testing.T) {
	clock := newTestClock(1)

	c := New[int, int](Options[int, int]{
		Max:           5,
		TTL:           10,
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})

	c.Set(1, 100)
	clock.advance(8) // age=8, TTL=10, not stale

	// Get with updateAgeOnGet resets the TTL
	v, ok := c.Get(1, GetOptions[int]{UpdateAgeOnGet: Bool(true)})
	assertTrue(t, ok)
	assertEqual(t, v, 100)

	clock.advance(8) // age since last get = 8, TTL=10, not stale
	v, ok = c.Get(1)
	assertTrue(t, ok, "TTL should have been reset by updateAgeOnGet")
	assertEqual(t, v, 100)
}

func TestTTLStatusTracking(t *testing.T) {
	clock := newTestClock(1)

	c := New[int, int](Options[int, int]{
		Max:           5,
		TTL:           10,
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})

	s := &Status[int]{}
	c.Set(1, 1, SetOptions[int, int]{Status: s})
	if s.TTL != 10 {
		t.Errorf("expected TTL=10, got %d", s.TTL)
	}

	clock.advance(5)
	s2 := &Status[int]{}
	c.Get(1, GetOptions[int]{Status: s2})
	if s2.RemainingTTL != 5 {
		t.Errorf("expected RemainingTTL=5, got %d", s2.RemainingTTL)
	}
}

// ===========================================================================
// Dispose tests
// TS source: test/dispose.ts
// ===========================================================================

func TestDispose(t *testing.T) {
	// TS source: test/dispose.ts "disposal" (line 5)
	type disposal struct {
		value  int
		key    int
		reason DisposeReason
	}
	var disposed []disposal

	c := New[int, int](Options[int, int]{
		Max: 5,
		Dispose: func(v int, k int, r DisposeReason) {
			disposed = append(disposed, disposal{v, k, r})
		},
	})

	// Fill cache (5 items) then add 4 more to trigger evictions
	for i := 0; i < 9; i++ {
		c.Set(i, i)
	}

	// Items 0-3 should be evicted
	if len(disposed) != 4 {
		t.Fatalf("expected 4 disposals, got %d", len(disposed))
	}
	for i := 0; i < 4; i++ {
		assertEqual(t, disposed[i].value, i, "evicted value")
		assertEqual(t, disposed[i].key, i, "evicted key")
		assertEqual(t, string(disposed[i].reason), "evict", "eviction reason")
	}

	// One more eviction
	c.Set(9, 9)
	assertEqual(t, len(disposed), 5)
	assertEqual(t, disposed[4].value, 4)
	assertEqual(t, string(disposed[4].reason), "evict")

	// Overwrite triggers dispose with "set" reason
	disposed = nil
	c.Set(5, 50) // Overwrite key=5 with new value
	if len(disposed) != 1 {
		t.Fatalf("expected 1 disposal on overwrite, got %d", len(disposed))
	}
	// Note: dispose is called with the OLD value
	assertEqual(t, disposed[0].value, 5, "disposed old value")
	assertEqual(t, disposed[0].key, 5, "disposed key")
	assertEqual(t, string(disposed[0].reason), "set")

	// Delete triggers dispose with "delete" reason
	disposed = nil
	c.Delete(5)
	if len(disposed) != 1 {
		t.Fatalf("expected 1 disposal on delete, got %d", len(disposed))
	}
	assertEqual(t, disposed[0].value, 50, "disposed value on delete")
	assertEqual(t, string(disposed[0].reason), "delete")

	// Delete non-existing key, no disposal
	disposed = nil
	c.Delete(5)
	assertEqual(t, len(disposed), 0, "no disposal for non-existing key")

	// Clear triggers dispose for all remaining items
	disposed = nil
	c.Clear()
	assertEqual(t, len(disposed), 4, "dispose all items on clear")
	for _, d := range disposed {
		assertEqual(t, string(d.reason), "delete", "clear reason should be delete")
	}
}

func TestNoDisposeOnSet(t *testing.T) {
	// TS source: test/dispose.ts "noDisposeOnSet with delete()" (line 109)
	type disposal struct {
		value any
		key   int
	}
	var disposed []disposal

	c := New[int, int](Options[int, int]{
		Max:            5,
		NoDisposeOnSet: true,
		Dispose: func(v int, k int, r DisposeReason) {
			disposed = append(disposed, disposal{v, k})
		},
	})

	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}
	// Overwrite items 0-3 with new values
	for i := 0; i < 4; i++ {
		c.Set(i, i+100)
	}
	// No disposals because noDisposeOnSet is true
	assertEqual(t, len(disposed), 0, "no disposal on set")

	// Delete still triggers dispose
	c.Delete(0)
	c.Delete(4)
	assertEqual(t, len(disposed), 2, "delete should still dispose")
	assertEqual(t, disposed[0].value, 100) // new value (0+100)
	assertEqual(t, disposed[1].value, 4)   // original value (never overwritten)
}

func TestDisposeAfter(t *testing.T) {
	type disposal struct {
		value  int
		key    int
		reason DisposeReason
	}
	var disposed []disposal

	c := New[int, int](Options[int, int]{
		Max: 3,
		DisposeAfter: func(v int, k int, r DisposeReason) {
			disposed = append(disposed, disposal{v, k, r})
		},
	})

	c.Set(1, 1)
	c.Set(2, 2)
	c.Set(3, 3)
	assertEqual(t, len(disposed), 0, "no disposals yet")

	// Adding a 4th item should evict item 1 and call disposeAfter
	c.Set(4, 4)
	assertEqual(t, len(disposed), 1, "one disposal after eviction")
	assertEqual(t, disposed[0].value, 1)
	assertEqual(t, string(disposed[0].reason), "evict")
}

// ===========================================================================
// Size calculation tests
// TS source: test/size-calculation.ts
// ===========================================================================

func TestSizeTracking(t *testing.T) {
	c := New[string, string](Options[string, string]{
		MaxSize: 100,
		SizeCalculation: func(v string, k string) int {
			return len(v)
		},
	})

	c.Set("a", "hello")       // size=5
	c.Set("b", "world")       // size=5, total=10
	c.Set("c", "testing")     // size=7, total=17

	assertEqual(t, c.Size(), 3)
	assertEqual(t, c.CalculatedSize(), 17)

	// Add a large item that pushes past maxSize
	c.Set("d", "this is a very long string that should cause evictions of earlier items to make room for itself yup")
	// That string is 96 chars. Total would be 5+5+7+96=113 > 100.
	// Items should be evicted until total <= 100.
	assertTrue(t, c.CalculatedSize() <= 100, "total size should not exceed maxSize")
}

func TestMaxEntrySize(t *testing.T) {
	c := New[string, string](Options[string, string]{
		MaxSize:      100,
		MaxEntrySize: 10,
		SizeCalculation: func(v string, k string) int {
			return len(v)
		},
	})

	c.Set("a", "hello") // size=5, fits
	assertTrue(t, c.Has("a"), "small item should be stored")

	s := &Status[string]{}
	c.Set("b", "this is way too long for max entry size", SetOptions[string, string]{Status: s})
	assertFalse(t, c.Has("b"), "oversized item should not be stored")
	assertEqual(t, s.Set, "miss", "status should be miss for oversized")
	assertTrue(t, s.MaxEntrySizeExceeded, "should flag max entry size exceeded")
}

func TestExplicitSize(t *testing.T) {
	c := New[string, string](Options[string, string]{
		MaxSize: 20,
	})

	c.Set("a", "hello", SetOptions[string, string]{Size: 5})
	c.Set("b", "world", SetOptions[string, string]{Size: 5})
	c.Set("c", "foo", SetOptions[string, string]{Size: 5})
	c.Set("d", "bar", SetOptions[string, string]{Size: 5})

	assertEqual(t, c.Size(), 4)
	assertEqual(t, c.CalculatedSize(), 20)

	// Adding another item should evict
	c.Set("e", "baz", SetOptions[string, string]{Size: 5})
	assertEqual(t, c.Size(), 4)
	assertFalse(t, c.Has("a"), "LRU item should be evicted")
	assertTrue(t, c.Has("e"), "new item should be stored")
}

// ===========================================================================
// Peek tests
// TS source: test/basic.ts (peek functionality)
// ===========================================================================

func TestPeek(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 5})

	for i := 0; i < 5; i++ {
		c.Set(i, i*10)
	}

	// Peek should return value without updating recency
	v, ok := c.Peek(0)
	assertTrue(t, ok, "peek should find item")
	assertEqual(t, v, 0, "peek should return correct value")

	// After peek(0), the order should be unchanged (0 is still LRU)
	// Adding a new item should evict 0
	c.Set(5, 50)
	_, ok = c.Get(0)
	assertFalse(t, ok, "LRU item should be evicted (peek doesn't update recency)")
}

func TestPeekStale(t *testing.T) {
	clock := newTestClock(1)
	c := New[int, int](Options[int, int]{
		Max:           5,
		TTL:           10,
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})

	c.Set(1, 100)
	clock.advance(20) // Past TTL

	// Peek without allowStale should return nothing
	_, ok := c.Peek(1)
	assertFalse(t, ok, "peek should not return stale item by default")

	// But item is still in cache (peek doesn't delete)
	assertEqual(t, c.Size(), 1, "peek should not delete stale items")

	// Peek with allowStale should return the stale value
	v, ok := c.Peek(1, PeekOptions{AllowStale: Bool(true)})
	assertTrue(t, ok, "peek with allowStale should return stale item")
	assertEqual(t, v, 100)
}

// ===========================================================================
// Pop tests
// TS source: test/pop.ts
// ===========================================================================

func TestPop(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 5})

	for i := 0; i < 5; i++ {
		c.Set(i, i*10)
	}

	// Pop should return LRU item
	v, ok := c.Pop()
	assertTrue(t, ok, "pop should return item")
	assertEqual(t, v, 0, "pop should return LRU item value")
	assertEqual(t, c.Size(), 4)

	v, ok = c.Pop()
	assertTrue(t, ok)
	assertEqual(t, v, 10) // Next LRU: key=1, value=10

	// Pop until empty
	c.Pop()
	c.Pop()
	v, ok = c.Pop()
	assertTrue(t, ok)
	assertEqual(t, v, 40) // Last item: key=4, value=40

	// Pop on empty cache
	_, ok = c.Pop()
	assertFalse(t, ok, "pop on empty cache should return false")
}

// ===========================================================================
// Find tests
// TS source: test/find.ts
// ===========================================================================

func TestFind(t *testing.T) {
	c := New[string, int](Options[string, int]{Max: 10})

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	c.Set("d", 4)

	// Find first even number
	v, ok := c.Find(func(v int, k string) bool {
		return v%2 == 0
	})
	assertTrue(t, ok, "should find even number")
	// Most recently used even is 4, then 2
	assertTrue(t, v == 4 || v == 2, "should find an even number")
}

// ===========================================================================
// ForEach tests
// ===========================================================================

func TestForEach(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 5})

	c.Set(1, 10)
	c.Set(2, 20)
	c.Set(3, 30)

	// ForEach iterates MRU to LRU
	var keys []int
	c.ForEach(func(v int, k int) {
		keys = append(keys, k)
	})

	if len(keys) != 3 {
		t.Fatalf("expected 3 items, got %d", len(keys))
	}
	// Order should be MRU to LRU: 3, 2, 1
	assertEqual(t, keys[0], 3, "first should be MRU")
	assertEqual(t, keys[1], 2)
	assertEqual(t, keys[2], 1, "last should be LRU")
}

func TestRForEach(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 5})

	c.Set(1, 10)
	c.Set(2, 20)
	c.Set(3, 30)

	// RForEach iterates LRU to MRU
	var keys []int
	c.RForEach(func(v int, k int) {
		keys = append(keys, k)
	})

	if len(keys) != 3 {
		t.Fatalf("expected 3 items, got %d", len(keys))
	}
	// Order should be LRU to MRU: 1, 2, 3
	assertEqual(t, keys[0], 1, "first should be LRU")
	assertEqual(t, keys[1], 2)
	assertEqual(t, keys[2], 3, "last should be MRU")
}

// ===========================================================================
// Keys / Values / Entries tests
// ===========================================================================

func TestKeysValuesEntries(t *testing.T) {
	c := New[string, int](Options[string, int]{Max: 5})

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	// Keys - MRU to LRU order
	keys := c.Keys()
	assertEqual(t, len(keys), 3)
	assertEqual(t, keys[0], "c")
	assertEqual(t, keys[1], "b")
	assertEqual(t, keys[2], "a")

	// Values
	vals := c.Values()
	assertEqual(t, len(vals), 3)
	assertEqual(t, vals[0], 3)
	assertEqual(t, vals[1], 2)
	assertEqual(t, vals[2], 1)

	// Entries
	entries := c.Entries()
	assertEqual(t, len(entries), 3)
	assertEqual(t, entries[0][0].(string), "c")
	assertEqual(t, entries[0][1].(int), 3)

	// Reverse
	rkeys := c.RKeys()
	assertEqual(t, rkeys[0], "a")
	assertEqual(t, rkeys[2], "c")
}

// ===========================================================================
// Dump / Load tests
// TS source: test/load.ts
// ===========================================================================

func TestDumpLoad(t *testing.T) {
	c1 := New[string, int](Options[string, int]{Max: 10})

	c1.Set("a", 1)
	c1.Set("b", 2)
	c1.Set("c", 3)

	dump := c1.Dump()
	if len(dump) != 3 {
		t.Fatalf("expected 3 dump entries, got %d", len(dump))
	}

	// Load into a new cache
	c2 := New[string, int](Options[string, int]{Max: 10})
	c2.Load(dump)

	assertEqual(t, c2.Size(), 3)
	v, ok := c2.Get("a")
	assertTrue(t, ok, "loaded cache should have 'a'")
	assertEqual(t, v, 1)

	v, ok = c2.Get("c")
	assertTrue(t, ok, "loaded cache should have 'c'")
	assertEqual(t, v, 3)
}

func TestDumpLoadWithTTL(t *testing.T) {
	clock := newTestClock(1000)

	c1 := New[string, int](Options[string, int]{
		Max:           10,
		TTL:           1000, // 1 second
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})

	c1.Set("a", 1)
	clock.advance(500) // Half TTL elapsed

	dump := c1.Dump()

	// Load into a new cache with the same clock
	c2 := New[string, int](Options[string, int]{
		Max:           10,
		TTL:           1000,
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})
	c2.Load(dump)

	// Item should still be valid (only ~500ms of TTL used)
	v, ok := c2.Get("a")
	assertTrue(t, ok, "loaded item should still be valid")
	assertEqual(t, v, 1)
}

// ===========================================================================
// PurgeStale test
// ===========================================================================

func TestPurgeStale(t *testing.T) {
	clock := newTestClock(1)

	c := New[int, int](Options[int, int]{
		Max:           10,
		TTL:           10,
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})

	c.Set(1, 1)
	c.Set(2, 2)
	clock.advance(5)
	c.Set(3, 3) // This one will have 5ms more TTL

	clock.advance(6) // Items 1,2 are now stale (age=11 > ttl=10), item 3 is not (age=6)

	deleted := c.PurgeStale()
	assertTrue(t, deleted, "should have purged something")
	assertEqual(t, c.Size(), 1, "only non-stale item should remain")
	v, ok := c.Get(3)
	assertTrue(t, ok, "item 3 should survive")
	assertEqual(t, v, 3)
}

// ===========================================================================
// Info test
// ===========================================================================

func TestInfo(t *testing.T) {
	c := New[string, int](Options[string, int]{Max: 10})

	c.Set("a", 42)
	info := c.Info("a")
	if info == nil {
		t.Fatal("info should not be nil")
	}
	assertEqual(t, info.Value, 42)

	// Non-existing key
	info = c.Info("b")
	if info != nil {
		t.Error("info for non-existing key should be nil")
	}
}

func TestInfoWithTTL(t *testing.T) {
	clock := newTestClock(100)

	c := New[string, int](Options[string, int]{
		Max:           10,
		TTL:           1000,
		TTLResolution: 0,
		NowFn:         clock.nowFn,
	})

	c.Set("a", 42)
	clock.advance(300)

	info := c.Info("a")
	if info == nil {
		t.Fatal("info should not be nil")
	}
	assertEqual(t, info.Value, 42)
	// TTL should be remaining time: 1000 - 300 = 700
	assertEqual(t, info.TTL, int64(700), "remaining TTL should be 700")
}

// ===========================================================================
// GetRemainingTTL test
// ===========================================================================

func TestGetRemainingTTL(t *testing.T) {
	// TS source: test/basic.ts line 46-47
	c := New[int, int](Options[int, int]{Max: 10})

	// No TTL set, should return "infinity" (max int64)
	c.Set(1, 1)
	remaining := c.GetRemainingTTL(1)
	assertTrue(t, remaining > 0, "no TTL should return large positive value")

	// Non-existing key
	remaining = c.GetRemainingTTL(999)
	assertEqual(t, remaining, int64(0), "non-existing key should return 0")
}

// ===========================================================================
// OnInsert callback test
// ===========================================================================

func TestOnInsert(t *testing.T) {
	type insertion struct {
		value  int
		key    int
		reason InsertReason
	}
	var inserts []insertion

	c := New[int, int](Options[int, int]{
		Max: 5,
		OnInsert: func(v int, k int, r InsertReason) {
			inserts = append(inserts, insertion{v, k, r})
		},
	})

	c.Set(1, 10)
	assertEqual(t, len(inserts), 1)
	assertEqual(t, string(inserts[0].reason), "add")

	c.Set(1, 10) // Same value
	assertEqual(t, len(inserts), 2)
	assertEqual(t, string(inserts[1].reason), "update")

	c.Set(1, 20) // Different value
	assertEqual(t, len(inserts), 3)
	assertEqual(t, string(inserts[2].reason), "replace")
}

// ===========================================================================
// Concurrency test (Go-specific, not in TS)
// ===========================================================================

func TestConcurrentAccess(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 100})

	var wg sync.WaitGroup
	const goroutines = 10
	const opsPerGoroutine = 1000

	// Concurrent writers
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				c.Set(g*opsPerGoroutine+i, i)
			}
		}(g)
	}

	// Concurrent readers
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				c.Get(g*opsPerGoroutine + i)
				c.Has(g*opsPerGoroutine + i)
			}
		}(g)
	}

	wg.Wait()
	// No panics or data races = success
	assertTrue(t, c.Size() <= 100, "should not exceed max")
}

// ===========================================================================
// TTL Autopurge test
// ===========================================================================

func TestTTLAutopurge(t *testing.T) {
	c := New[int, int](Options[int, int]{
		Max:          10,
		TTL:          50, // 50ms TTL
		TTLAutopurge: true,
	})

	c.Set(1, 100)
	c.Set(2, 200)
	assertEqual(t, c.Size(), 2)

	// Wait for TTL to expire and autopurge to fire
	time.Sleep(150 * time.Millisecond)

	// Items should have been auto-purged
	assertEqual(t, c.Size(), 0, "items should be auto-purged after TTL")
}

// ===========================================================================
// Edge cases
// ===========================================================================

func TestSingleItemCache(t *testing.T) {
	c := New[string, string](Options[string, string]{Max: 1})

	c.Set("a", "1")
	assertEqual(t, c.Size(), 1)
	v, ok := c.Get("a")
	assertTrue(t, ok)
	assertEqual(t, v, "1")

	// Adding second item evicts first
	c.Set("b", "2")
	assertEqual(t, c.Size(), 1)
	_, ok = c.Get("a")
	assertFalse(t, ok, "first item should be evicted")
	v, ok = c.Get("b")
	assertTrue(t, ok)
	assertEqual(t, v, "2")
}

func TestOverwriteValue(t *testing.T) {
	c := New[string, int](Options[string, int]{Max: 5})

	c.Set("key", 1)
	c.Set("key", 2)
	c.Set("key", 3)

	assertEqual(t, c.Size(), 1, "overwriting should not increase size")
	v, ok := c.Get("key")
	assertTrue(t, ok)
	assertEqual(t, v, 3, "should have latest value")
}

func TestEvictionOrder(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 3})

	c.Set(1, 1) // LRU
	c.Set(2, 2)
	c.Set(3, 3) // MRU

	// Access key 1, making it MRU
	c.Get(1) // Order now: 2(LRU), 3, 1(MRU)

	// Add key 4, should evict key 2 (LRU)
	c.Set(4, 4)
	assertFalse(t, c.Has(2), "key 2 should be evicted (LRU)")
	assertTrue(t, c.Has(1), "key 1 should survive (was accessed)")
	assertTrue(t, c.Has(3), "key 3 should survive")
	assertTrue(t, c.Has(4), "key 4 should be present")
}

func TestClearWithDispose(t *testing.T) {
	var disposeCount int
	c := New[int, int](Options[int, int]{
		Max: 5,
		Dispose: func(v int, k int, r DisposeReason) {
			disposeCount++
		},
	})

	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}
	assertEqual(t, disposeCount, 0, "no disposals on initial fill")

	c.Clear()
	assertEqual(t, disposeCount, 5, "all items should be disposed on clear")
}

func TestMethodChaining(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 10})
	result := c.Set(1, 1).Set(2, 2).Set(3, 3)
	assertEqual(t, result.Size(), 3, "method chaining should work")
}

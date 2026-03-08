package ttlcache

import (
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"
)

type scheduledTimer struct {
	clock   *testClock
	at      float64
	fn      func()
	stopped bool
}

func (t *scheduledTimer) Stop() bool {
	if t.stopped {
		return false
	}
	t.stopped = true
	return true
}

type testClock struct {
	now    float64
	timers []*scheduledTimer
}

func newTestClock(start float64) *testClock {
	return &testClock{now: start}
}

func (tc *testClock) nowFn() float64 {
	return tc.now
}

func (tc *testClock) afterFunc(delay time.Duration, fn func()) timerHandle {
	timer := &scheduledTimer{
		clock: tc,
		at:    tc.now + float64(delay)/float64(time.Millisecond),
		fn:    fn,
	}
	tc.timers = append(tc.timers, timer)
	return timer
}

func (tc *testClock) advance(ms int64) {
	target := tc.now + float64(ms)
	for {
		var next *scheduledTimer
		nextAt := target + 1
		for _, timer := range tc.timers {
			if timer.stopped {
				continue
			}
			if timer.at <= target && timer.at < nextAt {
				next = timer
				nextAt = timer.at
			}
		}
		if next == nil {
			break
		}
		tc.now = next.at
		next.stopped = true
		next.fn()
	}
	tc.now = target
}

func newTestCache[K comparable, V any](t *testing.T, clock *testClock, opts ...TTLCacheOptions[K, V]) *TTLCache[K, V] {
	t.Helper()
	cache, err := New(opts...)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	cache.nowFn = clock.nowFn
	cache.afterFunc = clock.afterFunc
	return cache
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func collectKeys[K comparable, V any](cache *TTLCache[K, V]) []K {
	var keys []K
	for key := range cache.Keys() {
		keys = append(keys, key)
	}
	return keys
}

func collectValues[K comparable, V comparable](cache *TTLCache[K, V]) []V {
	var values []V
	for value := range cache.Values() {
		values = append(values, value)
	}
	return values
}

func collectEntries[K comparable, V comparable](cache *TTLCache[K, V]) []Entry[K, V] {
	var entries []Entry[K, V]
	for entry := range cache.Entries() {
		entries = append(entries, entry)
	}
	return entries
}

func assertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

func assertTrue(t *testing.T, got bool, msg string) {
	t.Helper()
	if !got {
		t.Fatalf("%s: expected true", msg)
	}
}

func assertFalse(t *testing.T, got bool, msg string) {
	t.Helper()
	if got {
		t.Fatalf("%s: expected false", msg)
	}
}

func assertSliceEqual[T comparable](t *testing.T, got, want []T, msg string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

func assertEntriesEqual[K comparable, V comparable](t *testing.T, got, want []Entry[K, V], msg string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", msg, got, want)
		}
	}
}

func assertPanicsWithError(t *testing.T, want error, fn func(), msg string) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("%s: expected panic", msg)
		}
		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("%s: panic = %T (%v), want error %v", msg, recovered, recovered, want)
		}
		if !errors.Is(err, want) {
			t.Fatalf("%s: panic = %v, want %v", msg, err, want)
		}
	}()
	fn()
}

func TestBasicOperation(t *testing.T) {
	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL: durationPtr(1000 * time.Millisecond),
	})

	cache.PurgeToCapacity()
	cache.Set(1, 2)
	cache.PurgeToCapacity()

	assertTrue(t, cache.Has(1), "has before expiration")
	value, ok := cache.Get(1)
	assertTrue(t, ok, "get should find value")
	assertEqual(t, value, 2, "get should return stored value")

	clock.advance(1001)

	assertFalse(t, cache.Has(1), "has after expiration")
	_, ok = cache.Get(1)
	assertFalse(t, ok, "get after expiration")

	cache.SetTimer(10000, -1)
}

func TestUpdateAgeOnGetAndHas(t *testing.T) {
	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL:            durationPtr(1000 * time.Millisecond),
		UpdateAgeOnGet: true,
		UpdateAgeOnHas: true,
	})

	cache.Set(1, 2)
	assertEqual(t, cache.GetRemainingTTL(1), int64(1000), "initial ttl")

	clock.advance(5)
	assertEqual(t, cache.GetRemainingTTL(1), int64(995), "ttl after 5ms")

	value, ok := cache.Get(1)
	assertTrue(t, ok, "get returns value")
	assertEqual(t, value, 2, "get returns stored value")
	assertEqual(t, cache.GetRemainingTTL(1), int64(1000), "get refreshes ttl")

	assertTrue(t, cache.Has(1, HasOptions[int, int]{
		TTL: durationPtr(100 * time.Millisecond),
	}), "has returns true with override")
	assertEqual(t, cache.GetRemainingTTL(1), int64(100), "has refreshes ttl with override")
}

func TestNoUpdateTTL(t *testing.T) {
	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL:         durationPtr(1000 * time.Millisecond),
		NoUpdateTTL: true,
	})

	cache.Set(1, 2)
	assertEqual(t, cache.GetRemainingTTL(1), int64(1000), "initial ttl")

	clock.advance(5)
	assertEqual(t, cache.GetRemainingTTL(1), int64(995), "ttl after 5ms")

	cache.Set(1, 3)
	assertEqual(t, cache.GetRemainingTTL(1), int64(995), "ttl should not update on set")
}

func TestBadValues(t *testing.T) {
	_, err := New[int, int](TTLCacheOptions[int, int]{
		Max: intPtr(-1),
	})
	if !errors.Is(err, ErrMaxMustBePositive) {
		t.Fatalf("bad max: got %v, want %v", err, ErrMaxMustBePositive)
	}

	_, err = New[int, int](TTLCacheOptions[int, int]{
		TTL: durationPtr(-1 * time.Millisecond),
	})
	if !errors.Is(err, ErrTTLMustBePositiveIfSet) {
		t.Fatalf("bad ttl: got %v, want %v", err, ErrTTLMustBePositiveIfSet)
	}

	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL: durationPtr(1 * time.Millisecond),
	})
	assertPanicsWithError(t, ErrTTLMustBePositive, func() {
		cache.Set(1, 2, SetOptions[int, int]{
			TTL: durationPtr(-1 * time.Millisecond),
		})
	}, "set with invalid ttl")
}

func TestSetBehavior(t *testing.T) {
	clock := newTestClock(1)
	var disposals []Entry[string, DisposeReason]
	var disposedValues []string

	cache := newTestCache[string, string](t, clock, TTLCacheOptions[string, string]{
		TTL: durationPtr(10 * time.Millisecond),
		Max: intPtr(5),
		Dispose: func(value string, key string, reason DisposeReason) {
			disposedValues = append(disposedValues, value)
			disposals = append(disposals, Entry[string, DisposeReason]{Key: key, Value: reason})
		},
	})

	cache.Set("set", "oldval")
	cache.Set("set", "newval")
	assertSliceEqual(t, disposedValues, []string{"oldval"}, "set dispose value")
	assertEntriesEqual(t, disposals, []Entry[string, DisposeReason]{
		{Key: "set", Value: DisposeSet},
	}, "set dispose reason")

	disposedValues = nil
	disposals = nil
	cache.Set("set", "newnewval", SetOptions[string, string]{
		NoDisposeOnSet: boolPtr(true),
	})
	assertEqual(t, len(disposals), 0, "no dispose on set")

	clock.advance(5)
	assertEqual(t, cache.GetRemainingTTL("set"), int64(5), "ttl after 5ms")

	cache.Set("set", "newnewval", SetOptions[string, string]{
		NoUpdateTTL: boolPtr(true),
	})
	assertEqual(t, cache.GetRemainingTTL("set"), int64(5), "no update ttl keeps old ttl")
	assertEqual(t, len(disposals), 0, "same value should not dispose")

	cache.Set("set", "newnewval")
	assertEqual(t, cache.GetRemainingTTL("set"), int64(10), "set refreshes ttl")
	assertEqual(t, len(disposals), 0, "same value still should not dispose")

	clock.advance(3)
	cache.Set("set", "back to old val", SetOptions[string, string]{
		NoUpdateTTL: boolPtr(true),
	})
	assertEqual(t, cache.GetRemainingTTL("set"), int64(7), "ttl preserved")
	assertSliceEqual(t, disposedValues, []string{"newnewval"}, "set dispose old value")
	assertEntriesEqual(t, disposals, []Entry[string, DisposeReason]{
		{Key: "set", Value: DisposeSet},
	}, "set dispose after changed value")

	disposedValues = nil
	disposals = nil
	for i := 0; i < 5; i++ {
		k := strconv.Itoa(i)
		cache.Set(k, k)
	}
	assertSliceEqual(t, disposedValues, []string{"back to old val"}, "evict disposed value")
	assertEntriesEqual(t, disposals, []Entry[string, DisposeReason]{
		{Key: "set", Value: DisposeEvict},
	}, "evict dispose reason")

	disposedValues = nil
	disposals = nil
	cache.Set("0", "99", SetOptions[string, string]{
		NoUpdateTTL:    boolPtr(true),
		NoDisposeOnSet: boolPtr(true),
	})
	assertEqual(t, len(disposals), 0, "overwrite without dispose")

	clock.advance(11)
	assertSliceEqual(t, disposedValues, []string{"99", "1", "2", "3", "4"}, "stale dispose values")
	assertEntriesEqual(t, disposals, []Entry[string, DisposeReason]{
		{Key: "0", Value: DisposeStale},
		{Key: "1", Value: DisposeStale},
		{Key: "2", Value: DisposeStale},
		{Key: "3", Value: DisposeStale},
		{Key: "4", Value: DisposeStale},
	}, "stale dispose reasons")

	disposedValues = nil
	disposals = nil
	cache.Set("key", "val", SetOptions[string, string]{
		TTL: durationPtr(1000 * time.Millisecond),
	})
	for i := 0; i < 5; i++ {
		k := strconv.Itoa(i)
		cache.Set(k, k, SetOptions[string, string]{
			TTL: durationPtr(1000 * time.Millisecond),
		})
		clock.advance(1)
	}
	assertSliceEqual(t, disposedValues, []string{"val"}, "evict oldest by expiration")
	assertEntriesEqual(t, disposals, []Entry[string, DisposeReason]{
		{Key: "key", Value: DisposeEvict},
	}, "evict earliest expiration")
}

func TestDelete(t *testing.T) {
	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL: durationPtr(10 * time.Millisecond),
	})

	cache.Set(0, 0)
	cache.Set(1, 1)
	cache.Set(2, 2, SetOptions[int, int]{
		TTL: durationPtr(0),
	})

	assertTrue(t, cache.Delete(2), "delete immortal")
	assertTrue(t, cache.Delete(0), "delete finite")
	_, ok := cache.Get(0)
	assertFalse(t, ok, "deleted item should be gone")
	assertFalse(t, cache.Has(0), "deleted item should not exist")

	value, ok := cache.Get(1)
	assertTrue(t, ok, "other item should remain")
	assertEqual(t, value, 1, "other item value")

	assertTrue(t, cache.Delete(1), "delete last item")
	assertFalse(t, cache.Delete(0), "delete missing item")
	assertEqual(t, cache.GetRemainingTTL(0), int64(0), "missing ttl")
}

func TestIteratorsAndInfinityIteration(t *testing.T) {
	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL: durationPtr(10 * time.Millisecond),
	})

	for i := 0; i < 3; i++ {
		cache.Set(i, i*2)
	}

	assertEntriesEqual(t, collectEntries(cache), []Entry[int, int]{
		{Key: 0, Value: 0},
		{Key: 1, Value: 2},
		{Key: 2, Value: 4},
	}, "entries iteration")
	assertSliceEqual(t, collectKeys(cache), []int{0, 1, 2}, "key iteration")
	assertSliceEqual(t, collectValues(cache), []int{0, 2, 4}, "value iteration")

	clock2 := newTestClock(1)
	cache2 := newTestCache[int, int](t, clock2, TTLCacheOptions[int, int]{
		Max: intPtr(10),
		TTL: durationPtr(1000 * time.Millisecond),
	})
	cache2.Set(1, 11, SetOptions[int, int]{TTL: durationPtr(0)})
	cache2.Set(2, 22, SetOptions[int, int]{TTL: durationPtr(1000 * time.Millisecond)})

	assertSliceEqual(t, collectKeys(cache2), []int{2, 1}, "mortal keys come before immortal keys")
	assertEntriesEqual(t, collectEntries(cache2), []Entry[int, int]{
		{Key: 2, Value: 22},
		{Key: 1, Value: 11},
	}, "mortal entries come before immortal entries")
	assertSliceEqual(t, collectValues(cache2), []int{22, 11}, "mortal values come before immortal values")
}

func TestClearAndGracefulTimerCancellation(t *testing.T) {
	clock := newTestClock(1)
	var disposals []Entry[int, DisposeReason]
	var values []int

	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL: durationPtr(10 * time.Millisecond),
		Dispose: func(value int, key int, reason DisposeReason) {
			values = append(values, value)
			disposals = append(disposals, Entry[int, DisposeReason]{Key: key, Value: reason})
		},
	})

	for i := 0; i < 3; i++ {
		cache.Set(i, i*2)
	}
	assertTrue(t, cache.timer != nil, "timer should exist after set")

	cache.Clear()

	assertEqual(t, cache.Size(), 0, "size after clear")
	assertTrue(t, cache.timer == nil, "timer cancelled after clear")
	assertSliceEqual(t, values, []int{0, 2, 4}, "clear dispose values")
	assertEntriesEqual(t, disposals, []Entry[int, DisposeReason]{
		{Key: 0, Value: DisposeDelete},
		{Key: 1, Value: DisposeDelete},
		{Key: 2, Value: DisposeDelete},
	}, "clear dispose reasons")

	longTTL := 1_000_000_000 * time.Millisecond
	cache.Set(1, 1, SetOptions[int, int]{TTL: durationPtr(longTTL)})
	assertTrue(t, cache.timer != nil, "timer should exist after long ttl set")
	assertTrue(t, cache.Delete(1), "delete last key")
	assertTrue(t, cache.timer == nil, "timer cancelled after delete")
}

func TestSetTTLExplicit(t *testing.T) {
	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL: durationPtr(10 * time.Millisecond),
	})

	cache.Set(1, 1)
	assertEqual(t, cache.GetRemainingTTL(1), int64(10), "default ttl")

	cache.SetTTL(1, 1000*time.Millisecond)
	assertEqual(t, cache.GetRemainingTTL(1), int64(1000), "set ttl explicitly")

	cache.SetTTL(1)
	assertEqual(t, cache.GetRemainingTTL(1), int64(10), "reset to default ttl")
}

func TestConstructorNoArg(t *testing.T) {
	cache, err := New[int, int]()
	if err != nil {
		t.Fatalf("new without opts: %v", err)
	}
	if cache.ttl != nil {
		t.Fatalf("ttl without opts: got %v, want nil", cache.ttl)
	}
}

func TestValidateAgeWhenTimerHasNotFired(t *testing.T) {
	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL:           durationPtr(10 * time.Millisecond),
		CheckAgeOnGet: true,
		CheckAgeOnHas: true,
	})

	cache.Set(1, 1)
	cache.Set(2, 2)

	value, ok := cache.Get(1)
	assertTrue(t, ok, "get before timer cancel")
	assertEqual(t, value, 1, "value before timer cancel")

	cache.CancelTimer()
	clock.advance(1000)

	assertEqual(t, cache.Size(), 2, "entries still present without timer")
	value, ok = cache.Get(1, GetOptions[int, int]{CheckAgeOnGet: boolPtr(false)})
	assertTrue(t, ok, "get without age check should return stale value")
	assertEqual(t, value, 1, "stale value returned when not checking age")
	assertTrue(t, cache.Has(2, HasOptions[int, int]{CheckAgeOnHas: boolPtr(false)}), "has without age check should return true")

	_, ok = cache.Get(1)
	assertFalse(t, ok, "get with age check should delete stale item")
	assertEqual(t, cache.Size(), 1, "one item left after stale get")
	assertFalse(t, cache.Has(2), "has with age check should delete stale item")
	assertEqual(t, cache.Size(), 0, "all stale items removed")

	_, ok = cache.Get(1, GetOptions[int, int]{CheckAgeOnGet: boolPtr(false)})
	assertFalse(t, ok, "deleted item stays deleted")
	assertFalse(t, cache.Has(2, HasOptions[int, int]{CheckAgeOnHas: boolPtr(false)}), "deleted item stays deleted")
}

func TestImmortality(t *testing.T) {
	clock := newTestClock(1)
	cache := newTestCache[int, int](t, clock, TTLCacheOptions[int, int]{
		TTL: durationPtr(0),
	})

	cache.Set(1, 1, SetOptions[int, int]{TTL: durationPtr(0)})
	assertEqual(t, cache.GetRemainingTTL(1), int64(^uint64(0)>>1), "immortal ttl is maxint64")

	cache.Set(2, 2, SetOptions[int, int]{TTL: durationPtr(100 * time.Millisecond)})
	assertEqual(t, cache.GetRemainingTTL(2), int64(100), "finite ttl remains finite")
	assertSliceEqual(t, cache.expirationOrder, []int64{101}, "only finite key is scheduled")
	assertSliceEqual(t, cache.expirations[101], []int{2}, "finite key expiration bucket")

	clock.advance(200)
	assertSliceEqual(t, collectKeys(cache), []int{1}, "finite key purged, immortal remains")

	cache.Set(1, 2, SetOptions[int, int]{TTL: durationPtr(100 * time.Millisecond)})
	assertSliceEqual(t, cache.expirationOrder, []int64{301}, "immortal key becomes finite")
	assertSliceEqual(t, cache.expirations[301], []int{1}, "finite bucket after overwrite")
}

func TestLongTTL(t *testing.T) {
	clock := newTestClock(2)
	cache := newTestCache[string, int](t, clock, TTLCacheOptions[string, int]{
		Max: intPtr(10),
		TTL: durationPtr(30 * 24 * time.Hour),
	})

	cache.Set("a", 1)
	cache.Set("b", 2, SetOptions[string, int]{
		TTL: durationPtr(60 * 24 * time.Hour),
	})

	assertSliceEqual(t, collectKeys(cache), []string{"a", "b"}, "initial long ttl order")

	clock.advance(timerMaxMs)
	assertSliceEqual(t, collectKeys(cache), []string{"a", "b"}, "still alive after timer max")

	clock.advance(1000)
	assertSliceEqual(t, collectKeys(cache), []string{"a", "b"}, "still alive shortly after timer max")

	clock.advance(cache.GetRemainingTTL("a") + 1)
	assertSliceEqual(t, collectKeys(cache), []string{"b"}, "shorter long ttl expires first")
}

func TestSetWhileDisposeEviction(t *testing.T) {
	clock := newTestClock(1)
	didReset := false
	cache := newTestCache[string, string](t, clock, TTLCacheOptions[string, string]{
		Max:            intPtr(2),
		TTL:            durationPtr(1000 * time.Millisecond),
		NoDisposeOnSet: true,
	})

	cache.dispose = func(value string, key string, reason DisposeReason) {
		assertEqual(t, reason, DisposeEvict, "eviction reason")
		if !didReset {
			assertEqual(t, key, "key", "first evicted key")
			assertEqual(t, value, "val", "first evicted value")
			didReset = true
			cache.Set("key", "otherval")
			return
		}
		assertEqual(t, key, "x", "second evicted key")
		assertEqual(t, value, "y", "second evicted value")
	}

	cache.Set("key", "val")
	clock.advance(1)
	cache.Set("x", "y")
	clock.advance(1)
	cache.Set("a", "b")

	assertSliceEqual(t, cache.expirationOrder, []int64{1003}, "single expiration bucket after reset")
	assertSliceEqual(t, cache.expirations[1003], []string{"a", "key"}, "reset key appended to current bucket")
	assertSliceEqual(t, collectKeys(cache), []string{"a", "key"}, "iteration order after eviction reset")
	value, ok := cache.Get("key")
	assertTrue(t, ok, "reset key should exist")
	assertEqual(t, value, "otherval", "reset key value")
}

func TestSetWhileDisposeStale(t *testing.T) {
	clock := newTestClock(3)
	didReset := false
	cache := newTestCache[string, string](t, clock, TTLCacheOptions[string, string]{
		TTL:            durationPtr(2 * time.Millisecond),
		NoDisposeOnSet: true,
	})

	cache.dispose = func(value string, key string, reason DisposeReason) {
		assertEqual(t, reason, DisposeStale, "stale reason")
		if !didReset {
			assertEqual(t, key, "key", "first stale key")
			assertEqual(t, value, "val", "first stale value")
			didReset = true
			cache.Set("key", "otherval")
			return
		}
		assertEqual(t, key, "x", "second stale key")
		assertEqual(t, value, "y", "second stale value")
	}

	cache.Set("key", "val")
	clock.advance(1)
	cache.Set("x", "y")
	clock.advance(1)
	cache.Set("a", "b")
	clock.advance(1)

	assertSliceEqual(t, cache.expirationOrder, []int64{7}, "single expiration bucket after stale reset")
	assertSliceEqual(t, cache.expirations[7], []string{"key", "a"}, "reset key keeps bucket order")
	assertSliceEqual(t, collectKeys(cache), []string{"key", "a"}, "iteration order after stale reset")
	value, ok := cache.Get("key")
	assertTrue(t, ok, "reset stale key should exist")
	assertEqual(t, value, "otherval", "reset stale key value")
}

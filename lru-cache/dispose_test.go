package lrucache

// dispose_test.go — Faithful 1:1 port of node-lru-cache test/dispose.ts
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/dispose.ts
//
// Uses helpers from helpers_test.go: assertEqual, assertTrue, assertFalse,
// disposal, assertDisposals, testClock, newTestClock.
//
// IMPORTANT: The TS dispose/disposeAfter callback signature is (value, key, reason).
// In the TS test source, params are sometimes named (k, v, r) but the FIRST arg
// is always the VALUE and the SECOND arg is always the KEY, per the LRUCache API.
// Our Go Dispose callback matches: func(value V, key K, reason DisposeReason).

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// TestDisposeTS_Disposal — port of t.test('disposal', ...) from test/dispose.ts:5-107
// ---------------------------------------------------------------------------

func TestDisposeTS_Disposal(t *testing.T) {
	// test/dispose.ts:5-107 — 'disposal'
	//
	// This test uses mixed key/value types (int and string) in the original TS.
	// In Go, we use any for both K and V to accommodate the type mixing.
	// The test builds on prior cache state sequentially, matching the TS original.

	// test/dispose.ts:6 — const disposed: any[] = []
	var disposed []disposal[any, any]

	// test/dispose.ts:7-10 — new LRU({ max: 5, dispose: (k, v, r) => disposed.push([k, v, r]) })
	// NOTE: In the TS source, the callback params are named (k, v, r) but per the
	// LRUCache API, the first arg is VALUE and second is KEY. So k=value, v=key.
	// The disposed array stores [value, key, reason] matching the push order.
	c := New[any, any](Options[any, any]{
		Max: 5,
		Dispose: func(value any, key any, reason DisposeReason) {
			disposed = append(disposed, disposal[any, any]{value: value, key: key, reason: reason})
		},
	})

	// test/dispose.ts:11-13 — for (let i = 0; i < 9; i++) { c.set(i, i) }
	for i := 0; i < 9; i++ {
		c.Set(i, i)
	}

	// test/dispose.ts:14-19 — t.strictSame(disposed, [[0,0,'evict'],[1,1,'evict'],[2,2,'evict'],[3,3,'evict']])
	// With max=5 and 9 items set, items 0-3 are evicted.
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 0, key: 0, reason: DisposeEvict},
		{value: 1, key: 1, reason: DisposeEvict},
		{value: 2, key: 2, reason: DisposeEvict},
		{value: 3, key: 3, reason: DisposeEvict},
	}, "after setting 0-8")

	// test/dispose.ts:20 — t.equal(c.size, 5)
	assertEqual(t, c.Size(), 5, "size after 9 sets")

	// test/dispose.ts:22-29 — c.set(9, 9); check disposed includes [4,4,'evict']
	c.Set(9, 9)
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 0, key: 0, reason: DisposeEvict},
		{value: 1, key: 1, reason: DisposeEvict},
		{value: 2, key: 2, reason: DisposeEvict},
		{value: 3, key: 3, reason: DisposeEvict},
		{value: 4, key: 4, reason: DisposeEvict},
	}, "after setting 9")

	// test/dispose.ts:31 — disposed.length = 0
	disposed = disposed[:0]

	// test/dispose.ts:32-37 — c.set('asdf', 'foo'); c.set('asdf', 'asdf')
	// Setting 'asdf' evicts LRU (5). Then overwriting 'asdf' disposes old value 'foo'.
	// The disposed callback receives (value='foo', key='asdf', reason='set').
	// In the TS push: [k=value, v=key, r=reason] => ['foo', 'asdf', 'set'].
	// But the TS expected is: [[5, 5, 'evict'], ['foo', 'asdf', 'set']]
	// So the first entry is eviction of key=5,value=5.
	c.Set("asdf", "foo")
	c.Set("asdf", "asdf")
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 5, key: 5, reason: DisposeEvict},
		{value: "foo", key: "asdf", reason: DisposeSet},
	}, "after set asdf twice")

	// test/dispose.ts:39-49 — disposed.length = 0; set 0-4 again
	disposed = disposed[:0]
	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}
	// Cache was [6,7,8,9,asdf]. Setting 0 evicts 6, setting 1 evicts 7, etc.
	// Setting 4 evicts 'asdf'.
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 6, key: 6, reason: DisposeEvict},
		{value: 7, key: 7, reason: DisposeEvict},
		{value: 8, key: 8, reason: DisposeEvict},
		{value: 9, key: 9, reason: DisposeEvict},
		{value: "asdf", key: "asdf", reason: DisposeEvict},
	}, "after re-setting 0-4")

	// test/dispose.ts:51-58 — dispose both old and current
	disposed = disposed[:0]
	c.Set("asdf", "foo")
	c.Delete("asdf")
	// Setting 'asdf' evicts LRU (0). Deleting 'asdf' disposes value 'foo'.
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 0, key: 0, reason: DisposeEvict},
		{value: "foo", key: "asdf", reason: DisposeDelete},
	}, "dispose both old and current")

	// test/dispose.ts:60-63 — delete non-existing key, no disposal
	disposed = disposed[:0]
	c.Delete("asdf")
	assertDisposals(t, disposed, []disposal[any, any]{}, "delete non-existing key")

	// test/dispose.ts:65-73 — delete via clear()
	disposed = disposed[:0]
	c.Clear()
	// Cache had [1,2,3,4] (0 was evicted, asdf was deleted).
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 1, key: 1, reason: DisposeDelete},
		{value: 2, key: 2, reason: DisposeDelete},
		{value: 3, key: 3, reason: DisposeDelete},
		{value: 4, key: 4, reason: DisposeDelete},
	}, "delete via clear()")

	// test/dispose.ts:75-79 — set, get, delete
	disposed = disposed[:0]
	c.Set(3, 3)
	v, ok := c.Get(3)
	assertTrue(t, ok, "get 3 should succeed")
	assertEqual(t, v, any(3), "get 3 value")
	c.Delete(3)
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 3, key: 3, reason: DisposeDelete},
	}, "set-get-delete")

	// test/dispose.ts:81-93 — disposed because of being overwritten
	c.Clear()
	disposed = disposed[:0]
	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}
	c.Set(2, "two")
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 2, key: 2, reason: DisposeSet},
	}, "overwrite key 2")

	// test/dispose.ts:89-91 — verify values
	for i := 0; i < 5; i++ {
		v, ok := c.Get(i)
		assertTrue(t, ok, fmt.Sprintf("get %d should succeed", i))
		if i == 2 {
			assertEqual(t, v, any("two"), "get 2 should return 'two'")
		} else {
			assertEqual(t, v, any(i), fmt.Sprintf("get %d value", i))
		}
	}
	// test/dispose.ts:92 — verify disposed unchanged
	assertDisposals(t, disposed, []disposal[any, any]{
		{value: 2, key: 2, reason: DisposeSet},
	}, "disposed unchanged after gets")

	// test/dispose.ts:94-104 — noDisposeOnSet = true
	// In TS: c.noDisposeOnSet = true (direct field mutation).
	// In Go: we access the unexported field since test is in same package.
	c.noDisposeOnSet = true
	c.Clear()
	disposed = disposed[:0]
	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}
	c.Set(2, "two")
	// With noDisposeOnSet, overwriting does NOT call dispose.
	for i := 0; i < 5; i++ {
		v, ok := c.Get(i)
		assertTrue(t, ok, fmt.Sprintf("noDisposeOnSet: get %d should succeed", i))
		if i == 2 {
			assertEqual(t, v, any("two"), "noDisposeOnSet: get 2 should return 'two'")
		} else {
			assertEqual(t, v, any(i), fmt.Sprintf("noDisposeOnSet: get %d value", i))
		}
	}
	// test/dispose.ts:104 — t.strictSame(disposed, [])
	assertDisposals(t, disposed, []disposal[any, any]{}, "noDisposeOnSet should suppress dispose on overwrite")
}

// ---------------------------------------------------------------------------
// TestDisposeTS_NoDisposeOnSetWithDelete — port of t.test('noDisposeOnSet with delete()', ...)
// test/dispose.ts:109-154
// ---------------------------------------------------------------------------

func TestDisposeTS_NoDisposeOnSetWithDelete(t *testing.T) {
	// test/dispose.ts:109-154 — 'noDisposeOnSet with delete()'

	// test/dispose.ts:110-111 — disposed collects [value, key] pairs (no reason).
	// The TS callback is: (v, k) => disposed.push([v, k]) where v=value, k=key.
	// We track disposals with vkPair since reason is not checked per-entry
	// in the first part (but still recorded).
	var disposed []vkPair

	// test/dispose.ts:113 — new LRU({ max: 5, dispose, noDisposeOnSet: true })
	c := New[int, any](Options[int, any]{
		Max:            5,
		NoDisposeOnSet: true,
		Dispose: func(value any, key int, reason DisposeReason) {
			disposed = append(disposed, vkPair{value: value, key: key})
		},
	})

	// test/dispose.ts:114-116 — set 0-4
	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}
	// test/dispose.ts:117-119 — overwrite 0-3 with "new X"
	for i := 0; i < 4; i++ {
		c.Set(i, fmt.Sprintf("new %d", i))
	}
	// test/dispose.ts:120 — noDisposeOnSet, so no disposals yet
	assertVKPairs(t, disposed, []vkPair{}, "noDisposeOnSet: no disposals on overwrite")

	// test/dispose.ts:121-126 — delete(0) and delete(4)
	c.Delete(0)
	c.Delete(4)
	assertVKPairs(t, disposed, []vkPair{
		{value: "new 0", key: 0},
		{value: 4, key: 4},
	}, "noDisposeOnSet: delete disposes current value")

	// test/dispose.ts:127 — disposed.length = 0
	disposed = disposed[:0]

	// test/dispose.ts:129-141 — new cache WITHOUT noDisposeOnSet
	d := New[int, any](Options[int, any]{
		Max: 5,
		Dispose: func(value any, key int, reason DisposeReason) {
			disposed = append(disposed, vkPair{value: value, key: key})
		},
	})

	// test/dispose.ts:130-132 — set 0-4
	for i := 0; i < 5; i++ {
		d.Set(i, i)
	}
	// test/dispose.ts:133-135 — overwrite 0-3 with "new X"
	for i := 0; i < 4; i++ {
		d.Set(i, fmt.Sprintf("new %d", i))
	}
	// test/dispose.ts:136-141 — without noDisposeOnSet, overwriting disposes old values
	assertVKPairs(t, disposed, []vkPair{
		{value: 0, key: 0},
		{value: 1, key: 1},
		{value: 2, key: 2},
		{value: 3, key: 3},
	}, "without noDisposeOnSet: overwrite disposes old values")

	// test/dispose.ts:142-151 — delete(0) and delete(4)
	d.Delete(0)
	d.Delete(4)
	assertVKPairs(t, disposed, []vkPair{
		{value: 0, key: 0},
		{value: 1, key: 1},
		{value: 2, key: 2},
		{value: 3, key: 3},
		{value: "new 0", key: 0},
		{value: 4, key: 4},
	}, "without noDisposeOnSet: delete also disposes")
}

// vkPair is a value/key pair for tracking disposals without reason.
// Used in TestNoDisposeOnSetWithDelete where the TS callback only records (value, key).
type vkPair struct {
	value any
	key   any
}

// assertVKPairs compares slices of value/key pairs using fmt.Sprint for comparison.
// This is needed because the values can be int or string (any type).
func assertVKPairs(t *testing.T, got, want []vkPair, msg ...string) {
	t.Helper()
	prefix := ""
	if len(msg) > 0 {
		prefix = msg[0] + ": "
	}
	if len(got) != len(want) {
		t.Errorf("%svkPairs length mismatch: got %d, want %d\n  got:  %v\n  want: %v",
			prefix, len(got), len(want), got, want)
		return
	}
	for i := range want {
		if fmt.Sprint(got[i].value) != fmt.Sprint(want[i].value) ||
			fmt.Sprint(got[i].key) != fmt.Sprint(want[i].key) {
			t.Errorf("%svkPairs[%d]: got {%v, %v}, want {%v, %v}",
				prefix, i, got[i].value, got[i].key, want[i].value, want[i].key)
			return
		}
	}
}

// ---------------------------------------------------------------------------
// TestDisposeTS_DisposeAfter — port of t.test('disposeAfter', ...)
// test/dispose.ts:156-208
// ---------------------------------------------------------------------------

func TestDisposeTS_DisposeAfter(t *testing.T) {
	// test/dispose.ts:156-208 — 'disposeAfter'
	//
	// This test exercises re-entrancy: the disposeAfter callback calls c.Set()
	// on the cache. This is safe because disposeAfter is called after lock release.
	// Whenever key 2 is disposed, the callback re-inserts it with value+1
	// using noDisposeOnSet: true (to avoid infinite recursion).

	// test/dispose.ts:157-165 — create cache with disposeAfter
	var c *LRUCache[int, int]
	c = New[int, int](Options[int, int]{
		Max: 5,
		DisposeAfter: func(value int, key int, reason DisposeReason) {
			// test/dispose.ts:160-163 — if (k === 2) { c.set(k, v+1, { noDisposeOnSet: true }) }
			if key == 2 {
				// Increment value every time key 2 is disposed, but only one time
				// (noDisposeOnSet prevents the set from triggering another dispose).
				c.Set(key, value+1, SetOptions[int, int]{NoDisposeOnSet: Bool(true)})
			}
		},
	})

	// test/dispose.ts:167-169 — for (let i = 0; i < 100; i++) { c.set(i, i) }
	for i := 0; i < 100; i++ {
		c.Set(i, i)
	}

	// test/dispose.ts:170-179 — t.same([...c.entries()], [[99,99],[98,98],[2,21],[97,97],[96,96]])
	entries := c.Entries()
	assertEntryPairsEqual[int, int](t, entries, [][2]any{
		{99, 99},
		{98, 98},
		{2, 21},
		{97, 97},
		{96, 96},
	}, "entries after 100 sets")

	// test/dispose.ts:180 — c.delete(2)
	c.Delete(2)

	// test/dispose.ts:181-190 — after delete(2), disposeAfter re-inserts 2 with value+1=22
	entries = c.Entries()
	assertEntryPairsEqual[int, int](t, entries, [][2]any{
		{2, 22},
		{99, 99},
		{98, 98},
		{97, 97},
		{96, 96},
	}, "entries after delete(2)")

	// test/dispose.ts:191-193 — set 96-99 with value+1
	for i := 96; i < 100; i++ {
		c.Set(i, i+1)
	}

	// test/dispose.ts:194-203
	entries = c.Entries()
	assertEntryPairsEqual[int, int](t, entries, [][2]any{
		{99, 100},
		{98, 99},
		{97, 98},
		{96, 97},
		{2, 22},
	}, "entries after updating 96-99")

	// test/dispose.ts:204 — c.clear()
	// Clearing disposes all entries. Key 2 is disposed, disposeAfter re-inserts it with value+1=23.
	c.Clear()

	// test/dispose.ts:205 — t.same([...c.entries()], [[2, 23]])
	entries = c.Entries()
	assertEntryPairsEqual[int, int](t, entries, [][2]any{
		{2, 23},
	}, "entries after clear — key 2 re-inserted by disposeAfter")
}

// ---------------------------------------------------------------------------
// TestDisposeTS_ExpirationReflectedInDisposeReason — port of
// t.test('expiration reflected in dispose reason', async t => ...)
// test/dispose.ts:210-257
// ---------------------------------------------------------------------------

func TestDisposeTS_ExpirationReflectedInDisposeReason(t *testing.T) {
	// test/dispose.ts:210-257 — 'expiration reflected in dispose reason'
	//
	// Uses a test clock to control time advancement for TTL expiration.
	// Verifies that expired items receive DisposeExpire reason, not DisposeDelete.

	// test/dispose.ts:211-212 — t.clock.enter(); t.clock.advance(1)
	clock := newTestClock(1)

	// test/dispose.ts:213 — const disposes: [number, number, LRUCache.DisposeReason][] = []
	var disposes []disposal[int, int]

	// test/dispose.ts:214-218 — new LRUCache({ ttl: 100, max: 5, dispose: (v, k, r) => disposes.push([k, v, r]) })
	// NOTE: The TS callback params are (v, k, r) = (value, key, reason) per the API.
	// But the push is [k, v, r] = [key, value, reason]. In our disposal struct, we
	// store key and value separately and compare accordingly.
	c := New[int, int](Options[int, int]{
		TTL:   100,
		Max:   5,
		NowFn: clock.nowFn,
		Dispose: func(value int, key int, reason DisposeReason) {
			// test/dispose.ts:217 — disposes.push([k, v, r]) where k=key(2nd param), v=value(1st param)
			// Our disposal struct stores {value, key, reason} matching Go callback param order.
			// But the TS expected arrays are [key, value, reason], so we store to match:
			// disposal{value: key, key: value, ...} — NO! We should store consistently.
			// Let's just use a struct that matches the TS expected format: [key, value, reason].
			disposes = append(disposes, disposal[int, int]{value: value, key: key, reason: reason})
		},
	})

	// test/dispose.ts:219-223 — set items 1-5
	c.Set(1, 1)
	c.Set(2, 2, SetOptions[int, int]{TTL: Int64(10)}) // test/dispose.ts:220 — { ttl: 10 }
	c.Set(3, 3)
	c.Set(4, 4)
	c.Set(5, 5)

	// test/dispose.ts:224 — t.strictSame(disposes, [])
	assertDisposals(t, disposes, []disposal[int, int]{}, "no disposals after 5 sets")

	// test/dispose.ts:225 — c.set(6, 6) — evicts key 1 (LRU)
	c.Set(6, 6)
	// test/dispose.ts:226 — t.strictSame(disposes, [[1, 1, 'evict']])
	assertDisposals(t, disposes, []disposal[int, int]{
		{value: 1, key: 1, reason: DisposeEvict},
	}, "evict key 1")

	// test/dispose.ts:227-229 — delete 6, 5, 4
	c.Delete(6)
	c.Delete(5)
	c.Delete(4)

	// test/dispose.ts:232-237
	assertDisposals(t, disposes, []disposal[int, int]{
		{value: 1, key: 1, reason: DisposeEvict},
		{value: 6, key: 6, reason: DisposeDelete},
		{value: 5, key: 5, reason: DisposeDelete},
		{value: 4, key: 4, reason: DisposeDelete},
	}, "after deletes")

	// test/dispose.ts:238 — t.clock.advance(20)
	// Key 2 was set with TTL=10 at time 1. After advancing 20ms, time=21.
	// Key 2's TTL expired at time 11, so at time 21 it's stale.
	clock.advance(20)

	// test/dispose.ts:239 — t.equal(c.get(2), undefined)
	_, ok := c.Get(2)
	assertFalse(t, ok, "key 2 should be expired")

	// test/dispose.ts:240-246
	assertDisposals(t, disposes, []disposal[int, int]{
		{value: 1, key: 1, reason: DisposeEvict},
		{value: 6, key: 6, reason: DisposeDelete},
		{value: 5, key: 5, reason: DisposeDelete},
		{value: 4, key: 4, reason: DisposeDelete},
		{value: 2, key: 2, reason: DisposeExpire},
	}, "key 2 expired")

	// test/dispose.ts:247 — t.clock.advance(200)
	// Key 3 was set with default TTL=100 at time 1. After advancing 200ms total,
	// time=221. Key 3's TTL expired at time 101.
	clock.advance(200)

	// test/dispose.ts:248 — t.equal(c.get(3), undefined)
	_, ok = c.Get(3)
	assertFalse(t, ok, "key 3 should be expired")

	// test/dispose.ts:249-256
	assertDisposals(t, disposes, []disposal[int, int]{
		{value: 1, key: 1, reason: DisposeEvict},
		{value: 6, key: 6, reason: DisposeDelete},
		{value: 5, key: 5, reason: DisposeDelete},
		{value: 4, key: 4, reason: DisposeDelete},
		{value: 2, key: 2, reason: DisposeExpire},
		{value: 3, key: 3, reason: DisposeExpire},
	}, "key 3 expired")
}

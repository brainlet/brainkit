package lrucache

// Tests ported from node-lru-cache test/info.ts
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/info.ts
//
// Uses helpers from helpers_test.go: assertEqual, assertTrue

import (
	"testing"
)

// ---------------------------------------------------------------------------
// test/info.ts — "just kv" (line 4)
// ---------------------------------------------------------------------------

func TestInfo_JustKV(t *testing.T) {
	// TS source: test/info.ts lines 4-12
	// t.test('just kv', t => {
	//   const c = new LRUCache<number, number>({ max: 2 })
	//   c.set(1, 10)
	//   c.set(2, 20)
	//   c.set(3, 30)
	//   t.equal(c.info(1), undefined)
	//   t.strictSame(c.info(2), { value: 20 })
	//   t.strictSame(c.info(3), { value: 30 })

	c := New[int, int](Options[int, int]{Max: 2})

	// TS source: line 6 — c.set(1, 10)
	c.Set(1, 10)
	// TS source: line 7 — c.set(2, 20)
	c.Set(2, 20)
	// TS source: line 8 — c.set(3, 30)
	// Adding key 3 should evict key 1 (LRU, max=2)
	c.Set(3, 30)

	// TS source: line 9 — t.equal(c.info(1), undefined)
	// Key 1 was evicted, info should return nil
	info1 := c.Info(1)
	if info1 != nil {
		t.Errorf("info(1) should be nil (evicted), got %+v", info1)
	}

	// TS source: line 10 — t.strictSame(c.info(2), { value: 20 })
	info2 := c.Info(2)
	if info2 == nil {
		t.Fatal("info(2) should not be nil")
	}
	assertEqual(t, info2.Value, 20, "info(2).Value")
	// No TTL, no size tracking → TTL and Size should be zero
	assertEqual(t, info2.TTL, int64(0), "info(2).TTL should be 0 (no TTL)")
	assertEqual(t, info2.Size, 0, "info(2).Size should be 0 (no size tracking)")

	// TS source: line 11 — t.strictSame(c.info(3), { value: 30 })
	info3 := c.Info(3)
	if info3 == nil {
		t.Fatal("info(3) should not be nil")
	}
	assertEqual(t, info3.Value, 30, "info(3).Value")
	assertEqual(t, info3.TTL, int64(0), "info(3).TTL should be 0 (no TTL)")
	assertEqual(t, info3.Size, 0, "info(3).Size should be 0 (no size tracking)")
}

// ---------------------------------------------------------------------------
// test/info.ts — "other info" (line 15)
// ---------------------------------------------------------------------------

func TestInfo_OtherInfo(t *testing.T) {
	// TS source: test/info.ts lines 15-38
	// t.test('other info', t => {
	//   const c = new LRUCache<number, number>({
	//     max: 2,
	//     ttl: 1000,
	//     maxSize: 10000,
	//   })
	//   c.set(1, 10, { size: 100 })
	//   c.set(2, 20, { size: 200 })
	//   c.set(3, 30, { size: 300 })
	//   t.equal(c.info(1), undefined)
	//   t.match(c.info(2), { value: 20, size: 200, ttl: Number, start: Number })
	//   t.match(c.info(3), { value: 30, size: 300, ttl: Number, start: Number })

	c := New[int, int](Options[int, int]{
		Max:     2,
		TTL:     1000,
		MaxSize: 10000,
	})

	// TS source: line 21 — c.set(1, 10, { size: 100 })
	c.Set(1, 10, SetOptions[int, int]{Size: 100})
	// TS source: line 22 — c.set(2, 20, { size: 200 })
	c.Set(2, 20, SetOptions[int, int]{Size: 200})
	// TS source: line 23 — c.set(3, 30, { size: 300 })
	// Adding key 3 should evict key 1 (LRU, max=2)
	c.Set(3, 30, SetOptions[int, int]{Size: 300})

	// TS source: line 24 — t.equal(c.info(1), undefined)
	info1 := c.Info(1)
	if info1 != nil {
		t.Errorf("info(1) should be nil (evicted), got %+v", info1)
	}

	// TS source: lines 25-30
	// t.match(c.info(2), { value: 20, size: 200, ttl: Number, start: Number })
	info2 := c.Info(2)
	if info2 == nil {
		t.Fatal("info(2) should not be nil")
	}
	assertEqual(t, info2.Value, 20, "info(2).Value")
	assertEqual(t, info2.Size, 200, "info(2).Size")
	// TTL should be a positive number (remaining TTL from 1000ms)
	assertTrue(t, info2.TTL > 0, "info(2).TTL should be positive")
	// Start should be a positive number (current timestamp)
	assertTrue(t, info2.Start > 0, "info(2).Start should be positive")

	// TS source: lines 31-36
	// t.match(c.info(3), { value: 30, size: 300, ttl: Number, start: Number })
	info3 := c.Info(3)
	if info3 == nil {
		t.Fatal("info(3) should not be nil")
	}
	assertEqual(t, info3.Value, 30, "info(3).Value")
	assertEqual(t, info3.Size, 300, "info(3).Size")
	assertTrue(t, info3.TTL > 0, "info(3).TTL should be positive")
	assertTrue(t, info3.Start > 0, "info(3).Start should be positive")
}

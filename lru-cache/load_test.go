package lrucache

// Tests ported from node-lru-cache test/load.ts
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/load.ts
//
// Uses helpers from helpers_test.go: assertEqual, assertTrue

import (
	"testing"
)

// ---------------------------------------------------------------------------
// test/load.ts — Dump/Load roundtrip (lines 1-12)
// ---------------------------------------------------------------------------

func TestLoad_DumpLoadRoundtrip(t *testing.T) {
	// TS source: test/load.ts lines 1-12
	//
	// const c = new LRU<number, number>({ max: 5 })
	// for (let i = 0; i < 9; i++) {
	//   c.set(i, i)
	// }
	//
	// const d = new LRU(c)    ← copies options from c
	// d.load(c.dump())
	//
	// t.strictSame(d, c)

	// TS source: line 4 — const c = new LRU<number, number>({ max: 5 })
	c := New[int, int](Options[int, int]{Max: 5})

	// TS source: lines 5-7 — fill with 9 items (0..8), only last 5 survive (max=5)
	for i := 0; i < 9; i++ {
		c.Set(i, i)
	}

	// After setting 0..8 in a max=5 cache, items 0..3 are evicted.
	// Remaining: keys 4,5,6,7,8
	assertEqual(t, c.Size(), 5, "cache should have 5 items after overflow")

	// TS source: line 9 — const d = new LRU(c) — in TS, passing an LRU instance
	// copies its options. In Go, we create a new cache with the same options.
	d := New[int, int](Options[int, int]{Max: 5})

	// TS source: line 10 — d.load(c.dump())
	dump := c.Dump()
	d.Load(dump)

	// TS source: line 12 — t.strictSame(d, c)
	// Verify the two caches have the same entries.
	// We check size and then verify each key/value pair.
	assertEqual(t, d.Size(), c.Size(), "loaded cache should have same size")

	// Verify all entries match
	for i := 4; i <= 8; i++ {
		vc, okC := c.Get(i)
		vd, okD := d.Get(i)
		assertTrue(t, okC, "original cache should have key")
		assertTrue(t, okD, "loaded cache should have key")
		assertEqual(t, vd, vc, "loaded value should match original")
	}

	// Verify evicted items are not present in either cache
	for i := 0; i < 4; i++ {
		_, okC := c.Get(i)
		_, okD := d.Get(i)
		assertFalse(t, okC, "evicted key should not be in original")
		assertFalse(t, okD, "evicted key should not be in loaded")
	}

	// Also compare dump outputs to ensure order and entries match
	dumpC := c.Dump()
	dumpD := d.Dump()
	assertEqual(t, len(dumpD), len(dumpC), "dump lengths should match")
	for i := range dumpC {
		assertEqual(t, dumpD[i].Key, dumpC[i].Key, "dump keys should match")
		assertEqual(t, dumpD[i].Entry.Value, dumpC[i].Entry.Value, "dump values should match")
	}
}

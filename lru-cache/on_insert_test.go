package lrucache

// Tests ported from node-lru-cache test/onInsert.ts
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/onInsert.ts
//
// Test helpers (assertEqual, assertTrue, assertFalse, assertSliceEqual, etc.)
// are defined in helpers_test.go — shared across all test files.

import (
	"testing"
)

// ===========================================================================
// OnInsert tests
// TS source: test/onInsert.ts
// ===========================================================================

// insertRecord captures a single OnInsert callback invocation.
// TS equivalent: the [v, k, r] arrays pushed to the `inserted` array.
type insertRecord[K comparable, V any] struct {
	value  V
	key    K
	reason InsertReason
}

func TestOnInsertAdd(t *testing.T) {
	// TS source: test/onInsert.ts lines 4-20 — t.test('onInsert')
	// Records all insertions and verifies they are all 'add' reason.
	var inserted []insertRecord[int, int]

	c := New[int, int](Options[int, int]{
		Max: 5,
		// TS source: test/onInsert.ts line 7 — onInsert: (v, k, r) => inserted.push([v, k, r])
		OnInsert: func(value int, key int, reason InsertReason) {
			inserted = append(inserted, insertRecord[int, int]{value, key, reason})
		},
	})

	// TS source: test/onInsert.ts lines 10-12
	// for (let i = 0; i < 5; i++) { c.set(i, i) }
	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}

	// TS source: test/onInsert.ts lines 13-19
	// t.strictSame(inserted, [[0,0,'add'], [1,1,'add'], [2,2,'add'], [3,3,'add'], [4,4,'add']])
	expected := []insertRecord[int, int]{
		{0, 0, InsertAdd},
		{1, 1, InsertAdd},
		{2, 2, InsertAdd},
		{3, 3, InsertAdd},
		{4, 4, InsertAdd},
	}

	if len(inserted) != len(expected) {
		t.Fatalf("inserted length mismatch: got %d, want %d", len(inserted), len(expected))
	}
	for i, e := range expected {
		// TS source: test/onInsert.ts line 13 — t.strictSame(inserted, [...])
		assertEqual(t, inserted[i].value, e.value, "inserted value")
		assertEqual(t, inserted[i].key, e.key, "inserted key")
		assertEqual(t, string(inserted[i].reason), string(e.reason), "inserted reason")
	}
}

func TestOnInsertWithReplace(t *testing.T) {
	// TS source: test/onInsert.ts lines 22-38 — t.test('onInsert with replace')
	// Sets key 1 and 2 as 'add', then sets key 1 with a different value => 'replace'.
	var inserted []insertRecord[int, any]

	c := New[int, any](Options[int, any]{
		Max: 5,
		// TS source: test/onInsert.ts line 25 — onInsert: (v, k, r) => inserted.push([v, k, r])
		OnInsert: func(value any, key int, reason InsertReason) {
			inserted = append(inserted, insertRecord[int, any]{value, key, reason})
		},
	})

	// TS source: test/onInsert.ts lines 28-30
	c.Set(1, 1)
	c.Set(2, 2)
	c.Set(1, "one") // different value → replace

	// TS source: test/onInsert.ts lines 32-36
	// t.strictSame(inserted, [[1, 1, 'add'], [2, 2, 'add'], ['one', 1, 'replace']])
	if len(inserted) != 3 {
		t.Fatalf("inserted length mismatch: got %d, want 3", len(inserted))
	}

	// [1, 1, 'add']
	// TS source: test/onInsert.ts line 33
	assertEqual(t, inserted[0].key, 1, "first insert key")
	assertEqual(t, inserted[0].value, 1, "first insert value")
	assertEqual(t, string(inserted[0].reason), string(InsertAdd), "first insert reason")

	// [2, 2, 'add']
	// TS source: test/onInsert.ts line 34
	assertEqual(t, inserted[1].key, 2, "second insert key")
	assertEqual(t, inserted[1].value, 2, "second insert value")
	assertEqual(t, string(inserted[1].reason), string(InsertAdd), "second insert reason")

	// ['one', 1, 'replace']
	// TS source: test/onInsert.ts line 35
	assertEqual(t, inserted[2].key, 1, "third insert key")
	assertEqual(t, inserted[2].value, "one", "third insert value")
	assertEqual(t, string(inserted[2].reason), string(InsertReplace), "third insert reason")
}

func TestOnInsertWithValueUndefined(t *testing.T) {
	// TS source: test/onInsert.ts lines 40-52 — t.test('onInsert with value === undefined')
	//
	// SKIP: In TypeScript, setting a key to `undefined` is a valid operation that
	// behaves differently from setting it to a real value — the TS LRUCache treats
	// `undefined` as a deletion/no-op and does NOT fire onInsert.
	//
	// In Go, there is no concept of `undefined`. The zero value for any type is
	// always a valid value (0 for int, "" for string, nil for pointers, etc.).
	// Setting a key to the zero value is a normal Set operation that WILL fire
	// onInsert.
	//
	// This test cannot be faithfully ported because Go's type system does not
	// have the undefined/null distinction that JavaScript has.
	t.Skip("Go has no undefined concept — setting zero value is a valid insertion, unlike JS undefined which acts as deletion")
}

func TestOnInsertWithUpdate(t *testing.T) {
	// TS source: test/onInsert.ts lines 54-68 — t.test('onInsert with update (same value)')
	// Sets key 1, then sets key 1 again with the same value.
	// First call is 'add', second is 'update' (same value re-set).
	var inserted []insertRecord[int, int]

	c := New[int, int](Options[int, int]{
		Max: 5,
		// TS source: test/onInsert.ts line 57 — onInsert: (v, k, r) => inserted.push([v, k, r])
		OnInsert: func(value int, key int, reason InsertReason) {
			inserted = append(inserted, insertRecord[int, int]{value, key, reason})
		},
	})

	// TS source: test/onInsert.ts lines 60-61
	c.Set(1, 1)
	c.Set(1, 1) // update with the same value

	// TS source: test/onInsert.ts lines 63-66
	// t.strictSame(inserted, [[1, 1, 'add'], [1, 1, 'update']])
	if len(inserted) != 2 {
		t.Fatalf("inserted length mismatch: got %d, want 2", len(inserted))
	}

	// [1, 1, 'add']
	// TS source: test/onInsert.ts line 64
	assertEqual(t, inserted[0].value, 1, "first insert value")
	assertEqual(t, inserted[0].key, 1, "first insert key")
	assertEqual(t, string(inserted[0].reason), string(InsertAdd), "first insert reason")

	// [1, 1, 'update']
	// TS source: test/onInsert.ts line 65
	assertEqual(t, inserted[1].value, 1, "second insert value")
	assertEqual(t, inserted[1].key, 1, "second insert key")
	assertEqual(t, string(inserted[1].reason), string(InsertUpdate), "second insert reason")
}

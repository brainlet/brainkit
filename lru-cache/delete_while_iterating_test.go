package lrucache

// delete_while_iterating_test.go — Faithful 1:1 port of node-lru-cache test/delete-while-iterating.ts.
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/delete-while-iterating.ts
//
// Every test case includes a comment with the original source file and line number.
// Uses test helpers from helpers_test.go.
//
// Adaptation notes:
//   - In TS, keys()/rkeys() are generators that yield lazily, so deleting during
//     iteration modifies the underlying linked list and the generator sees the changes.
//   - In Go, Keys()/RKeys() return snapshot slices, so iterating a snapshot and
//     deleting is safe by design. The test verifies the same end state.
//   - t.beforeEach → setup helper function called at the start of each subtest.

import "testing"

// setupDeleteWhileIterating replicates the t.beforeEach from test/delete-while-iterating.ts lines 4-12.
// Creates a cache with max:5, sets keys 0-4 with values 0-4.
func setupDeleteWhileIterating() *LRUCache[int, int] {
	// test/delete-while-iterating.ts lines 4-12
	c := New[int, int](Options[int, int]{Max: 5})
	c.Set(0, 0)
	c.Set(1, 1)
	c.Set(2, 2)
	c.Set(3, 3)
	c.Set(4, 4)
	return c
}

func TestDeleteWhileIterating_DeleteEvens(t *testing.T) {
	// test/delete-while-iterating.ts lines 14-26: "delete evens"
	c := setupDeleteWhileIterating()

	// test/delete-while-iterating.ts line 16: t.same([...c.keys()], [4, 3, 2, 1, 0])
	assertSliceEqual(t, c.Keys(), []int{4, 3, 2, 1, 0}, "initial keys")

	// test/delete-while-iterating.ts lines 18-22: iterate keys, delete k%2==0
	// In TS this iterates a generator; in Go we iterate the snapshot.
	for _, k := range c.Keys() {
		if k%2 == 0 {
			c.Delete(k)
		}
	}

	// test/delete-while-iterating.ts line 24: t.same([...c.keys()], [3, 1])
	assertSliceEqual(t, c.Keys(), []int{3, 1}, "keys after deleting evens")
}

func TestDeleteWhileIterating_DeleteOdds(t *testing.T) {
	// test/delete-while-iterating.ts lines 28-40: "delete odds"
	c := setupDeleteWhileIterating()

	// test/delete-while-iterating.ts line 30: t.same([...c.keys()], [4, 3, 2, 1, 0])
	assertSliceEqual(t, c.Keys(), []int{4, 3, 2, 1, 0}, "initial keys")

	// test/delete-while-iterating.ts lines 32-36: iterate keys, delete k%2==1
	for _, k := range c.Keys() {
		if k%2 == 1 {
			c.Delete(k)
		}
	}

	// test/delete-while-iterating.ts line 38: t.same([...c.keys()], [4, 2, 0])
	assertSliceEqual(t, c.Keys(), []int{4, 2, 0}, "keys after deleting odds")
}

func TestDeleteWhileIterating_RDeleteEvens(t *testing.T) {
	// test/delete-while-iterating.ts lines 42-54: "rdelete evens"
	c := setupDeleteWhileIterating()

	// test/delete-while-iterating.ts line 44: t.same([...c.keys()], [4, 3, 2, 1, 0])
	assertSliceEqual(t, c.Keys(), []int{4, 3, 2, 1, 0}, "initial keys")

	// test/delete-while-iterating.ts lines 46-50: iterate rkeys, delete k%2==0
	for _, k := range c.RKeys() {
		if k%2 == 0 {
			c.Delete(k)
		}
	}

	// test/delete-while-iterating.ts line 52: t.same([...c.keys()], [3, 1])
	assertSliceEqual(t, c.Keys(), []int{3, 1}, "keys after rdeleting evens")
}

func TestDeleteWhileIterating_RDeleteOdds(t *testing.T) {
	// test/delete-while-iterating.ts lines 56-68: "rdelete odds"
	c := setupDeleteWhileIterating()

	// test/delete-while-iterating.ts line 58: t.same([...c.keys()], [4, 3, 2, 1, 0])
	assertSliceEqual(t, c.Keys(), []int{4, 3, 2, 1, 0}, "initial keys")

	// test/delete-while-iterating.ts lines 60-64: iterate rkeys, delete k%2==1
	for _, k := range c.RKeys() {
		if k%2 == 1 {
			c.Delete(k)
		}
	}

	// test/delete-while-iterating.ts line 66: t.same([...c.keys()], [4, 2, 0])
	assertSliceEqual(t, c.Keys(), []int{4, 2, 0}, "keys after rdeleting odds")
}

func TestDeleteWhileIterating_DeleteTwoOfThem(t *testing.T) {
	// test/delete-while-iterating.ts lines 70-84: "delete two of them"
	c := setupDeleteWhileIterating()

	// test/delete-while-iterating.ts line 72: t.same([...c.keys()], [4, 3, 2, 1, 0])
	assertSliceEqual(t, c.Keys(), []int{4, 3, 2, 1, 0}, "initial keys")

	// test/delete-while-iterating.ts lines 73-81: iterate keys, delete pairs
	for _, k := range c.Keys() {
		if k == 3 {
			c.Delete(3)
			c.Delete(4)
		} else if k == 1 {
			c.Delete(1)
			c.Delete(0)
		}
	}

	// test/delete-while-iterating.ts line 82: t.same([...c.keys()], [2])
	assertSliceEqual(t, c.Keys(), []int{2}, "keys after deleting two pairs")
}

func TestDeleteWhileIterating_RDeleteTwoOfThem(t *testing.T) {
	// test/delete-while-iterating.ts lines 86-100: "rdelete two of them"
	c := setupDeleteWhileIterating()

	// test/delete-while-iterating.ts line 88: t.same([...c.keys()], [4, 3, 2, 1, 0])
	assertSliceEqual(t, c.Keys(), []int{4, 3, 2, 1, 0}, "initial keys")

	// test/delete-while-iterating.ts lines 89-97: iterate rkeys, delete pairs
	for _, k := range c.RKeys() {
		if k == 3 {
			c.Delete(3)
			c.Delete(4)
		} else if k == 1 {
			c.Delete(1)
			c.Delete(0)
		}
	}

	// test/delete-while-iterating.ts line 98: t.same([...c.keys()], [2])
	assertSliceEqual(t, c.Keys(), []int{2}, "keys after rdeleting two pairs")
}

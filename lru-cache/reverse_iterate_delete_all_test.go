package lrucache

// reverse_iterate_delete_all_test.go — Faithful 1:1 port of node-lru-cache
// test/reverse-iterate-delete-all.ts.
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/reverse-iterate-delete-all.ts
//
// Every test case includes a comment with the original source file and line number.
// Uses test helpers from helpers_test.go.
//
// Context: https://github.com/isaacs/node-lru-cache/issues/278
// This test verifies that reverse-iterating and deleting all entries leaves
// the cache empty. Uses maxSize with sizeCalculation instead of max.

import "testing"

func TestReverseIterateDeleteAll(t *testing.T) {
	// test/reverse-iterate-delete-all.ts lines 1-13
	// https://github.com/isaacs/node-lru-cache/issues/278

	// test/reverse-iterate-delete-all.ts lines 4-7:
	// const lru = new LRU<string, string>({ maxSize: 2, sizeCalculation: () => 1 })
	lru := New[string, string](Options[string, string]{
		MaxSize: 2,
		SizeCalculation: func(value string, key string) int {
			// test/reverse-iterate-delete-all.ts line 6: sizeCalculation: () => 1
			return 1
		},
	})

	// test/reverse-iterate-delete-all.ts lines 8-9:
	// lru.set('x', 'x')
	// lru.set('y', 'y')
	lru.Set("x", "x")
	lru.Set("y", "y")

	// test/reverse-iterate-delete-all.ts lines 10-12:
	// for (const key of lru.rkeys()) { lru.delete(key) }
	// In Go, RKeys() returns a snapshot, so iterating and deleting is safe.
	for _, key := range lru.RKeys() {
		lru.Delete(key)
	}

	// test/reverse-iterate-delete-all.ts line 13: t.equal(lru.size, 0)
	assertEqual(t, lru.Size(), 0, "size after deleting all via rkeys")
}

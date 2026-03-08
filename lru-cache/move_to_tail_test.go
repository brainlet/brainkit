package lrucache

// move_to_tail_test.go — Faithful 1:1 port of node-lru-cache test/move-to-tail.ts.
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/test/move-to-tail.ts
//
// Every test case includes a comment with the original source file and line number.
// Uses test helpers from helpers_test.go: exposeMoveToTail, exposeHead, exposeTail,
// exposeNext, exposePrev.
//
// Adaptation notes:
//   - t.matchSnapshot → replaced with direct list integrity verification.
//     The TS test uses matchSnapshot to record the linked list state. In Go,
//     we verify the list integrity invariants directly (n[p[i]]==i, p[n[i]]==i).
//   - expose(c) → direct access to internals via expose* helpers (same package).
//   - The "e" and "snap" helpers in TS build a readable snapshot object.
//     We skip that representation and verify the invariants directly.

import (
	"fmt"
	"testing"
)

// verifyListIntegrity checks that the doubly-linked list is consistent.
// For every index i in [0, max):
//   - if i is not head: next[prev[i]] == i
//   - if i is not tail: prev[next[i]] == i
// This is the direct equivalent of the TS "integrity" helper at lines 27-39.
func verifyListIntegrity(t *testing.T, c *LRUCache[int, int], msg string) {
	t.Helper()
	head := exposeHead(c)
	tail := exposeTail(c)
	next := exposeNext(c)
	prev := exposePrev(c)

	for i := 0; i < c.Max(); i++ {
		// test/move-to-tail.ts line 31: if (i !== exp.head)
		if i != head {
			// test/move-to-tail.ts line 32: t.equal(exp.next[exp.prev[i]], i, 'n[p[i]] === i')
			if next[prev[i]] != i {
				t.Errorf("%s: n[p[%d]] = %d, want %d (head=%d, tail=%d, prev=%v, next=%v)",
					msg, i, next[prev[i]], i, head, tail, prev, next)
			}
		}
		// test/move-to-tail.ts line 34: if (i !== exp.tail)
		if i != tail {
			// test/move-to-tail.ts line 35: t.equal(exp.prev[exp.next[i]], i, 'p[n[i]] === i')
			if prev[next[i]] != i {
				t.Errorf("%s: p[n[%d]] = %d, want %d (head=%d, tail=%d, prev=%v, next=%v)",
					msg, i, prev[next[i]], i, head, tail, prev, next)
			}
		}
	}
}

func TestMoveToTail_ListIntegrity(t *testing.T) {
	// test/move-to-tail.ts lines 5-55: main test block

	// test/move-to-tail.ts lines 5-6:
	// const c = new LRU({ max: 5 })
	// const exp = expose(c)
	c := New[int, int](Options[int, int]{Max: 5})

	// test/move-to-tail.ts lines 41-43: fill with 0-4
	// for (let i = 0; i < 5; i++) { c.set(i, i) }
	for i := 0; i < 5; i++ {
		c.Set(i, i)
	}

	// test/move-to-tail.ts line 45: t.matchSnapshot(snap(), 'list after initial fill')
	// SKIPPED: matchSnapshot — verify integrity directly.
	// After filling 0,1,2,3,4 in order:
	//   head=0 (LRU), tail=4 (MRU)
	//   list order (head→tail): 0→1→2→3→4

	// test/move-to-tail.ts line 46: integrity('after initial fill')
	verifyListIntegrity(t, c, "after initial fill")

	// Verify expected head/tail after initial fill
	assertEqual(t, exposeHead(c), 0, "head after initial fill")
	assertEqual(t, exposeTail(c), 4, "tail after initial fill")

	// test/move-to-tail.ts line 47: exp.moveToTail(2)
	exposeMoveToTail(c, 2)

	// test/move-to-tail.ts line 48: t.matchSnapshot(snap(), 'list after moveToTail 2')
	// SKIPPED: matchSnapshot — verify integrity and expected head/tail.
	// After moveToTail(2): list order should be 0→1→3→4→2
	//   head=0, tail=2

	// test/move-to-tail.ts line 49: integrity('after moveToTail 2')
	verifyListIntegrity(t, c, "after moveToTail 2")

	// Verify head stays at 0, tail is now 2
	assertEqual(t, exposeHead(c), 0, "head after moveToTail 2")
	assertEqual(t, exposeTail(c), 2, "tail after moveToTail 2")

	// Verify the full order by walking head→tail
	order := walkHeadToTail(c)
	assertSliceEqual(t, order, []int{0, 1, 3, 4, 2}, "order after moveToTail 2")

	// test/move-to-tail.ts line 50: exp.moveToTail(4)
	exposeMoveToTail(c, 4)

	// test/move-to-tail.ts line 51: t.matchSnapshot(snap(), 'list after moveToTail 4')
	// SKIPPED: matchSnapshot — verify integrity and expected head/tail.
	// After moveToTail(4): list order should be 0→1→3→2→4
	//   head=0, tail=4

	// test/move-to-tail.ts line 52: integrity('after moveToTail 4')
	verifyListIntegrity(t, c, "after moveToTail 4")

	// Verify head stays at 0, tail is now 4
	assertEqual(t, exposeHead(c), 0, "head after moveToTail 4")
	assertEqual(t, exposeTail(c), 4, "tail after moveToTail 4")

	// Verify the full order by walking head→tail
	order2 := walkHeadToTail(c)
	assertSliceEqual(t, order2, []int{0, 1, 3, 2, 4}, "order after moveToTail 4")
}

// walkHeadToTail returns the list of indices from head to tail by following next pointers.
// This is a debugging helper that walks the internal linked list.
func walkHeadToTail(c *LRUCache[int, int]) []int {
	head := exposeHead(c)
	tail := exposeTail(c)
	next := exposeNext(c)

	var result []int
	for i := head; ; {
		result = append(result, i)
		if i == tail {
			break
		}
		i = next[i]
		// Safety guard against infinite loops
		if len(result) > c.Max()+1 {
			panic(fmt.Sprintf("walkHeadToTail: infinite loop detected (visited %d nodes for max %d)", len(result), c.Max()))
		}
	}
	return result
}

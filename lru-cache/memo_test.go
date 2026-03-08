package lrucache

import "testing"

func TestMemoBasic(t *testing.T) {
	memoCalls := []int{}
	c := New[int, int](Options[int, int]{
		Max: 5,
		MemoMethod: func(key int, staleValue *int, _ MemoizerOptions[int, int]) int {
			if staleValue != nil {
				t.Fatalf("unexpected stale value: %v", *staleValue)
			}
			memoCalls = append(memoCalls, key)
			return key * key
		},
	})

	_, ok := c.Get(2)
	assertFalse(t, ok, "get should miss before memo")

	four := c.Memo(2)
	fourAgain := c.Memo(2)
	assertEqual(t, four, 4, "first memo value")
	assertEqual(t, fourAgain, 4, "second memo value")

	v, ok := c.Get(2)
	assertTrue(t, ok, "memoized value should be cached")
	assertEqual(t, v, 4, "memoized cached value")
	assertSliceEqual(t, memoCalls, []int{2}, "memoMethod should only run once")
}

func TestMemoWithContext(t *testing.T) {
	memoCalls := [][3]any{}
	c := New[int, int](Options[int, int]{
		Max: 5,
		MemoMethod: func(key int, staleValue *int, opts MemoizerOptions[int, int]) int {
			contextFlag, _ := opts.Context.(bool)
			var stale any
			if staleValue != nil {
				stale = *staleValue
			}
			memoCalls = append(memoCalls, [3]any{key, stale, contextFlag})
			if contextFlag {
				return key
			}
			if !opts.Options.NoDeleteOnStaleGet {
				t.Fatal("expected noDeleteOnStaleGet to be forwarded to memoMethod")
			}
			return key * key
		},
	})

	assertEqual(t, c.Memo(1, MemoOptions[int, int]{Context: true}), 1, "memo with true context")
	assertEqual(t, c.Memo(1, MemoOptions[int, int]{Context: true}), 1, "memo hit with true context")
	assertEqual(t, c.Memo(1, MemoOptions[int, int]{Context: false}), 1, "cached value should win regardless of context")
	assertEqual(t, c.Memo(2, MemoOptions[int, int]{
		Context:            false,
		NoDeleteOnStaleGet: Bool(true),
	}), 4, "memo with false context")
	assertEqual(t, c.Memo(2, MemoOptions[int, int]{Context: true}), 4, "memo hit with true context should keep cached value")

	if len(memoCalls) != 2 {
		t.Fatalf("expected 2 memo calls, got %d", len(memoCalls))
	}
}

func TestMemoWithoutMemoMethodPanics(t *testing.T) {
	c := New[int, int](Options[int, int]{Max: 1})
	assertPanics(t, func() {
		c.Memo(3)
	}, "memo without MemoMethod should panic")
}

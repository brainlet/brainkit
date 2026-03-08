// Ported from: packages/ai/src/util/async-iterable-stream.test.ts
package util

import (
	"context"
	"testing"
)

func TestStream_ReadAllChunks(t *testing.T) {
	ctx := context.Background()
	s := StreamFromSlice(ctx, []string{"chunk1", "chunk2", "chunk3"})

	result, err := CollectStream(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 || result[0] != "chunk1" || result[1] != "chunk2" || result[2] != "chunk3" {
		t.Fatalf("expected [chunk1 chunk2 chunk3], got %v", result)
	}
}

func TestStream_EmptyStream(t *testing.T) {
	ctx := context.Background()
	s := StreamFromSlice[string](ctx, nil)

	result, err := CollectStream(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
}

func TestStream_EarlyCancel(t *testing.T) {
	ctx := context.Background()
	s := NewStream[string](ctx, func(w *StreamWriter[string]) {
		w.Enqueue("chunk1")
		w.Enqueue("chunk2")
		w.Enqueue("chunk3")
	})

	var collected []string
	for val, err := range s.Iter() {
		if err != nil {
			break
		}
		collected = append(collected, val)
		if val == "chunk2" {
			break // early exit
		}
	}

	if len(collected) != 2 || collected[0] != "chunk1" || collected[1] != "chunk2" {
		t.Fatalf("expected [chunk1 chunk2], got %v", collected)
	}
}

func TestStream_ConsumeStream(t *testing.T) {
	ctx := context.Background()
	s := StreamFromSlice(ctx, []int{1, 2, 3})

	var lastErr error
	ConsumeStream(s, func(err error) {
		lastErr = err
	})
	if lastErr != nil {
		t.Fatalf("unexpected error: %v", lastErr)
	}
}

func TestStream_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	expectedErr := context.Canceled

	s := NewStream[string](ctx, func(w *StreamWriter[string]) {
		w.Enqueue("chunk1")
		w.Error(expectedErr)
	})

	var collected []string
	var gotErr error
	for val, err := range s.Iter() {
		if err != nil {
			gotErr = err
			break
		}
		collected = append(collected, val)
	}

	if len(collected) != 1 || collected[0] != "chunk1" {
		t.Fatalf("expected [chunk1], got %v", collected)
	}
	if gotErr == nil {
		t.Fatal("expected error")
	}
}

// Ported from: packages/ai/src/util/create-stitchable-stream.test.ts
package util

import (
	"context"
	"reflect"
	"testing"
)

func TestStitchableStream_ImmediateClose(t *testing.T) {
	ctx := context.Background()
	ss := CreateStitchableStream[int](ctx)

	ss.Close()

	result, err := CollectStream(ss.Stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
}

func TestStitchableStream_SingleInnerStream(t *testing.T) {
	ctx := context.Background()
	ss := CreateStitchableStream[int](ctx)

	ss.AddStream(StreamFromSlice(ctx, []int{1, 2, 3}))
	ss.Close()

	result, err := CollectStream(ss.Stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestStitchableStream_TwoInnerStreams(t *testing.T) {
	ctx := context.Background()
	ss := CreateStitchableStream[int](ctx)

	ss.AddStream(StreamFromSlice(ctx, []int{1, 2, 3}))
	ss.AddStream(StreamFromSlice(ctx, []int{4, 5, 6}))
	ss.Close()

	result, err := CollectStream(ss.Stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3, 4, 5, 6}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestStitchableStream_ThreeInnerStreams(t *testing.T) {
	ctx := context.Background()
	ss := CreateStitchableStream[int](ctx)

	ss.AddStream(StreamFromSlice(ctx, []int{1, 2, 3}))
	ss.AddStream(StreamFromSlice(ctx, []int{4, 5, 6}))
	ss.AddStream(StreamFromSlice(ctx, []int{7, 8, 9}))
	ss.Close()

	result, err := CollectStream(ss.Stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestStitchableStream_EmptyInnerStreams(t *testing.T) {
	ctx := context.Background()
	ss := CreateStitchableStream[int](ctx)

	ss.AddStream(StreamFromSlice[int](ctx, nil))
	ss.AddStream(StreamFromSlice(ctx, []int{1, 2}))
	ss.AddStream(StreamFromSlice[int](ctx, nil))
	ss.AddStream(StreamFromSlice(ctx, []int{3, 4}))
	ss.Close()

	result, err := CollectStream(ss.Stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{1, 2, 3, 4}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestStitchableStream_AddAfterClose_Panics(t *testing.T) {
	ctx := context.Background()
	ss := CreateStitchableStream[int](ctx)

	ss.Close()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
	}()
	ss.AddStream(StreamFromSlice(ctx, []int{1, 2}))
}

func TestStitchableStream_AddAfterTerminate_Panics(t *testing.T) {
	ctx := context.Background()
	ss := CreateStitchableStream[int](ctx)

	ss.Terminate()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
	}()
	ss.AddStream(StreamFromSlice(ctx, []int{1, 2}))
}

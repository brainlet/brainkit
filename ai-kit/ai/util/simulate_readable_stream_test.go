// Ported from: packages/ai/src/util/simulate-readable-stream.test.ts
package util

import (
	"context"
	"reflect"
	"testing"
)

func TestSimulateReadableStream_BasicValues(t *testing.T) {
	ctx := context.Background()
	values := []string{"a", "b", "c"}
	s := SimulateReadableStream(ctx, SimulateReadableStreamOptions[string]{
		Chunks:          values,
		InitialDelaySet: true,
		InitialDelayInMs: nil,
		ChunkDelaySet:   true,
		ChunkDelayInMs:  nil,
	})

	result, err := CollectStream(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result, values) {
		t.Fatalf("expected %v, got %v", values, result)
	}
}

func TestSimulateReadableStream_DelayTracking(t *testing.T) {
	ctx := context.Background()
	var delayValues []*int

	mockDelay := func(ms *int) {
		delayValues = append(delayValues, ms)
	}

	s := SimulateReadableStream(ctx, SimulateReadableStreamOptions[int]{
		Chunks:           []int{1, 2, 3},
		InitialDelayInMs: IntPtr(500),
		ChunkDelayInMs:   IntPtr(100),
		DelayFunc:        mockDelay,
	})

	_, err := CollectStream(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(delayValues) != 3 {
		t.Fatalf("expected 3 delays, got %d", len(delayValues))
	}
	if *delayValues[0] != 500 {
		t.Fatalf("expected initial delay 500, got %d", *delayValues[0])
	}
	if *delayValues[1] != 100 || *delayValues[2] != 100 {
		t.Fatalf("expected chunk delays 100, got %v", delayValues)
	}
}

func TestSimulateReadableStream_EmptyValues(t *testing.T) {
	ctx := context.Background()
	s := SimulateReadableStream(ctx, SimulateReadableStreamOptions[int]{
		Chunks:          []int{},
		InitialDelaySet: true,
		InitialDelayInMs: nil,
		ChunkDelaySet:   true,
		ChunkDelayInMs:  nil,
	})

	result, err := CollectStream(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
}

func TestSimulateReadableStream_NullDelays(t *testing.T) {
	ctx := context.Background()
	var delayValues []*int

	mockDelay := func(ms *int) {
		delayValues = append(delayValues, ms)
	}

	s := SimulateReadableStream(ctx, SimulateReadableStreamOptions[int]{
		Chunks:           []int{1, 2, 3},
		InitialDelayInMs: nil,
		InitialDelaySet:  true,
		ChunkDelayInMs:   nil,
		ChunkDelaySet:    true,
		DelayFunc:        mockDelay,
	})

	_, err := CollectStream(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(delayValues) != 3 {
		t.Fatalf("expected 3 delays, got %d", len(delayValues))
	}
	for i, v := range delayValues {
		if v != nil {
			t.Fatalf("expected nil delay at index %d, got %d", i, *v)
		}
	}
}

func TestSimulateReadableStream_NullInitialOnlyDelay(t *testing.T) {
	ctx := context.Background()
	var delayValues []*int

	mockDelay := func(ms *int) {
		delayValues = append(delayValues, ms)
	}

	s := SimulateReadableStream(ctx, SimulateReadableStreamOptions[int]{
		Chunks:           []int{1, 2, 3},
		InitialDelayInMs: nil,
		InitialDelaySet:  true,
		ChunkDelayInMs:   IntPtr(100),
		DelayFunc:        mockDelay,
	})

	_, err := CollectStream(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(delayValues) != 3 {
		t.Fatalf("expected 3 delays, got %d", len(delayValues))
	}
	if delayValues[0] != nil {
		t.Fatalf("expected nil initial delay, got %d", *delayValues[0])
	}
	if *delayValues[1] != 100 || *delayValues[2] != 100 {
		t.Fatal("expected chunk delays of 100")
	}
}

func TestSimulateReadableStream_NullChunkOnlyDelay(t *testing.T) {
	ctx := context.Background()
	var delayValues []*int

	mockDelay := func(ms *int) {
		delayValues = append(delayValues, ms)
	}

	s := SimulateReadableStream(ctx, SimulateReadableStreamOptions[int]{
		Chunks:           []int{1, 2, 3},
		InitialDelayInMs: IntPtr(500),
		ChunkDelayInMs:   nil,
		ChunkDelaySet:    true,
		DelayFunc:        mockDelay,
	})

	_, err := CollectStream(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(delayValues) != 3 {
		t.Fatalf("expected 3 delays, got %d", len(delayValues))
	}
	if *delayValues[0] != 500 {
		t.Fatalf("expected initial delay 500, got %d", *delayValues[0])
	}
	if delayValues[1] != nil || delayValues[2] != nil {
		t.Fatal("expected nil chunk delays")
	}
}

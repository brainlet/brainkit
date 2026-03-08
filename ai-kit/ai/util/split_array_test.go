// Ported from: packages/ai/src/util/split-array.test.ts
package util

import (
	"reflect"
	"testing"
)

func TestSplitArray_BasicChunks(t *testing.T) {
	array := []int{1, 2, 3, 4, 5}
	result, err := SplitArray(array, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := [][]int{{1, 2}, {3, 4}, {5}}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestSplitArray_EmptyArray(t *testing.T) {
	var array []int
	result, err := SplitArray(array, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
}

func TestSplitArray_ChunkSizeGreaterThanLength(t *testing.T) {
	array := []int{1, 2, 3}
	result, err := SplitArray(array, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := [][]int{{1, 2, 3}}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestSplitArray_ChunkSizeEqualToLength(t *testing.T) {
	array := []int{1, 2, 3}
	result, err := SplitArray(array, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := [][]int{{1, 2, 3}}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestSplitArray_ChunkSizeOfOne(t *testing.T) {
	array := []int{1, 2, 3}
	result, err := SplitArray(array, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := [][]int{{1}, {2}, {3}}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestSplitArray_ChunkSizeZero(t *testing.T) {
	array := []int{1, 2, 3}
	_, err := SplitArray(array, 0)
	if err == nil {
		t.Fatal("expected error for chunk size 0")
	}
}

func TestSplitArray_NegativeChunkSize(t *testing.T) {
	array := []int{1, 2, 3}
	_, err := SplitArray(array, -1)
	if err == nil {
		t.Fatal("expected error for negative chunk size")
	}
}

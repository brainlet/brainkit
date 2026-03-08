// Ported from: packages/ai/src/util/as-array.ts (no separate test file in TS)
package util

import (
	"testing"
)

func TestAsArray_Nil(t *testing.T) {
	result := AsArray[string](nil)
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %v", result)
	}
}

func TestAsArray_Slice(t *testing.T) {
	input := []string{"a", "b"}
	result := AsArray(input)
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Fatalf("expected [a b], got %v", result)
	}
}

func TestAsArraySingle(t *testing.T) {
	result := AsArraySingle("hello")
	if len(result) != 1 || result[0] != "hello" {
		t.Fatalf("expected [hello], got %v", result)
	}
}

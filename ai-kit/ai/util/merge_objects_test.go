// Ported from: packages/ai/src/util/merge-objects.test.ts
package util

import (
	"reflect"
	"testing"
)

func TestMergeObjects_FlatObjects(t *testing.T) {
	base := map[string]interface{}{"a": 1, "b": 2}
	overrides := map[string]interface{}{"b": 3, "c": 4}
	result := MergeObjects(base, overrides)

	expected := map[string]interface{}{"a": 1, "b": 3, "c": 4}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	// Original objects should not be modified
	if base["b"] != 2 {
		t.Fatal("base was mutated")
	}
}

func TestMergeObjects_DeepMerge(t *testing.T) {
	base := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{"c": 2, "d": 3},
	}
	overrides := map[string]interface{}{
		"b": map[string]interface{}{"c": 4, "e": 5},
	}
	result := MergeObjects(base, overrides)

	expected := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{"c": 4, "d": 3, "e": 5},
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestMergeObjects_ArraysReplaced(t *testing.T) {
	base := map[string]interface{}{
		"a": []interface{}{1, 2, 3},
		"b": 2,
	}
	overrides := map[string]interface{}{
		"a": []interface{}{4, 5},
	}
	result := MergeObjects(base, overrides)

	expected := map[string]interface{}{
		"a": []interface{}{4, 5},
		"b": 2,
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestMergeObjects_NullValues(t *testing.T) {
	base := map[string]interface{}{"a": 1, "b": nil}
	overrides := map[string]interface{}{"a": nil, "b": 2}
	result := MergeObjects(base, overrides)

	expected := map[string]interface{}{"a": nil, "b": 2}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestMergeObjects_ComplexNested(t *testing.T) {
	base := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{
			"c": []interface{}{1, 2, 3},
			"d": map[string]interface{}{"e": 4, "f": 5},
		},
	}
	overrides := map[string]interface{}{
		"b": map[string]interface{}{
			"c": []interface{}{4, 5},
			"d": map[string]interface{}{"f": 6, "g": 7},
		},
		"h": 8,
	}
	result := MergeObjects(base, overrides)

	expected := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{
			"c": []interface{}{4, 5},
			"d": map[string]interface{}{"e": 4, "f": 6, "g": 7},
		},
		"h": 8,
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestMergeObjects_EmptyObjects(t *testing.T) {
	base := map[string]interface{}{}
	overrides := map[string]interface{}{"a": 1}
	result := MergeObjects(base, overrides)
	expected := map[string]interface{}{"a": 1}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	base2 := map[string]interface{}{"a": 1}
	overrides2 := map[string]interface{}{}
	result2 := MergeObjects(base2, overrides2)
	expected2 := map[string]interface{}{"a": 1}
	if !reflect.DeepEqual(result2, expected2) {
		t.Fatalf("expected %v, got %v", expected2, result2)
	}
}

func TestMergeObjects_NilInputs(t *testing.T) {
	if MergeObjects(nil, nil) != nil {
		t.Fatal("expected nil for both nil inputs")
	}

	result1 := MergeObjects(map[string]interface{}{"a": 1}, nil)
	if !reflect.DeepEqual(result1, map[string]interface{}{"a": 1}) {
		t.Fatalf("expected {a:1}, got %v", result1)
	}

	result2 := MergeObjects(nil, map[string]interface{}{"b": 2})
	if !reflect.DeepEqual(result2, map[string]interface{}{"b": 2}) {
		t.Fatalf("expected {b:2}, got %v", result2)
	}
}

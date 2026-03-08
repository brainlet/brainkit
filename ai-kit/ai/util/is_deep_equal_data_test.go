// Ported from: packages/ai/src/util/is-deep-equal-data.test.ts
package util

import (
	"testing"
	"time"
)

func TestIsDeepEqualData_Primitives(t *testing.T) {
	if !IsDeepEqualData(1, 1) {
		t.Fatal("expected equal for same int")
	}
	if IsDeepEqualData(1, 2) {
		t.Fatal("expected not equal for different ints")
	}
}

func TestIsDeepEqualData_DifferentTypes(t *testing.T) {
	obj := map[string]interface{}{"a": 1}
	if IsDeepEqualData(obj, 1) {
		t.Fatal("expected not equal for map vs int")
	}
}

func TestIsDeepEqualData_NilVsObject(t *testing.T) {
	obj := map[string]interface{}{"a": 1}
	if IsDeepEqualData(obj, nil) {
		t.Fatal("expected not equal for map vs nil")
	}
}

func TestIsDeepEqualData_EqualObjects(t *testing.T) {
	obj1 := map[string]interface{}{"a": 1, "b": 2}
	obj2 := map[string]interface{}{"a": 1, "b": 2}
	if !IsDeepEqualData(obj1, obj2) {
		t.Fatal("expected equal objects")
	}
}

func TestIsDeepEqualData_DifferentValues(t *testing.T) {
	obj1 := map[string]interface{}{"a": 1, "b": 2}
	obj2 := map[string]interface{}{"a": 1, "b": 3}
	if IsDeepEqualData(obj1, obj2) {
		t.Fatal("expected not equal")
	}
}

func TestIsDeepEqualData_DifferentKeyCount(t *testing.T) {
	obj1 := map[string]interface{}{"a": 1, "b": 2}
	obj2 := map[string]interface{}{"a": 1, "b": 2, "c": 3}
	if IsDeepEqualData(obj1, obj2) {
		t.Fatal("expected not equal")
	}
}

func TestIsDeepEqualData_NestedObjects(t *testing.T) {
	obj1 := map[string]interface{}{"a": map[string]interface{}{"c": 1}, "b": 2}
	obj2 := map[string]interface{}{"a": map[string]interface{}{"c": 1}, "b": 2}
	if !IsDeepEqualData(obj1, obj2) {
		t.Fatal("expected equal nested objects")
	}
}

func TestIsDeepEqualData_NestedObjectsInequality(t *testing.T) {
	obj1 := map[string]interface{}{"a": map[string]interface{}{"c": 1}, "b": 2}
	obj2 := map[string]interface{}{"a": map[string]interface{}{"c": 2}, "b": 2}
	if IsDeepEqualData(obj1, obj2) {
		t.Fatal("expected not equal")
	}
}

func TestIsDeepEqualData_Arrays(t *testing.T) {
	arr1 := []interface{}{1, 2, 3}
	arr2 := []interface{}{1, 2, 3}
	if !IsDeepEqualData(arr1, arr2) {
		t.Fatal("expected equal arrays")
	}

	arr3 := []interface{}{1, 2, 3}
	arr4 := []interface{}{1, 2, 4}
	if IsDeepEqualData(arr3, arr4) {
		t.Fatal("expected not equal arrays")
	}
}

func TestIsDeepEqualData_NilComparison(t *testing.T) {
	obj := map[string]interface{}{"a": 1}
	if IsDeepEqualData(obj, nil) {
		t.Fatal("expected not equal")
	}
}

func TestIsDeepEqualData_DateObjects(t *testing.T) {
	date1 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	date3 := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	if !IsDeepEqualData(date1, date2) {
		t.Fatal("expected equal dates")
	}
	if IsDeepEqualData(date1, date3) {
		t.Fatal("expected not equal dates")
	}
}

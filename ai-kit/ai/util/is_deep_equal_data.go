// Ported from: packages/ai/src/util/is-deep-equal-data.ts
package util

import (
	"reflect"
	"time"
)

// IsDeepEqualData performs a deep-equal comparison of two parsed JSON objects.
// It handles primitives, maps, slices, time.Time, and nested structures.
func IsDeepEqualData(obj1, obj2 interface{}) bool {
	// Check if both are nil
	if obj1 == nil && obj2 == nil {
		return true
	}

	// Check if either is nil
	if obj1 == nil || obj2 == nil {
		return false
	}

	v1 := reflect.ValueOf(obj1)
	v2 := reflect.ValueOf(obj2)

	// If both are non-object types (primitives)
	if v1.Kind() != reflect.Map && v1.Kind() != reflect.Slice && v1.Kind() != reflect.Struct &&
		v2.Kind() != reflect.Map && v2.Kind() != reflect.Slice && v2.Kind() != reflect.Struct {
		return reflect.DeepEqual(obj1, obj2)
	}

	// If they are not the same type, they are not equal
	if v1.Type() != v2.Type() {
		return false
	}

	// Special handling for time.Time (equivalent to Date objects)
	if t1, ok := obj1.(time.Time); ok {
		if t2, ok := obj2.(time.Time); ok {
			return t1.Equal(t2)
		}
		return false
	}

	// Handle slices
	if v1.Kind() == reflect.Slice {
		if v1.Len() != v2.Len() {
			return false
		}
		for i := 0; i < v1.Len(); i++ {
			if !IsDeepEqualData(v1.Index(i).Interface(), v2.Index(i).Interface()) {
				return false
			}
		}
		return true
	}

	// Handle maps
	if v1.Kind() == reflect.Map {
		keys1 := v1.MapKeys()
		keys2 := v2.MapKeys()
		if len(keys1) != len(keys2) {
			return false
		}
		for _, key := range keys1 {
			val2 := v2.MapIndex(key)
			if !val2.IsValid() {
				return false
			}
			if !IsDeepEqualData(v1.MapIndex(key).Interface(), val2.Interface()) {
				return false
			}
		}
		return true
	}

	// Fall back to reflect.DeepEqual for structs and other types
	return reflect.DeepEqual(obj1, obj2)
}

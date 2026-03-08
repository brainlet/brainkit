// Ported from: packages/ai/src/util/merge-objects.ts
package util

import (
	"reflect"
	"time"
)

// MergeObjects deeply merges two maps together.
//   - Properties from the overrides map override those in the base map with the same key.
//   - For nested maps, the merge is performed recursively (deep merge).
//   - Slices are replaced, not merged.
//   - Primitive values are replaced.
//   - If both base and overrides are nil, returns nil.
//   - If one of base or overrides is nil, returns the other.
func MergeObjects(base, overrides map[string]interface{}) map[string]interface{} {
	if base == nil && overrides == nil {
		return nil
	}

	if base == nil {
		return overrides
	}

	if overrides == nil {
		return base
	}

	// Create a new map to avoid mutating the inputs.
	result := make(map[string]interface{})
	for k, v := range base {
		result[k] = v
	}

	for key, overridesValue := range overrides {
		// Skip nil overrides values — but we need to distinguish between
		// "key not present" and "key present with nil value".
		// In Go maps, a nil value is a valid entry, so we include it.

		baseValue, baseHasKey := result[key]

		isSourceObject := isPlainObject(overridesValue)
		isTargetObject := baseHasKey && isPlainObject(baseValue)

		if isSourceObject && isTargetObject {
			result[key] = MergeObjects(
				baseValue.(map[string]interface{}),
				overridesValue.(map[string]interface{}),
			)
		} else {
			result[key] = overridesValue
		}
	}

	return result
}

// isPlainObject checks if a value is a plain map[string]interface{} (not a slice,
// not a time.Time, not nil).
func isPlainObject(v interface{}) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return false
	}
	// Exclude time.Time (corresponds to Date exclusion in TS)
	if _, ok := v.(time.Time); ok {
		return false
	}
	// Must be map[string]interface{}
	_, ok := v.(map[string]interface{})
	return ok
}

// Ported from: packages/ai/src/util/as-array.ts
package util

// AsArray converts a value into a slice. If the value is nil, returns an empty slice.
// If the value is already a slice, it is returned as-is. Otherwise, a single-element
// slice is returned.
func AsArray[T any](value []T) []T {
	if value == nil {
		return []T{}
	}
	return value
}

// AsArraySingle wraps a single value into a single-element slice.
func AsArraySingle[T any](value T) []T {
	return []T{value}
}

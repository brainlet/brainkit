// Ported from: packages/ai/src/util/is-non-empty-object.ts
package util

// IsNonEmptyObject returns true if the given map is non-nil and has at least one key.
func IsNonEmptyObject(object map[string]interface{}) bool {
	return object != nil && len(object) > 0
}

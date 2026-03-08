// Ported from: packages/provider-utils/src/is-non-nullable.ts
package providerutils

// IsNonNil checks whether a pointer value is not nil.
// This is the Go equivalent of the TypeScript isNonNullable type guard.
func IsNonNil[T any](value *T) bool {
	return value != nil
}

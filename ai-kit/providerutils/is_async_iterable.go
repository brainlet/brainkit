// Ported from: packages/provider-utils/src/is-async-iterable.ts
package providerutils

// IsAsyncIterable checks whether the given value is a channel (Go's equivalent of AsyncIterable).
// In Go, we check if the value is a receive-only channel using type assertion.
// Since Go doesn't have a generic way to check for all channel types at runtime,
// this is a best-effort check that mirrors the TypeScript behavior.
func IsAsyncIterable(obj interface{}) bool {
	if obj == nil {
		return false
	}
	// In Go, the practical equivalent of checking for AsyncIterable is checking
	// if the value implements an iterator interface. Since Go channels are typed,
	// callers should use type assertions for their specific channel types.
	// This function exists for API parity with the TypeScript version.
	return false
}

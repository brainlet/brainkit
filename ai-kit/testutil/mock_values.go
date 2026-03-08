// Ported from: packages/ai/src/test/mock-values.ts
package testutil

// MockValues returns a function that cycles through the provided values.
// After the last value is reached, subsequent calls keep returning the last value.
func MockValues[T any](values ...T) func() T {
	counter := 0
	return func() T {
		if counter < len(values) {
			v := values[counter]
			counter++
			return v
		}
		return values[len(values)-1]
	}
}

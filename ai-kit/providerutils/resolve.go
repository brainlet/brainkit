// Ported from: packages/provider-utils/src/resolve.ts
package providerutils

// Resolvable represents a value that could be a raw value or a function returning a value.
// In TypeScript this also covered Promises; in Go, functions serve the same purpose.
type Resolvable[T any] struct {
	value *T
	fn    func() (T, error)
}

// NewResolvableValue creates a Resolvable from a raw value.
func NewResolvableValue[T any](v T) Resolvable[T] {
	return Resolvable[T]{value: &v}
}

// NewResolvableFunc creates a Resolvable from a function.
func NewResolvableFunc[T any](fn func() (T, error)) Resolvable[T] {
	return Resolvable[T]{fn: fn}
}

// Resolve resolves the value. If it's a function, calls it. Otherwise returns the raw value.
func (r Resolvable[T]) Resolve() (T, error) {
	if r.fn != nil {
		return r.fn()
	}
	if r.value != nil {
		return *r.value, nil
	}
	var zero T
	return zero, nil
}

// ResolveValue is a helper that resolves a Resolvable and returns just the value.
// Panics on error. Use Resolve() for error handling.
func ResolveValue[T any](r Resolvable[T]) T {
	v, err := r.Resolve()
	if err != nil {
		panic(err)
	}
	return v
}

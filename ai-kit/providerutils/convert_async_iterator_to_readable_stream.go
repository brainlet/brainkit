// Ported from: packages/provider-utils/src/convert-async-iterator-to-readable-stream.ts
package providerutils

// ConvertAsyncIteratorToReadableStream converts a channel (Go's equivalent of AsyncIterator)
// to another channel. In Go, channels already serve as the equivalent of ReadableStream,
// so this function acts as a pass-through adapter, faithfully mirroring the TypeScript API.
func ConvertAsyncIteratorToReadableStream[T any](iterator <-chan T) <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		for value := range iterator {
			ch <- value
		}
	}()
	return ch
}

// ConvertAsyncIteratorFuncToChannel converts a function that calls a callback for each
// value (similar to an async iterator's next() pattern) into a channel.
func ConvertAsyncIteratorFuncToChannel[T any](iterFunc func(yield func(T) bool)) <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		iterFunc(func(value T) bool {
			ch <- value
			return true
		})
	}()
	return ch
}

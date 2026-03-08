// Package pmap provides a Go port of p-map.
//
// This is a faithful 1:1 port of https://github.com/sindresorhus/p-map
// JS source: index.js (283 lines)
//
// Key differences from JS:
//   - Go generics [T, R any] replace JS dynamic typing
//   - Goroutines replace async/await concurrency model
//   - context.Context replaces AbortController/signal for cancellation
//   - Iterator interface replaces Symbol.iterator / Symbol.asyncIterator
//   - Awaitable[T] interface replaces Promise-valued input items
//   - Mapper returns (any, error) — return PMapSkip to skip, error for failures
//   - sync.Mutex provides goroutine safety (JS is single-threaded)
//   - Infinity constant (-1) replaces Number.POSITIVE_INFINITY
//   - Pointer helpers Int() and Bool() for optional struct fields
//
// Ported features:
//   - PMap with concurrency control and ordered results
//   - PMapIterable with concurrency and backpressure control
//   - PMapSkip sentinel for excluding mapper results
//   - StopOnError / AggregateError for error collection
//   - Context cancellation (equivalent to AbortController)
//   - Input validation with TypeError
//   - Slice and custom Iterator input support
//
// Usage:
//
//	// Basic concurrent mapping
//	results, err := pmap.PMap[string, int](urls, func(url string, i int) (any, error) {
//	    resp, err := http.Get(url)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return resp.StatusCode, nil
//	}, pmap.Options{Concurrency: pmap.Int(5)})
//
//	// Skip items with PMapSkip
//	results, err := pmap.PMap[int, int](numbers, func(n int, _ int) (any, error) {
//	    if n < 0 {
//	        return pmap.PMapSkip, nil
//	    }
//	    return n * 2, nil
//	})
//
//	// Streaming with backpressure
//	iter, err := pmap.PMapIterable[int, string](ids, fetchName, pmap.IterableOptions{
//	    Concurrency:  pmap.Int(8),
//	    Backpressure: pmap.Int(16),
//	})
//	for {
//	    name, done, err := iter.Next(ctx)
//	    if err != nil || done {
//	        break
//	    }
//	    process(name)
//	}
package pmap

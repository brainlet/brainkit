// Package lrucache provides a Go port of node-lru-cache.
//
// This is a faithful 1:1 port of https://github.com/isaacs/node-lru-cache
// TS source: src/index.ts (2955 lines)
//
// Key differences from TS:
//   - Go generics [K comparable, V any] replace TS generics <K, V, FC>
//   - sync.Mutex provides thread safety (TS is single-threaded)
//   - fetch/memo/forceFetch methods are omitted (JS async/Promise patterns)
//   - BackgroundFetch type is omitted (JS Promise pattern)
//   - AbortController integration is omitted (no Go equivalent needed)
//   - Iterators use callbacks or return slices instead of JS generators
//   - time.Now() replaces performance.now()
//   - (V, bool) return pattern replaces V | undefined
//   - Dispose callbacks are called with the lock held; DisposeAfter is called
//     after the lock is released. Neither callback should call cache methods.
//
// Ported features:
//   - Count-based eviction (Max)
//   - Size-based eviction (MaxSize, MaxEntrySize, SizeCalculation)
//   - TTL-based expiration (TTL, TTLResolution, TTLAutopurge)
//   - Dispose/DisposeAfter callbacks with reason codes
//   - OnInsert callback with reason codes
//   - Status tracking for observability
//   - AllowStale, NoDisposeOnSet, NoUpdateTTL, NoDeleteOnStaleGet
//   - UpdateAgeOnGet, UpdateAgeOnHas
//   - Dump/Load for serialization
//   - PurgeStale, Info, GetRemainingTTL
//   - Peek (get without LRU update)
//   - Pop (remove and return LRU item)
//   - Find (search by predicate)
//   - ForEach/RForEach iteration
//
// Usage:
//
//	cache := lrucache.New[string, int](lrucache.Options[string, int]{Max: 100})
//	cache.Set("key", 42)
//	val, ok := cache.Get("key") // 42, true
//	cache.Delete("key")
//	val, ok = cache.Get("key")  // 0, false
package lrucache

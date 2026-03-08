# lru-cache

Go port of [node-lru-cache](https://github.com/isaacs/node-lru-cache) (v11.x, 2955-line TypeScript source).

Faithful 1:1 port — same data structures, same algorithms, same option names. Designed for re-syncability with upstream changes.

## Usage

```go
import "github.com/brainlet/brainkit/lru-cache"

// Count-based eviction
cache := lrucache.New[string, int](lrucache.Options[string, int]{Max: 100})
cache.Set("key", 42)
val, ok := cache.Get("key") // 42, true
cache.Delete("key")

// Size-based eviction
cache := lrucache.New[string, []byte](lrucache.Options[string, []byte]{
    MaxSize: 1 << 20, // 1 MB
    SizeCalculation: func(v []byte, k string) int {
        return len(v)
    },
})

// TTL-based expiration
cache := lrucache.New[string, string](lrucache.Options[string, string]{
    Max: 500,
    TTL: int64((5 * time.Minute) / time.Millisecond),
})

// Dispose callback on eviction
cache := lrucache.New[string, *os.File](lrucache.Options[string, *os.File]{
    Max: 10,
    Dispose: func(v *os.File, k string, reason lrucache.DisposeReason) {
        v.Close()
    },
})

// Per-operation overrides
cache.Set("temp", 99, lrucache.SetOptions[string, int]{TTL: lrucache.Int64(1000)})
val, ok := cache.Get("temp", lrucache.GetOptions[int]{AllowStale: lrucache.Bool(true)})

// Background loading with context-based cancellation
fetching := lrucache.New[string, int](lrucache.Options[string, int]{
    Max: 100,
    FetchMethod: func(k string, stale *int, opts lrucache.FetcherOptions[string, int]) (int, bool, error) {
        return len(k), true, nil
    },
})
fetched, ok, err := fetching.Fetch("abc")
```

## TS Source

Ported from: `node-lru-cache/src/index.ts`

Upstream repo: https://github.com/isaacs/node-lru-cache

## Tests

```
cd lru-cache && go test ./...
go test -race ./lru-cache/...
```

Package tests cover basic operations, TTL, size tracking, dispose callbacks, fetch/forceFetch/memo flows, stale-while-revalidate, abort handling, pop/find/iteration, dump/load, purgeStale, info, getRemainingTTL, onInsert, and concurrent access.

## Adaptations

- `Fetch()` returns `(V, bool, error)` instead of a Promise. `ok=false, err=nil` means the upstream promise would have resolved to `undefined`.
- Fetch cancellation uses `context.Context` instead of `AbortController` / `AbortSignal`.
- Generator-based iterators are exposed as slice-returning methods and callback-based iteration helpers.

## TS → Go Patterns

### Generics

```
TS:  LRUCache<K extends {}, V extends {}, FC = unknown>
Go:  LRUCache[K comparable, V any]
```

Fetch/memo context is passed as `any` via `FetchOptions.Context` / `MemoOptions.Context`.

### undefined → nil Pointers

TS uses `undefined` for empty array slots and optional fields. Go uses pointer slices and pointer option fields:

```
TS:  #keyList: (K | undefined)[]           →  Go: keyList []*K
TS:  #valList: (V | BackgroundFetch | undefined)[]  →  Go: valList []*cacheValue[V]
TS:  ttl?: number                          →  Go: TTL *int64
TS:  allowStale?: boolean                  →  Go: AllowStale *bool
```

### Generators → Callbacks and Slices

TS uses `function*` generators for lazy iteration. Go returns concrete slices or uses callback-based iteration:

```
TS:  *entries(): Generator<[K, V]>         →  Go: Entries() [][2]any
TS:  *keys(): Generator<K>                 →  Go: Keys() []K
TS:  for (const [k,v] of cache) { ... }    →  Go: cache.ForEach(func(v V, k K) { ... })
```

### setTimeout / clearTimeout → time.AfterFunc

TS uses `setTimeout` for TTL autopurge timers. Go uses `time.AfterFunc`:

```
TS:  #ttls[i] = setTimeout(() => ..., ttl)
Go:  c.ttlTimers[i] = time.AfterFunc(dur, func() { c.Delete(key) })
```

### Thread Safety (Go addition)

TS is single-threaded. Go adds `sync.Mutex` for all public methods. Lock semantics:

- `Dispose` callbacks run **with** lock held (atomic cleanup)
- `DisposeAfter` callbacks run **after** lock release (prevents deadlock on blocking I/O)

### Value Comparison

TS uses `!==` which works on all types. Go generics with `any` constraint don't support `==`. Solved with `sameValue[V any]` using panic recovery:

```go
func sameValue[V any](a, b V) bool {
    defer func() { recover() }()
    return any(a) == any(b)
}
```

### Optional Parameters → Variadic Options

TS overloads and optional parameters become Go variadic option structs:

```
TS:  get(k: K, getOptions?: GetOptions): V | undefined
Go:  Get(k K, opts ...GetOptions[V]) (V, bool)
```

### Performance Internals

The core data structure is identical to TS: array-based doubly-linked list using parallel arrays (`keyList`, `valList`, `next`, `prev`) with a `keyMap` for O(1) lookups and a free stack for index reuse. This avoids pointer chasing and keeps data cache-friendly.

```
TS:  #keyList: (K | undefined)[]    →  Go: keyList []*K
TS:  #next: NumberArray              →  Go: next []int
TS:  #prev: NumberArray              →  Go: prev []int
TS:  #keyMap: Map<K, Index>          →  Go: keyMap map[K]int
TS:  #free: StackLike                →  Go: free []int
```

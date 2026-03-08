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
    SizeCalculation: func(v []byte, k string) int64 {
        return int64(len(v))
    },
})

// TTL-based expiration
cache := lrucache.New[string, string](lrucache.Options[string, string]{
    Max: 500,
    TTL: 5 * time.Minute,
})

// Dispose callback on eviction
cache := lrucache.New[string, *os.File](lrucache.Options[string, *os.File]{
    Max: 10,
    Dispose: func(v *os.File, k string, reason lrucache.DisposeReason) {
        v.Close()
    },
})

// Per-operation overrides
cache.Set("temp", 99, lrucache.SetOptions[int]{TTL: Int64p(1000)})
val, ok := cache.Get("key", lrucache.GetOptions[int]{AllowStale: Boolp(true)})
```

## TS Source

Ported from: `node-lru-cache/src/index.ts`

Upstream repo: https://github.com/isaacs/node-lru-cache

## Tests

```
cd lru-cache && go test ./...
go test -race ./lru-cache/...
```

37 tests covering: basic operations, TTL (basic, per-item, immortal, allowStale, noDeleteOnStaleGet, updateAgeOnGet, autopurge), dispose callbacks (evict/set/delete/clear reasons, noDisposeOnSet, disposeAfter), size tracking (maxSize, maxEntrySize), peek, pop, find, forEach/rForEach, keys/values/entries, dump/load, purgeStale, info, getRemainingTTL, onInsert, concurrent access.

## Omitted Features

These TS features rely on JavaScript async/Promise patterns with no direct Go equivalent:

- `fetch()` / `forceFetch()` / `memo()` — async data loading via fetchMethod
- `BackgroundFetch` type — Promise-based background data fetching
- `AbortController` / `AbortSignal` integration — JS cancellation pattern
- Generator-based iterators — replaced with callbacks and slice returns

## TS → Go Patterns

### Generics

```
TS:  LRUCache<K extends {}, V extends {}, FC = unknown>
Go:  LRUCache[K comparable, V any]
```

FC (fetch context) type parameter omitted since fetch methods are not ported.

### undefined → nil Pointers

TS uses `undefined` for empty array slots and optional fields. Go uses pointer slices and pointer option fields:

```
TS:  #keyList: (K | undefined)[]           →  Go: keyList []*K
TS:  #valList: (V | undefined)[]           →  Go: valList []*V
TS:  ttl?: number                          →  Go: TTL *int64
TS:  allowStale?: boolean                  →  Go: AllowStale *bool
```

### Generators → Callbacks and Slices

TS uses `function*` generators for lazy iteration. Go returns concrete slices or uses callback-based iteration:

```
TS:  *entries(): Generator<[K, V]>         →  Go: Entries() []Entry[K, V]
TS:  *keys(): Generator<K>                 →  Go: Keys() []K
TS:  for (const [k,v] of cache) { ... }    →  Go: cache.ForEach(func(v V, k K, cache *LRUCache[K,V]) { ... })
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

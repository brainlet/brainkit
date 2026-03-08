// Ported from: packages/core/src/processors/memory/embedding-cache.ts
package memory

import (
	"container/list"
	"sync"
)

const defaultCacheMaxSize = 1000

// globalEmbeddingCache is a process-wide embedding cache shared across all
// SemanticRecall instances.  This ensures embeddings are cached and reused even
// when new processor instances are created.
//
// Cache key format: xxhash hex of "${indexName}:${content}"
// Cache value: embedding vector ([]float64)
var globalEmbeddingCache = NewLRUCache[string, []float64](defaultCacheMaxSize)

// ---------------------------------------------------------------------------
// LRUCache – generic LRU cache (replaces npm lru-cache)
// ---------------------------------------------------------------------------

// LRUCache is a concurrency-safe, size-bounded LRU cache.
type LRUCache[K comparable, V any] struct {
	mu       sync.Mutex
	max      int
	items    map[K]*list.Element
	eviction *list.List
}

type lruEntry[K comparable, V any] struct {
	key   K
	value V
}

// NewLRUCache creates an LRU cache with the given maximum number of entries.
func NewLRUCache[K comparable, V any](max int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		max:      max,
		items:    make(map[K]*list.Element, max),
		eviction: list.New(),
	}
}

// Get retrieves a value and promotes it to the front.
// ok is false when the key is not present.
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.eviction.MoveToFront(el)
		return el.Value.(*lruEntry[K, V]).value, true
	}
	var zero V
	return zero, false
}

// Clear removes all entries from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]*list.Element, c.max)
	c.eviction.Init()
}

// Set inserts or updates a key-value pair.  If the cache is at capacity,
// the least-recently-used entry is evicted.
func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.eviction.MoveToFront(el)
		el.Value.(*lruEntry[K, V]).value = value
		return
	}

	if c.eviction.Len() >= c.max {
		// Evict least-recently-used entry.
		back := c.eviction.Back()
		if back != nil {
			c.eviction.Remove(back)
			delete(c.items, back.Value.(*lruEntry[K, V]).key)
		}
	}

	entry := &lruEntry[K, V]{key: key, value: value}
	el := c.eviction.PushFront(entry)
	c.items[key] = el
}

// Ported from: packages/core/src/cache/inmemory.ts
package cache

import (
	"fmt"
	"sync"
	"time"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

const (
	defaultMaxEntries = 1000
	defaultTTL        = 5 * time.Minute
)

// cacheEntry holds a value and its expiration time.
type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// InMemoryServerCache is a TTL-based in-memory cache with a max entry limit.
// Ported from InMemoryServerCache which uses @isaacs/ttlcache with max=1000, ttl=5min.
type InMemoryServerCache struct {
	*agentkit.MastraBase

	mu      sync.Mutex
	entries map[string]*cacheEntry
	// order tracks insertion order for eviction (oldest first).
	order []string
	max   int
	ttl   time.Duration
}

// Compile-time check that InMemoryServerCache implements MastraServerCache.
var _ MastraServerCache = (*InMemoryServerCache)(nil)

// NewInMemoryServerCache creates a new InMemoryServerCache with default settings.
func NewInMemoryServerCache() *InMemoryServerCache {
	return &InMemoryServerCache{
		MastraBase: agentkit.NewMastraBase(agentkit.MastraBaseOptions{
			Component: logger.RegisteredLoggerServerCache,
			Name:      "InMemoryServerCache",
		}),
		entries: make(map[string]*cacheEntry),
		order:   make([]string, 0),
		max:     defaultMaxEntries,
		ttl:     defaultTTL,
	}
}

// Get retrieves a value by key. Returns nil if the key does not exist or has expired.
func (c *InMemoryServerCache) Get(key string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, nil
	}
	if time.Now().After(entry.expiresAt) {
		c.deleteLocked(key)
		return nil, nil
	}
	return entry.value, nil
}

// Set stores a value with the default TTL. If the cache is at max capacity,
// the oldest entry is evicted.
func (c *InMemoryServerCache) Set(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setLocked(key, value)
	return nil
}

// setLocked stores a value; caller must hold c.mu.
func (c *InMemoryServerCache) setLocked(key string, value any) {
	// If the key already exists, update in place (no order change needed for simplicity).
	if _, exists := c.entries[key]; exists {
		c.entries[key] = &cacheEntry{
			value:     value,
			expiresAt: time.Now().Add(c.ttl),
		}
		return
	}

	// Evict oldest if at capacity.
	if len(c.entries) >= c.max {
		c.evictOldest()
	}

	c.entries[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.order = append(c.order, key)
}

// evictOldest removes the oldest entry from the cache; caller must hold c.mu.
func (c *InMemoryServerCache) evictOldest() {
	for len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		if _, exists := c.entries[oldest]; exists {
			delete(c.entries, oldest)
			return
		}
		// Key was already deleted; skip and try next.
	}
}

// Delete removes a key from the cache.
func (c *InMemoryServerCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteLocked(key)
	return nil
}

// deleteLocked removes a key; caller must hold c.mu.
// The key is lazily removed from c.order during eviction.
func (c *InMemoryServerCache) deleteLocked(key string) {
	delete(c.entries, key)
}

// Clear removes all entries from the cache.
func (c *InMemoryServerCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
	c.order = make([]string, 0)
	return nil
}

// ListPush appends a value to a list stored at key. If the key does not exist
// or the existing value is not a []any, a new list [value] is created.
func (c *InMemoryServerCache) ListPush(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if ok && !time.Now().After(entry.expiresAt) {
		if list, isList := entry.value.([]any); isList {
			entry.value = append(list, value)
			// Refresh TTL on modification.
			entry.expiresAt = time.Now().Add(c.ttl)
			return nil
		}
	}

	// Key doesn't exist, expired, or value is not a list: create new list.
	c.setLocked(key, []any{value})
	return nil
}

// ListLength returns the length of the list stored at key.
// Returns an error if the key does not exist or the value is not a []any.
func (c *InMemoryServerCache) ListLength(key string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		if !ok {
			return 0, fmt.Errorf("%s is not an array", key)
		}
		c.deleteLocked(key)
		return 0, fmt.Errorf("%s is not an array", key)
	}

	list, isList := entry.value.([]any)
	if !isList {
		return 0, fmt.Errorf("%s is not an array", key)
	}
	return len(list), nil
}

// ListFromTo returns a slice of the list stored at key, from index `from` to
// index `to` (inclusive), following Redis LRANGE semantics.
// If to is -1, it means "to end of list".
// Returns an empty slice if the key does not exist or the value is not a list.
func (c *InMemoryServerCache) ListFromTo(key string, from int, to int) ([]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			c.deleteLocked(key)
		}
		return []any{}, nil
	}

	list, isList := entry.value.([]any)
	if !isList {
		return []any{}, nil
	}

	length := len(list)

	// Handle negative indices (like Go/Python slice from end).
	if from < 0 {
		from = length + from
		if from < 0 {
			from = 0
		}
	}

	// to == -1 means "to end of array" (default in TS).
	var end int
	if to == -1 {
		end = length
	} else {
		if to < 0 {
			to = length + to
		}
		// Inclusive end index: add 1 for Go slice.
		end = to + 1
	}

	if from >= length {
		return []any{}, nil
	}
	if end > length {
		end = length
	}
	if from >= end {
		return []any{}, nil
	}

	// Return a copy to avoid mutation of internal state by callers.
	result := make([]any, end-from)
	copy(result, list[from:end])
	return result, nil
}

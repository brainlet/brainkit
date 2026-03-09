// Ported from: packages/core/src/cache/inmemory.ts
package cache

import (
	"fmt"
	"sync"
	"time"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/ttlcache"
)

const (
	defaultMaxEntries = 1000
	defaultTTL        = 5 * time.Minute
)

// InMemoryServerCache is a TTL-based in-memory cache with a max entry limit.
// Backed by ttlcache.TTLCache which handles TTL expiration and capacity eviction.
type InMemoryServerCache struct {
	*agentkit.MastraBase

	mu    sync.Mutex // protects read-modify-write sequences (e.g. ListPush)
	cache *ttlcache.TTLCache[string, any]
}

// Compile-time check that InMemoryServerCache implements MastraServerCache.
var _ MastraServerCache = (*InMemoryServerCache)(nil)

// NewInMemoryServerCache creates a new InMemoryServerCache with default settings.
func NewInMemoryServerCache() *InMemoryServerCache {
	max := defaultMaxEntries
	ttl := defaultTTL
	cache, err := ttlcache.New[string, any](ttlcache.TTLCacheOptions[string, any]{
		Max:           &max,
		TTL:           &ttl,
		CheckAgeOnGet: true,
	})
	if err != nil {
		panic(fmt.Sprintf("inmemory: failed to create ttlcache: %v", err))
	}

	return &InMemoryServerCache{
		MastraBase: agentkit.NewMastraBase(agentkit.MastraBaseOptions{
			Component: logger.RegisteredLoggerServerCache,
			Name:      "InMemoryServerCache",
		}),
		cache: cache,
	}
}

// Get retrieves a value by key. Returns nil if the key does not exist or has expired.
func (c *InMemoryServerCache) Get(key string) (any, error) {
	value, ok := c.cache.Get(key)
	if !ok {
		return nil, nil
	}
	return value, nil
}

// Set stores a value with the default TTL. If the cache is at max capacity,
// the oldest entry is evicted.
func (c *InMemoryServerCache) Set(key string, value any) error {
	c.cache.Set(key, value)
	return nil
}

// Delete removes a key from the cache.
func (c *InMemoryServerCache) Delete(key string) error {
	c.cache.Delete(key)
	return nil
}

// Clear removes all entries from the cache.
func (c *InMemoryServerCache) Clear() error {
	c.cache.Clear()
	return nil
}

// ListPush appends a value to a list stored at key. If the key does not exist
// or the existing value is not a []any, a new list [value] is created.
func (c *InMemoryServerCache) ListPush(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	existing, ok := c.cache.Get(key)
	if ok {
		if list, isList := existing.([]any); isList {
			// Copy into a new slice so the ttlcache sees a distinct value
			// (it skips updates when old and new compare equal by pointer).
			newList := make([]any, len(list)+1)
			copy(newList, list)
			newList[len(list)] = value
			c.cache.Set(key, newList)
			return nil
		}
	}

	// Key doesn't exist, expired, or value is not a list: create new list.
	c.cache.Set(key, []any{value})
	return nil
}

// ListLength returns the length of the list stored at key.
// Returns an error if the key does not exist or the value is not a []any.
func (c *InMemoryServerCache) ListLength(key string) (int, error) {
	value, ok := c.cache.Get(key)
	if !ok {
		return 0, fmt.Errorf("%s is not an array", key)
	}

	list, isList := value.([]any)
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
	value, ok := c.cache.Get(key)
	if !ok {
		return []any{}, nil
	}

	list, isList := value.([]any)
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

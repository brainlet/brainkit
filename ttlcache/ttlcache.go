package ttlcache

// Go port of @isaacs/ttlcache
// TS source: https://github.com/isaacs/ttlcache/blob/main/src/index.ts

import (
	"errors"
	"math"
	"time"
)

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// now returns the current time in milliseconds.
// TS source: now() function
func now() float64 {
	return float64(time.Now().UnixMilli())
}

// isPosInt returns true if n is a positive integer.
// TS source: isPosInt (line 104)
func isPosInt(n int64) bool {
	return n > 0
}

// isPosIntOrInf returns true if n is a positive integer or 0 (infinity).
// TS source: isPosIntOrInf (line 104)
func isPosIntOrInf(n int64) bool {
	return n >= 0
}

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

// ErrMaxMustBePositive is returned when max is not a positive integer.
var ErrMaxMustBePositive = errors.New("max must be a positive integer")

// ErrTTLMustBePositive is returned when TTL is not a positive duration.
var ErrTTLMustBePositive = errors.New("ttl must be a positive duration")

// ---------------------------------------------------------------------------
// Type definitions
// ---------------------------------------------------------------------------

// DisposeReason indicates why an item was removed from the cache.
// TS source: DisposeReason (line 21)
type DisposeReason string

const (
	// DisposeSet means the item was overwritten by a new value.
	DisposeSet DisposeReason = "set"
	// DisposeDelete means the item was explicitly deleted.
	DisposeDelete DisposeReason = "delete"
	// DisposeStale means the item was removed as stale/expired.
	DisposeStale DisposeReason = "stale"
	// DisposeEvict means the item was evicted to make room.
	DisposeEvict DisposeReason = "evict"
)

// DisposeFunction is called when an item is removed from the cache.
// TS source: DisposeFunction (lines 23-27)
type DisposeFunction[K comparable, V any] func(value V, key K, reason DisposeReason)

// ---------------------------------------------------------------------------
// Option types
// ---------------------------------------------------------------------------

// TTLCacheOptions configures the TTLCache behavior.
// TS source: TTLCacheOptions (lines 33-60)
type TTLCacheOptions[K comparable, V any] struct {
	// Max is the maximum number of items in the cache.
	// 0 means infinity (unbounded).
	Max *int

	// TTL is the default time-to-live for all items.
	// 0 means no expiration.
	// Must be a positive duration if set.
	TTL *time.Duration

	// UpdateAgeOnGet resets the TTL start time when Get() retrieves an item.
	UpdateAgeOnGet bool

	// CheckAgeOnGet checks the remaining age when getting items.
	// If not set, expired items may be returned that haven't been preemptively purged.
	CheckAgeOnGet bool

	// UpdateAgeOnHas resets the TTL start time when Has() checks an item.
	UpdateAgeOnHas bool

	// CheckAgeOnHas checks the remaining age when checking for an item's presence.
	// If not set, expired items will return true if they haven't been preemptively purged.
	CheckAgeOnHas bool

	// NoUpdateTTL prevents updating the TTL when setting a new value for an existing key.
	NoUpdateTTL bool

	// Dispose is called when an item is removed from the cache.
	Dispose DisposeFunction[K, V]

	// NoDisposeOnSet prevents calling dispose when setting an existing key to a new value.
	NoDisposeOnSet bool
}

// SetOptions overrides cache-level options for a single Set() call.
// TS source: SetOptions (lines 62-65)
type SetOptions[K comparable, V any] struct {
	// TTL overrides the cache TTL for this item. nil = use cache default.
	TTL *time.Duration

	// NoUpdateTTL overrides the cache NoUpdateTTL for this call.
	NoUpdateTTL *bool

	// NoDisposeOnSet overrides the cache NoDisposeOnSet for this call.
	NoDisposeOnSet *bool
}

// GetOptions overrides cache-level options for a single Get() call.
// TS source: GetOptions (lines 66-69)
type GetOptions[K comparable, V any] struct {
	// UpdateAgeOnGet overrides the cache UpdateAgeOnGet for this call.
	UpdateAgeOnGet *bool

	// TTL overrides the cache TTL for this item. nil = use cache default.
	TTL *time.Duration

	// CheckAgeOnGet overrides the cache CheckAgeOnGet for this call.
	CheckAgeOnGet *bool
}

// HasOptions overrides cache-level options for a single Has() call.
// TS source: HasOptions (lines 71-74)
type HasOptions[K comparable, V any] struct {
	// UpdateAgeOnHas overrides the cache UpdateAgeOnHas for this call.
	UpdateAgeOnHas *bool

	// TTL overrides the cache TTL for this item. nil = use cache default.
	TTL *time.Duration

	// CheckAgeOnHas overrides the cache CheckAgeOnHas for this call.
	CheckAgeOnHas *bool
}

// ---------------------------------------------------------------------------
// TTLCache struct
// TS source: class TTLCache (lines 76-131)
// ---------------------------------------------------------------------------

// TTLCache is a time-to-live cache with optional expiration and size limits.
// Not thread-safe (matching TS source which is single-threaded).
type TTLCache[K comparable, V any] struct {
	// expirations maps expiration timestamps to keys that expire at that time.
	expirations map[int64][]K

	// data stores the key-value pairs.
	data map[K]V

	// expirationMap maps keys to their expiration timestamp.
	// 0 represents Infinity (immortal keys).
	expirationMap map[K]int64

	// ttl is the default TTL in milliseconds.
	ttl *time.Duration

	// max is the maximum number of items (0 = infinity).
	max int

	// updateAgeOnGet resets TTL on Get() if true.
	updateAgeOnGet bool

	// updateAgeOnHas resets TTL on Has() if true.
	updateAgeOnHas bool

	// noUpdateTTL prevents updating TTL on Set() for existing keys.
	noUpdateTTL bool

	// noDisposeOnSet prevents calling dispose on Set() for existing keys.
	noDisposeOnSet bool

	// checkAgeOnGet checks TTL on Get() if true.
	checkAgeOnGet bool

	// checkAgeOnHas checks TTL on Has() if true.
	checkAgeOnHas bool

	// dispose is called when an item is removed.
	dispose DisposeFunction[K, V]

	// timer is the background timer for purging stale items.
	timer *time.Timer

	// timerExpiration is the expiration time of the current timer.
	timerExpiration *int64

	// immortalKeys contains keys that should never expire.
	immortalKeys map[K]struct{}
}

// New creates a new TTLCache with the given options.
// TS source: constructor (lines 93-131)
func New[K comparable, V any](opts TTLCacheOptions[K, V]) (*TTLCache[K, V], error) {
	// Validate max
	// TS source: lines 109-111
	max := 0
	if opts.Max != nil {
		max = *opts.Max
	}
	if max < 0 {
		return nil, ErrMaxMustBePositive
	}

	// Validate TTL
	// TS source: lines 104-108
	if opts.TTL != nil {
		ttl := *opts.TTL
		if ttl <= 0 {
			return nil, ErrTTLMustBePositive
		}
	}

	// Initialize dispose function
	// TS source: lines 120-127
	var dispose DisposeFunction[K, V]
	if opts.Dispose != nil {
		dispose = opts.Dispose
	} else {
		dispose = func(value V, key K, reason DisposeReason) {}
	}

	c := &TTLCache[K, V]{
		expirations:    make(map[int64][]K),
		data:           make(map[K]V),
		expirationMap:  make(map[K]int64),
		ttl:            opts.TTL,
		max:            max,
		updateAgeOnGet: opts.UpdateAgeOnGet,
		checkAgeOnGet:  opts.CheckAgeOnGet,
		updateAgeOnHas: opts.UpdateAgeOnHas,
		checkAgeOnHas:  opts.CheckAgeOnHas,
		noUpdateTTL:    opts.NoUpdateTTL,
		noDisposeOnSet: opts.NoDisposeOnSet,
		dispose:        dispose,
		immortalKeys:   make(map[K]struct{}),
	}

	return c, nil
}

// setTimer sets a timer to trigger at the given expiration time.
// TS source: setTimer (lines 133-162)
func (c *TTLCache[K, V]) setTimer(expiration int64, ttlMs int64) {
	// If there's already a timer set for a sooner expiration, don't reschedule
	if c.timerExpiration != nil && *c.timerExpiration < expiration {
		return
	}

	// Stop existing timer
	if c.timer != nil {
		c.timer.Stop()
	}

	// Calculate delay
	nowMs := int64(now())
	var delay time.Duration
	if expiration > nowMs {
		delay = time.Duration(expiration-nowMs) * time.Millisecond
	} else {
		delay = 1 * time.Millisecond
	}

	// Cap at TIMER_MAX (2^31 - 1 ms)
	const timerMax = 1<<31 - 1
	if delay.Milliseconds() > int64(timerMax) {
		delay = time.Duration(timerMax) * time.Millisecond
	}
	if delay < 0 {
		delay = 0
	}

	// Store expiration before starting timer
	c.timerExpiration = &expiration

	// Start the timer
	c.timer = time.AfterFunc(delay, func() {
		c.timer = nil
		c.timerExpiration = nil
		c.PurgeStale()

		// Schedule next purge if there are more expirations
		for exp := range c.expirations {
			if exp > 0 {
				c.setTimer(exp, exp-nowMs)
				break
			}
		}
	})
}

// cancelTimer stops the auto-purge timer.
// TS source: cancelTimer (lines 167-173)
func (c *TTLCache[K, V]) cancelTimer() {
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
		c.timerExpiration = nil
	}
}

// setTTL sets the TTL for a key.
// TS source: setTTL (lines 199-224)
func (c *TTLCache[K, V]) setTTL(key K, ttl *time.Duration) {
	// Remove key from current expiration list
	current, exists := c.expirationMap[key]
	if exists && current != 0 {
		expList := c.expirations[current]
		if len(expList) <= 1 {
			delete(c.expirations, current)
		} else {
			newList := make([]K, 0, len(expList)-1)
			for _, k := range expList {
				if k != key {
					newList = append(newList, k)
				}
			}
			c.expirations[current] = newList
		}
	}

	// Determine TTL value
	var ttlMs int64
	if ttl != nil {
		ttlMs = ttl.Milliseconds()
	}

	if ttlMs > 0 {
		// Key has expiration
		delete(c.immortalKeys, key)
		expiration := int64(now()) + ttlMs
		c.expirationMap[key] = expiration
		if c.expirations[expiration] == nil {
			c.expirations[expiration] = []K{key}
			c.setTimer(expiration, ttlMs)
		} else {
			c.expirations[expiration] = append(c.expirations[expiration], key)
		}
	} else {
		// Key is immortal (no expiration) - in TS this is Infinity
		c.immortalKeys[key] = struct{}{}
		c.expirationMap[key] = 0 // 0 represents Infinity
	}
}

// Set adds or updates an item in the cache.
// TS source: set (lines 226-261)
func (c *TTLCache[K, V]) Set(key K, value V, opts ...SetOptions[K, V]) *TTLCache[K, V] {
	// Merge opts with cache defaults
	var ttl *time.Duration
	var noUpdateTTL bool
	var noDisposeOnSet bool

	if len(opts) > 0 {
		o := opts[0]
		if o.TTL != nil {
			ttl = o.TTL
		} else {
			ttl = c.ttl
		}
		if o.NoUpdateTTL != nil {
			noUpdateTTL = *o.NoUpdateTTL
		} else {
			noUpdateTTL = c.noUpdateTTL
		}
		if o.NoDisposeOnSet != nil {
			noDisposeOnSet = *o.NoDisposeOnSet
		} else {
			noDisposeOnSet = c.noDisposeOnSet
		}
	} else {
		ttl = c.ttl
		noUpdateTTL = c.noUpdateTTL
		noDisposeOnSet = c.noDisposeOnSet
	}

	// Validate TTL is positive or zero (infinity)
	if ttl != nil && *ttl < 0 {
		return c
	}

	// Check if key already exists
	_, keyExists := c.expirationMap[key]

	if keyExists {
		// Key exists - may need to update TTL
		if !noUpdateTTL {
			c.setTTL(key, ttl)
		}

		// Get old value
		oldValue := c.data[key]

		// Update the value
		c.data[key] = value

		// Call dispose if noDisposeOnSet is false
		// (dispose is called when value is overwritten)
		if !noDisposeOnSet {
			c.dispose(oldValue, key, DisposeSet)
		}
	} else {
		// New key - set TTL and add to data
		c.setTTL(key, ttl)
		c.data[key] = value
	}

	// Evict if over capacity
	for c.Size() > c.max && c.max > 0 {
		c.purgeToCapacity()
	}

	return c
}

// Size returns the number of items in the cache.
// TS source: size getter (line 366)
func (c *TTLCache[K, V]) Size() int {
	return len(c.data)
}

// purgeToCapacity removes items to bring cache within max size.
// TS source: purgeToCapacity (lines 336-364)
func (c *TTLCache[K, V]) purgeToCapacity() {
	for exp, keys := range c.expirations {
		// If removing all keys at this expiration gets us under max
		if c.Size()-len(keys) >= c.max {
			delete(c.expirations, exp)
			for _, key := range keys {
				if val, ok := c.data[key]; ok {
					c.dispose(val, key, DisposeEvict)
				}
				delete(c.data, key)
				delete(c.expirationMap, key)
			}
		} else {
			// Remove oldest entries to meet max
			toRemove := c.Size() - c.max
			type kv struct {
				K K
				V V
			}
			var entries []kv
			for i := 0; i < toRemove && i < len(keys); i++ {
				key := keys[i]
				if val, ok := c.data[key]; ok {
					entries = append(entries, kv{key, val})
				}
				delete(c.data, key)
				delete(c.expirationMap, key)
			}
			for _, e := range entries {
				c.dispose(e.V, e.K, DisposeEvict)
			}
			// Update expirations list
			c.expirations[exp] = keys[toRemove:]
			return
		}
	}
}

// Has checks if a key exists in the cache.
// TS source: has (lines 263-282)
func (c *TTLCache[K, V]) Has(key K, opts ...HasOptions[K, V]) bool {
	// Handle options
	var checkAgeOnHas, updateAgeOnHas bool
	var ttl *time.Duration

	if len(opts) > 0 {
		o := opts[0]
		if o.CheckAgeOnHas != nil {
			checkAgeOnHas = *o.CheckAgeOnHas
		} else {
			checkAgeOnHas = c.checkAgeOnHas
		}
		if o.UpdateAgeOnHas != nil {
			updateAgeOnHas = *o.UpdateAgeOnHas
		} else {
			updateAgeOnHas = c.updateAgeOnHas
		}
		if o.TTL != nil {
			ttl = o.TTL
		}
	} else {
		checkAgeOnHas = c.checkAgeOnHas
		updateAgeOnHas = c.updateAgeOnHas
	}

	// Check if key exists
	if _, exists := c.data[key]; exists {
		// Check if expired (=== 0 in TS)
		if checkAgeOnHas && c.GetRemainingTTL(key) == 0 {
			c.Delete(key)
			return false
		}
		// Update TTL if needed
		if updateAgeOnHas {
			c.setTTL(key, ttl)
		}
		return true
	}
	return false
}

// GetRemainingTTL returns the remaining TTL for a key in milliseconds.
// TS source: getRemainingTTL (lines 284-291)
// Returns MaxInt64 for immortal keys (representing Infinity in TS)
func (c *TTLCache[K, V]) GetRemainingTTL(key K) int64 {
	expiration, exists := c.expirationMap[key]
	if !exists {
		return 0
	}
	// 0 represents Infinity in our implementation
	if expiration == 0 {
		return math.MaxInt64 // represents Infinity
	}
	remaining := expiration - int64(now())
	if remaining > 0 {
		return remaining
	}
	return 0 // expired
}

// Get retrieves a value from the cache.
// TS source: get (lines 293-310)
func (c *TTLCache[K, V]) Get(key K, opts ...GetOptions[K, V]) (V, bool) {
	// Handle options
	var updateAgeOnGet, checkAgeOnGet bool
	var ttl *time.Duration

	if len(opts) > 0 {
		o := opts[0]
		if o.UpdateAgeOnGet != nil {
			updateAgeOnGet = *o.UpdateAgeOnGet
		} else {
			updateAgeOnGet = c.updateAgeOnGet
		}
		if o.CheckAgeOnGet != nil {
			checkAgeOnGet = *o.CheckAgeOnGet
		} else {
			checkAgeOnGet = c.checkAgeOnGet
		}
		if o.TTL != nil {
			ttl = o.TTL
		}
	} else {
		updateAgeOnGet = c.updateAgeOnGet
		checkAgeOnGet = c.checkAgeOnGet
	}

	// Get value from data map
	val, exists := c.data[key]
	if !exists {
		var zero V
		return zero, false
	}

	// Check if expired when checkAgeOnGet is true (=== 0 in TS)
	if checkAgeOnGet && c.GetRemainingTTL(key) == 0 {
		// Inline delete - remove from all maps
		delete(c.data, key)
		exp, _ := c.expirationMap[key]
		delete(c.expirationMap, key)
		delete(c.immortalKeys, key)
		// Remove from expirations list
		if exp != 0 {
			if list, ok := c.expirations[exp]; ok {
				if len(list) <= 1 {
					delete(c.expirations, exp)
				} else {
					newList := make([]K, 0, len(list)-1)
					for _, k := range list {
						if k != key {
							newList = append(newList, k)
						}
					}
					c.expirations[exp] = newList
				}
			}
		}
		var zero V
		return zero, false
	}

	// Update TTL if updateAgeOnGet is true
	if updateAgeOnGet {
		c.setTTL(key, ttl)
	}

	return val, true
}

// Delete removes a key from the cache.
// TS source: delete (lines 312-334)
func (c *TTLCache[K, V]) Delete(key K) bool {
	current, exists := c.expirationMap[key]
	if !exists {
		return false
	}

	value := c.data[key]
	delete(c.data, key)
	delete(c.expirationMap, key)
	delete(c.immortalKeys, key)

	// Remove from expirations list
	if list, ok := c.expirations[current]; ok {
		if len(list) <= 1 {
			delete(c.expirations, current)
		} else {
			newList := make([]K, 0, len(list)-1)
			for _, k := range list {
				if k != key {
					newList = append(newList, k)
				}
			}
			c.expirations[current] = newList
		}
	}

	c.dispose(value, key, DisposeDelete)

	if c.Size() == 0 {
		c.cancelTimer()
	}

	return true
}

// Clear removes all items from the cache.
// TS source: clear (lines 186-197)
func (c *TTLCache[K, V]) Clear() {
	// Collect entries for dispose if custom dispose is set
	if c.dispose != nil {
		for k, v := range c.data {
			c.dispose(v, k, DisposeDelete)
		}
	}

	c.data = make(map[K]V)
	c.expirationMap = make(map[K]int64)
	c.expirations = make(map[int64][]K)
	c.immortalKeys = make(map[K]struct{})
	c.cancelTimer()
}

// PurgeStale removes all expired items from the cache.
// TS source: purgeStale (lines 370-396)
func (c *TTLCache[K, V]) PurgeStale() {
	n := int64(now())

	for exp, keys := range c.expirations {
		if exp <= 0 || exp > n {
			continue
		}

		// Copy keys since we'll be modifying the map
		keysCopy := make([]K, len(keys))
		copy(keysCopy, keys)
		delete(c.expirations, exp)

		for _, key := range keysCopy {
			if val, ok := c.data[key]; ok {
				c.dispose(val, key, DisposeStale)
			}
			delete(c.data, key)
			delete(c.expirationMap, key)
		}
	}

	if c.Size() == 0 {
		c.cancelTimer()
	}
}

// Entry represents a key-value pair in the cache.
type Entry[K any, V any] struct {
	Key   K
	Value V
}

// Entries returns a channel of all key-value pairs in the cache.
// TS source: entries (lines 398-407)
func (c *TTLCache[K, V]) Entries() <-chan Entry[K, V] {
	ch := make(chan Entry[K, V], c.Size())
	go func() {
		defer close(ch)
		for _, keys := range c.expirations {
			for _, key := range keys {
				if val, ok := c.data[key]; ok {
					ch <- Entry[K, V]{Key: key, Value: val}
				}
			}
		}
		for key := range c.immortalKeys {
			if val, ok := c.data[key]; ok {
				ch <- Entry[K, V]{Key: key, Value: val}
			}
		}
	}()
	return ch
}

// Keys returns a channel of all keys in the cache.
// TS source: keys (lines 408-417)
func (c *TTLCache[K, V]) Keys() <-chan K {
	ch := make(chan K, c.Size())
	go func() {
		defer close(ch)
		for _, keys := range c.expirations {
			for _, key := range keys {
				ch <- key
			}
		}
		for key := range c.immortalKeys {
			ch <- key
		}
	}()
	return ch
}

// Values returns a channel of all values in the cache.
// TS source: values (lines 418-427)
func (c *TTLCache[K, V]) Values() <-chan V {
	ch := make(chan V, c.Size())
	go func() {
		defer close(ch)
		for _, keys := range c.expirations {
			for _, key := range keys {
				if val, ok := c.data[key]; ok {
					ch <- val
				}
			}
		}
		for key := range c.immortalKeys {
			if val, ok := c.data[key]; ok {
				ch <- val
			}
		}
	}()
	return ch
}

package lrucache

// Go port of node-lru-cache
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/src/index.ts

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Helper functions
// TS source: lines 97-104 (isPosInt, shouldWarn)
// ---------------------------------------------------------------------------

// Bool returns a pointer to the given bool value.
// Convenience helper for use with option structs.
func Bool(v bool) *bool { return &v }

// Int64 returns a pointer to the given int64 value.
// Convenience helper for use with option structs.
func Int64(v int64) *int64 { return &v }

// isPosInt returns true if n is a positive integer.
// TS source: line 103
func isPosInt(n int) bool {
	return n > 0
}

// sameValue attempts to compare two values of generic type V.
// For comparable types (int, string, etc.) this returns the equality result.
// For uncomparable types (slices, maps) it returns false (conservative).
// TS source: uses !== operator which is identity for objects, value for primitives.
func sameValue[V any](a, b V) (same bool) {
	defer func() {
		if r := recover(); r != nil {
			same = false
		}
	}()
	var ai, bi any = a, b
	return ai == bi
}

// ---------------------------------------------------------------------------
// Type definitions
// TS source: lines 182-600 (LRUCache namespace)
// ---------------------------------------------------------------------------

// DisposeReason indicates why an item was removed from the cache.
// TS source: LRUCache.DisposeReason (line 213)
type DisposeReason string

const (
	// DisposeEvict means the item was evicted as LRU to make room.
	DisposeEvict DisposeReason = "evict"
	// DisposeSet means the item was overwritten by a new value.
	DisposeSet DisposeReason = "set"
	// DisposeDelete means the item was explicitly deleted or cleared.
	DisposeDelete DisposeReason = "delete"
	// DisposeExpire means the item's TTL expired.
	DisposeExpire DisposeReason = "expire"
)

// InsertReason indicates why an item was added to the cache.
// TS source: LRUCache.InsertReason (line 238)
type InsertReason string

const (
	// InsertAdd means the item was not found in the cache and was added.
	InsertAdd InsertReason = "add"
	// InsertUpdate means the item was in the cache with the same value.
	InsertUpdate InsertReason = "update"
	// InsertReplace means the item was in the cache and was replaced.
	InsertReplace InsertReason = "replace"
)

// Status tracks the internal behavior of cache operations for observability.
// Pass a non-nil *Status to Get/Set/Has methods to collect operation details.
// TS source: LRUCache.Status (line 280)
type Status[V any] struct {
	// Set is the result of a Set() operation: "add", "update", "replace", or "miss".
	Set string
	// TTL is the TTL stored for the item in milliseconds.
	TTL int64
	// Start is the start time for TTL calculation.
	Start int64
	// Now is the timestamp used for TTL calculation.
	Now int64
	// RemainingTTL is the remaining time to live in milliseconds.
	RemainingTTL int64
	// EntrySize is the calculated size for the item.
	EntrySize int
	// TotalCalculatedSize is the total calculated size of the cache.
	TotalCalculatedSize int
	// MaxEntrySizeExceeded is true if the item was not stored due to exceeding MaxEntrySize.
	MaxEntrySizeExceeded bool
	// OldValue is the previous value on update/replace.
	OldValue *V
	// Has is the result of a Has() operation: "hit", "stale", or "miss".
	Has string
	// Get is the result of a Get() operation: "stale", "hit", or "miss".
	Get string
	// ReturnedStale is true if a stale value was returned.
	ReturnedStale bool
}

// Entry represents a serializable cache entry for Dump/Load and Info operations.
// TS source: implied by dump/load/info methods
type Entry[V any] struct {
	Value V
	TTL   int64 // Milliseconds. 0 means no TTL.
	Start int64 // For Dump: Unix milliseconds. For Info: current time.
	Size  int   // Item size. 0 means not tracked.
}

// ---------------------------------------------------------------------------
// Option types
// TS source: lines 600-1100 (OptionsBase, option variants, per-method options)
// ---------------------------------------------------------------------------

// Options configures the LRUCache behavior.
// At least one of Max, MaxSize, or TTL (with TTLAutopurge) must be set.
// TS source: LRUCache.OptionsBase (line 600+)
type Options[K comparable, V any] struct {
	// Max is the maximum number of items in the cache.
	// Set to 0 to use only MaxSize or TTL-based eviction (arrays grow dynamically).
	Max int

	// TTL is the default time-to-live in milliseconds for all items.
	// 0 means TTL is disabled. Must be a non-negative integer.
	TTL int64

	// TTLResolution is the minimum time in ms between staleness checks.
	// Higher values improve performance at the cost of staleness accuracy.
	// Default: 0 (check every time). TS source: line ~1480
	TTLResolution int64

	// TTLAutopurge creates a timer for each item that auto-deletes it when TTL expires.
	// WARNING: has performance impact with many items (one goroutine timer per item).
	TTLAutopurge bool

	// UpdateAgeOnGet resets the TTL start time when Get() retrieves an item.
	UpdateAgeOnGet bool

	// UpdateAgeOnHas resets the TTL start time when Has() checks an item.
	UpdateAgeOnHas bool

	// AllowStale returns expired items from Get() instead of deleting them.
	AllowStale bool

	// NoDisposeOnSet prevents calling Dispose when a value is overwritten by Set().
	NoDisposeOnSet bool

	// NoUpdateTTL prevents updating the TTL when a value is overwritten by Set().
	NoUpdateTTL bool

	// NoDeleteOnStaleGet prevents deleting stale items when retrieved via Get().
	NoDeleteOnStaleGet bool

	// MaxSize is the maximum total calculated size of all items.
	// When exceeded, the least recently used items are evicted.
	MaxSize int

	// MaxEntrySize is the maximum size allowed for a single item.
	// Items exceeding this are not stored. Defaults to MaxSize if not set.
	MaxEntrySize int

	// SizeCalculation returns the size of an item for size-based eviction.
	// Required when using MaxSize without explicit Size in SetOptions.
	SizeCalculation func(value V, key K) int

	// Dispose is called synchronously when an item is removed from the cache.
	// Called with the lock held - MUST NOT call any cache methods.
	// TS source: OptionsBase.dispose (line ~900)
	Dispose func(value V, key K, reason DisposeReason)

	// DisposeAfter is called after the cache operation completes (lock released).
	// Safe to call cache methods from this callback.
	// TS source: OptionsBase.disposeAfter (line ~930)
	DisposeAfter func(value V, key K, reason DisposeReason)

	// OnInsert is called when an item is added to or updated in the cache.
	// Called with the lock held - MUST NOT call any cache methods.
	// TS source: OptionsBase.onInsert (line ~960)
	OnInsert func(value V, key K, reason InsertReason)

	// NowFn provides a custom monotonic clock for testing.
	// Must return time in milliseconds. Default: time.Now() monotonic ms.
	NowFn func() int64
}

// SetOptions overrides cache-level options for a single Set() call.
// TS source: LRUCache.SetOptions (combined from multiple TS interfaces)
type SetOptions[K comparable, V any] struct {
	// TTL overrides the cache TTL for this item. nil = use cache default.
	// Set to Int64(0) to make the item immortal (no TTL).
	TTL *int64
	// Start provides a custom TTL start time. 0 = use current time.
	Start int64
	// NoDisposeOnSet overrides the cache NoDisposeOnSet for this call.
	NoDisposeOnSet *bool
	// NoUpdateTTL overrides the cache NoUpdateTTL for this call.
	NoUpdateTTL *bool
	// SizeCalculation overrides the cache SizeCalculation for this call.
	SizeCalculation func(value V, key K) int
	// Size provides an explicit size, skipping SizeCalculation.
	Size int
	// Status collects operation details for observability.
	Status *Status[V]
}

// GetOptions overrides cache-level options for a single Get() call.
// TS source: LRUCache.GetOptions (line 588)
type GetOptions[V any] struct {
	// AllowStale overrides the cache AllowStale for this call.
	AllowStale *bool
	// UpdateAgeOnGet overrides the cache UpdateAgeOnGet for this call.
	UpdateAgeOnGet *bool
	// NoDeleteOnStaleGet overrides the cache NoDeleteOnStaleGet for this call.
	NoDeleteOnStaleGet *bool
	// Status collects operation details for observability.
	Status *Status[V]
}

// HasOptions overrides cache-level options for a single Has() call.
// TS source: LRUCache.HasOptions (line 580)
type HasOptions[V any] struct {
	// UpdateAgeOnHas overrides the cache UpdateAgeOnHas for this call.
	UpdateAgeOnHas *bool
	// Status collects operation details for observability.
	Status *Status[V]
}

// PeekOptions overrides cache-level options for a single Peek() call.
// TS source: LRUCache.PeekOptions (line 599)
type PeekOptions struct {
	// AllowStale overrides the cache AllowStale for this call.
	AllowStale *bool
}

// ---------------------------------------------------------------------------
// Internal types
// ---------------------------------------------------------------------------

// disposeTask is a deferred disposal operation for DisposeAfter callbacks.
// TS source: DisposeTask type (line 176)
type disposeTask[K comparable, V any] struct {
	value  V
	key    K
	reason DisposeReason
}

// ---------------------------------------------------------------------------
// LRUCache struct
// TS source: class LRUCache (line ~1100)
// ---------------------------------------------------------------------------

// LRUCache is a least-recently-used cache with optional TTL and size tracking.
// Thread-safe via internal sync.Mutex.
type LRUCache[K comparable, V any] struct {
	mu sync.Mutex

	// --- Configuration (set in constructor, some publicly mutable in TS) ---

	max            int   // Maximum item count (0 = unbounded by count)
	maxSize        int   // Maximum total size (0 = no size tracking)
	maxEntrySize   int   // Maximum single entry size (0 = no limit)
	ttl            int64 // Default TTL in ms (0 = no TTL)
	ttlResolution  int64 // Time caching resolution in ms
	ttlAutopurge   bool  // Auto-delete expired items via timers
	updateAgeOnGet bool
	updateAgeOnHas bool
	allowStale     bool
	noDisposeOnSet bool
	noUpdateTTL    bool
	noDeleteOnStaleGet bool
	sizeCalculation    func(V, K) int

	// --- Callbacks ---
	dispose      func(V, K, DisposeReason)
	disposeAfter func(V, K, DisposeReason)
	onInsert     func(V, K, InsertReason)

	// --- Feature flags (cached from callbacks for fast checks) ---
	hasDispose      bool
	hasDisposeAfter bool
	hasOnInsert     bool

	// --- Core data structure ---
	// Parallel arrays + index-based doubly-linked list.
	// TS source: constructor initialization (lines ~1410-1470)
	//
	// The linked list goes: head (LRU) → ... → tail (MRU)
	// next[i] = index of next more-recently-used item
	// prev[i] = index of next less-recently-used item
	keyMap  map[K]int // key → array index (TS: #keyMap)
	keyList []*K      // index → key pointer, nil = empty slot (TS: #keyList)
	valList []*V      // index → value pointer, nil = empty slot (TS: #valList)
	next    []int     // forward linked list pointers (TS: #next)
	prev    []int     // backward linked list pointers (TS: #prev)
	head    int       // LRU end index (TS: #head)
	tail    int       // MRU end index (TS: #tail)
	free    []int     // stack of freed indices for reuse (TS: #free via Stack)
	size    int       // current item count (TS: #size)

	// --- TTL tracking (nil slices = TTL not initialized) ---
	// TS source: #initializeTTLTracking (lines ~1520-1625)
	ttls   []int64 // per-item TTL in ms (0 = no TTL for this item)
	starts []int64 // per-item start time in monotonic ms

	// --- Size tracking (nil slice = sizes not tracked) ---
	// TS source: #initializeSizeTracking (lines ~1641-1694)
	sizes          []int
	calculatedSize int

	// --- TTL autopurge timers ---
	autopurgeTimers []*time.Timer

	// --- Deferred dispose queue ---
	disposed []disposeTask[K, V]

	// --- Clock ---
	nowFn     func() int64 // monotonic time in ms
	cachedNow atomic.Int64 // cached time value for ttlResolution debouncing
}

// ---------------------------------------------------------------------------
// Constructor
// TS source: constructor (lines ~1380-1520)
// ---------------------------------------------------------------------------

// New creates a new LRUCache with the given options.
// Panics if options are invalid (matching TS TypeError throws).
func New[K comparable, V any](o Options[K, V]) *LRUCache[K, V] {
	max := o.Max

	// Validate max
	// TS source: lines ~1385-1400
	if max < 0 {
		panic(fmt.Sprintf("max option must be a nonnegative integer, got %d", max))
	}

	// Validate TTL
	// TS source: lines ~1400-1410
	if o.TTL < 0 {
		panic(fmt.Sprintf("ttl option must be a nonnegative integer, got %d", o.TTL))
	}
	if o.TTL > 0 && !isPosInt(int(o.TTL)) {
		panic("ttl option must be a positive integer when set")
	}

	// Validate maxSize / maxEntrySize
	if o.MaxSize < 0 {
		panic(fmt.Sprintf("maxSize option must be a nonnegative integer, got %d", o.MaxSize))
	}
	if o.MaxEntrySize < 0 {
		panic(fmt.Sprintf("maxEntrySize option must be a nonnegative integer, got %d", o.MaxEntrySize))
	}

	// Validate sizeCalculation
	if o.SizeCalculation != nil && o.MaxSize == 0 && o.MaxEntrySize == 0 {
		// sizeCalculation without maxSize is allowed but useless
	}

	// Must have at least one limiting factor
	// TS source: lines ~1395-1405
	if max == 0 && o.MaxSize == 0 && (o.TTL == 0 || !o.TTLAutopurge) {
		panic("at least one of max, maxSize, or ttl (with ttlAutopurge) must be specified")
	}

	// TTL resolution defaults to 0
	// TS source: line ~1480
	ttlResolution := o.TTLResolution
	if ttlResolution < 0 {
		ttlResolution = 0
	}

	// MaxEntrySize defaults to MaxSize
	// TS source: line ~1490
	maxEntrySize := o.MaxEntrySize
	if maxEntrySize == 0 && o.MaxSize > 0 {
		maxEntrySize = o.MaxSize
	}

	// Default clock
	nowFn := o.NowFn
	if nowFn == nil {
		nowFn = defaultNowFn
	}

	c := &LRUCache[K, V]{
		max:            max,
		maxSize:        o.MaxSize,
		maxEntrySize:   maxEntrySize,
		ttl:            o.TTL,
		ttlResolution:  ttlResolution,
		ttlAutopurge:   o.TTLAutopurge,
		updateAgeOnGet: o.UpdateAgeOnGet,
		updateAgeOnHas: o.UpdateAgeOnHas,
		allowStale:     o.AllowStale,
		noDisposeOnSet: o.NoDisposeOnSet,
		noUpdateTTL:    o.NoUpdateTTL,
		noDeleteOnStaleGet: o.NoDeleteOnStaleGet,
		sizeCalculation:    o.SizeCalculation,
		dispose:      o.Dispose,
		disposeAfter: o.DisposeAfter,
		onInsert:     o.OnInsert,
		hasDispose:      o.Dispose != nil,
		hasDisposeAfter: o.DisposeAfter != nil,
		hasOnInsert:     o.OnInsert != nil,
		nowFn:           nowFn,
	}

	// Pre-allocate parallel arrays
	// TS source: lines ~1410-1470
	// When max > 0, pre-allocate. When max == 0, start empty and grow.
	cap := max
	if cap == 0 {
		cap = 8 // Initial capacity for dynamic growth
	}

	c.keyMap = make(map[K]int, cap)
	c.keyList = make([]*K, cap)
	c.valList = make([]*V, cap)
	c.next = make([]int, cap)
	c.prev = make([]int, cap)
	c.free = make([]int, 0, cap)

	// Initialize TTL tracking if TTL is set
	// TS source: lines ~1500-1510
	if o.TTL > 0 {
		c.initializeTTLTracking()
	}

	// Initialize size tracking if maxSize is set
	// TS source: lines ~1515-1520
	if o.MaxSize > 0 {
		c.initializeSizeTracking()
	}

	return c
}

// defaultNowFn returns the current monotonic time in milliseconds.
func defaultNowFn() int64 {
	return time.Now().UnixMilli()
}

// ---------------------------------------------------------------------------
// TTL initialization
// TS source: #initializeTTLTracking (lines ~1520-1625)
// ---------------------------------------------------------------------------

// initializeTTLTracking sets up the TTL tracking arrays.
// Called lazily on first TTL usage or eagerly if TTL is set in constructor.
func (c *LRUCache[K, V]) initializeTTLTracking() {
	cap := len(c.keyList)
	c.ttls = make([]int64, cap)
	c.starts = make([]int64, cap)

	if c.ttlAutopurge {
		c.autopurgeTimers = make([]*time.Timer, cap)
	}
}

// ---------------------------------------------------------------------------
// Size initialization
// TS source: #initializeSizeTracking (lines ~1641-1694)
// ---------------------------------------------------------------------------

// initializeSizeTracking sets up the size tracking array.
func (c *LRUCache[K, V]) initializeSizeTracking() {
	cap := len(c.keyList)
	c.sizes = make([]int, cap)
	c.calculatedSize = 0
}

// ---------------------------------------------------------------------------
// Dynamic array growth (for max=0 mode)
// Not in TS source - JS arrays grow dynamically, Go slices need explicit growth.
// ---------------------------------------------------------------------------

// ensureIndex grows all parallel arrays if needed to accommodate the given index.
func (c *LRUCache[K, V]) ensureIndex(index int) {
	if index < len(c.keyList) {
		return
	}

	// Grow to at least double or index+1
	newCap := len(c.keyList) * 2
	if newCap <= index {
		newCap = index + 1
	}
	if newCap < 8 {
		newCap = 8
	}

	c.keyList = growSlicePtr[K](c.keyList, newCap)
	c.valList = growSlicePtr[V](c.valList, newCap)
	c.next = growSliceInt(c.next, newCap)
	c.prev = growSliceInt(c.prev, newCap)
	if c.ttls != nil {
		c.ttls = growSliceInt64(c.ttls, newCap)
	}
	if c.starts != nil {
		c.starts = growSliceInt64(c.starts, newCap)
	}
	if c.sizes != nil {
		c.sizes = growSliceInt(c.sizes, newCap)
	}
	if c.autopurgeTimers != nil {
		old := c.autopurgeTimers
		c.autopurgeTimers = make([]*time.Timer, newCap)
		copy(c.autopurgeTimers, old)
	}
}

func growSlicePtr[T any](s []*T, newCap int) []*T {
	grown := make([]*T, newCap)
	copy(grown, s)
	return grown
}

func growSliceInt(s []int, newCap int) []int {
	grown := make([]int, newCap)
	copy(grown, s)
	return grown
}

func growSliceInt64(s []int64, newCap int) []int64 {
	grown := make([]int64, newCap)
	copy(grown, s)
	return grown
}

// ---------------------------------------------------------------------------
// Internal TTL helpers
// TS source: lines ~1520-1640
// ---------------------------------------------------------------------------

// getNow returns the current monotonic time in ms, with ttlResolution caching.
// TS source: getNow closure inside #initializeTTLTracking (lines ~1588-1604)
func (c *LRUCache[K, V]) getNow() int64 {
	if c.ttlResolution > 0 {
		if cached := c.cachedNow.Load(); cached > 0 {
			return cached
		}
	}
	now := c.nowFn()
	if c.ttlResolution > 0 {
		c.cachedNow.Store(now)
		time.AfterFunc(time.Duration(c.ttlResolution)*time.Millisecond, func() {
			c.cachedNow.Store(0)
		})
	}
	return now
}

// isStale returns true if the item at index has exceeded its TTL.
// TS source: #isStale (lines ~1620-1624)
func (c *LRUCache[K, V]) isStale(index int) bool {
	if c.ttls == nil {
		return false
	}
	t := c.ttls[index]
	s := c.starts[index]
	if t == 0 || s == 0 {
		return false
	}
	return c.getNow()-s > t
}

// setItemTTL sets the TTL for an item at the given index.
// TS source: #setItemTTL inside #initializeTTLTracking (lines ~1538-1567)
func (c *LRUCache[K, V]) setItemTTL(index int, ttl int64, start int64) {
	if c.ttls == nil {
		return
	}
	c.ttls[index] = ttl
	if ttl > 0 {
		if start > 0 {
			c.starts[index] = start
		} else {
			c.starts[index] = c.getNow()
		}
	} else {
		c.starts[index] = 0
	}

	// TTL autopurge: create a timer to auto-delete after TTL
	// TS source: lines ~1549-1566
	if c.ttlAutopurge && c.autopurgeTimers != nil {
		// Cancel existing timer
		if c.autopurgeTimers[index] != nil {
			c.autopurgeTimers[index].Stop()
			c.autopurgeTimers[index] = nil
		}
		if ttl > 0 && c.keyList[index] != nil {
			k := *c.keyList[index]
			c.autopurgeTimers[index] = time.AfterFunc(
				time.Duration(ttl)*time.Millisecond,
				func() {
					c.mu.Lock()
					c.internalDelete(k, DisposeExpire)
					tasks := c.drainDisposed()
					c.mu.Unlock()
					c.runDisposeTasks(tasks)
				},
			)
		}
	}
}

// updateItemAge resets the TTL start time for an item.
// TS source: #updateItemAge inside #initializeTTLTracking (lines ~1570-1572)
func (c *LRUCache[K, V]) updateItemAge(index int) {
	if c.ttls == nil || c.starts == nil {
		return
	}
	if c.ttls[index] != 0 {
		c.starts[index] = c.getNow()
	}
}

// statusTTL fills TTL information into a Status object.
// TS source: #statusTTL inside #initializeTTLTracking (lines ~1574-1586)
func (c *LRUCache[K, V]) statusTTL(status *Status[V], index int) {
	if status == nil || c.ttls == nil {
		return
	}
	ttl := c.ttls[index]
	start := c.starts[index]
	if ttl == 0 || start == 0 {
		return
	}
	now := c.getNow()
	status.TTL = ttl
	status.Start = start
	status.Now = now
	status.RemainingTTL = ttl - (now - start)
}

// ---------------------------------------------------------------------------
// Internal size helpers
// TS source: lines ~1641-1719
// ---------------------------------------------------------------------------

// removeItemSize removes the tracked size of an item.
// TS source: #removeItemSize inside #initializeSizeTracking (lines ~1645-1648)
func (c *LRUCache[K, V]) removeItemSize(index int) {
	if c.sizes == nil {
		return
	}
	c.calculatedSize -= c.sizes[index]
	c.sizes[index] = 0
}

// addItemSize adds the tracked size of an item, evicting as needed.
// TS source: #addItemSize inside #initializeSizeTracking (lines ~1676-1693)
func (c *LRUCache[K, V]) addItemSize(index int, size int, status *Status[V]) {
	if c.sizes == nil {
		return
	}
	c.sizes[index] = size
	if c.maxSize > 0 {
		maxAllowed := c.maxSize - c.sizes[index]
		for c.calculatedSize > maxAllowed {
			c.evict(true)
		}
	}
	c.calculatedSize += c.sizes[index]
	if status != nil {
		status.EntrySize = size
		status.TotalCalculatedSize = c.calculatedSize
	}
}

// requireSize validates and returns the size for an item.
// TS source: #requireSize inside #initializeSizeTracking (lines ~1649-1675)
// and the default no-op version (lines ~1702-1719)
func (c *LRUCache[K, V]) requireSize(k K, v V, size int, sizeCalc func(V, K) int) int {
	if c.sizes == nil {
		// Not tracking sizes
		if size > 0 || sizeCalc != nil {
			panic("cannot set size without setting maxSize or maxEntrySize on cache")
		}
		return 0
	}

	if !isPosInt(size) {
		if sizeCalc != nil {
			size = sizeCalc(v, k)
			if !isPosInt(size) {
				panic("sizeCalculation return invalid (expect positive integer)")
			}
		} else {
			panic("invalid size value (must be positive integer). " +
				"When maxSize or maxEntrySize is used, sizeCalculation " +
				"or size must be set.")
		}
	}
	return size
}

// ---------------------------------------------------------------------------
// Internal linked list helpers
// TS source: lines ~2816-2842
// ---------------------------------------------------------------------------

// connect links two indices in the doubly-linked list.
// TS source: #connect (lines ~2816-2819)
func (c *LRUCache[K, V]) connect(p, n int) {
	c.prev[n] = p
	c.next[p] = n
}

// moveToTail moves an item to the MRU (tail) position.
// TS source: #moveToTail (lines ~2821-2842)
func (c *LRUCache[K, V]) moveToTail(index int) {
	if index == c.tail {
		return
	}
	if index == c.head {
		c.head = c.next[index]
	} else {
		c.connect(c.prev[index], c.next[index])
	}
	c.connect(c.tail, index)
	c.tail = index
}

// ---------------------------------------------------------------------------
// Eviction
// TS source: #evict (lines ~2218-2252)
// ---------------------------------------------------------------------------

// evict removes the least recently used item (at head).
// If free is true, the slot is cleared and pushed to the free stack.
// If free is false, the slot is immediately reused (index returned).
// Returns the index of the evicted item.
func (c *LRUCache[K, V]) evict(free bool) int {
	head := c.head
	k := c.keyList[head]
	v := c.valList[head]

	// Call dispose callbacks
	if k != nil && v != nil {
		if c.hasDispose || c.hasDisposeAfter {
			if c.hasDispose {
				c.dispose(*v, *k, DisposeEvict)
			}
			if c.hasDisposeAfter {
				c.disposed = append(c.disposed, disposeTask[K, V]{
					value: *v, key: *k, reason: DisposeEvict,
				})
			}
		}
	}

	c.removeItemSize(head)

	// Cancel autopurge timer
	if c.autopurgeTimers != nil && c.autopurgeTimers[head] != nil {
		c.autopurgeTimers[head].Stop()
		c.autopurgeTimers[head] = nil
	}

	// If freeing, clear the slot and push to free stack
	if free {
		c.keyList[head] = nil
		c.valList[head] = nil
		c.free = append(c.free, head)
	}

	// Update linked list
	if c.size == 1 {
		c.head = 0
		c.tail = 0
		c.free = c.free[:0]
	} else {
		c.head = c.next[head]
	}

	// Remove from key map
	if k != nil {
		delete(c.keyMap, *k)
	}
	c.size--

	return head
}

// ---------------------------------------------------------------------------
// Internal delete
// TS source: #delete (lines ~2853-2904)
// ---------------------------------------------------------------------------

// internalDelete removes an item by key. Returns true if the item existed.
// Caller must hold the lock.
func (c *LRUCache[K, V]) internalDelete(k K, reason DisposeReason) bool {
	if c.size == 0 {
		return false
	}

	index, exists := c.keyMap[k]
	if !exists {
		return false
	}

	// Cancel autopurge timer
	if c.autopurgeTimers != nil && index < len(c.autopurgeTimers) && c.autopurgeTimers[index] != nil {
		c.autopurgeTimers[index].Stop()
		c.autopurgeTimers[index] = nil
	}

	if c.size == 1 {
		c.internalClear(reason)
		return true
	}

	// Remove size tracking
	c.removeItemSize(index)

	// Call dispose
	v := c.valList[index]
	if v != nil {
		if c.hasDispose || c.hasDisposeAfter {
			if c.hasDispose {
				c.dispose(*v, k, reason)
			}
			if c.hasDisposeAfter {
				c.disposed = append(c.disposed, disposeTask[K, V]{
					value: *v, key: k, reason: reason,
				})
			}
		}
	}

	// Remove from key map
	delete(c.keyMap, k)
	c.keyList[index] = nil
	c.valList[index] = nil

	// Fix linked list
	// TS source: lines ~2881-2890
	if index == c.tail {
		c.tail = c.prev[index]
	} else if index == c.head {
		c.head = c.next[index]
	} else {
		c.next[c.prev[index]] = c.next[index]
		c.prev[c.next[index]] = c.prev[index]
	}

	c.size--
	c.free = append(c.free, index)

	return true
}

// ---------------------------------------------------------------------------
// Internal clear
// TS source: #clear (lines ~2912-2954)
// ---------------------------------------------------------------------------

// internalClear removes all items from the cache.
// Caller must hold the lock.
func (c *LRUCache[K, V]) internalClear(reason DisposeReason) {
	// Call dispose for each item (reverse order, LRU to MRU)
	// TS source: lines ~2913-2926
	c.forEachRIndex(true, func(index int) bool {
		v := c.valList[index]
		k := c.keyList[index]
		if v != nil && k != nil {
			if c.hasDispose {
				c.dispose(*v, *k, reason)
			}
			if c.hasDisposeAfter {
				c.disposed = append(c.disposed, disposeTask[K, V]{
					value: *v, key: *k, reason: reason,
				})
			}
		}
		return true
	})

	// Clear all data structures
	// TS source: lines ~2928-2946
	c.keyMap = make(map[K]int, len(c.keyList))
	for i := range c.valList {
		c.valList[i] = nil
	}
	for i := range c.keyList {
		c.keyList[i] = nil
	}
	if c.ttls != nil {
		for i := range c.ttls {
			c.ttls[i] = 0
		}
	}
	if c.starts != nil {
		for i := range c.starts {
			c.starts[i] = 0
		}
	}
	if c.autopurgeTimers != nil {
		for i, t := range c.autopurgeTimers {
			if t != nil {
				t.Stop()
			}
			c.autopurgeTimers[i] = nil
		}
	}
	if c.sizes != nil {
		for i := range c.sizes {
			c.sizes[i] = 0
		}
	}

	c.head = 0
	c.tail = 0
	c.free = c.free[:0]
	c.calculatedSize = 0
	c.size = 0
}

// ---------------------------------------------------------------------------
// Index validation and iteration
// TS source: #isValidIndex (lines ~1757-1762), #indexes/#rindexes (lines ~1721-1755)
// ---------------------------------------------------------------------------

// isValidIndex checks that an index is valid and maps to the correct key.
// TS source: #isValidIndex (lines ~1757-1762)
func (c *LRUCache[K, V]) isValidIndex(index int) bool {
	if index < 0 || index >= len(c.keyList) {
		return false
	}
	k := c.keyList[index]
	if k == nil {
		return false
	}
	mappedIndex, exists := c.keyMap[*k]
	return exists && mappedIndex == index
}

// forEachIndex iterates from tail (MRU) to head (LRU).
// fn returns false to stop iteration.
// TS source: *#indexes generator (lines ~1721-1737)
func (c *LRUCache[K, V]) forEachIndex(allowStale bool, fn func(index int) bool) {
	if c.size == 0 {
		return
	}
	for i := c.tail; ; {
		if !c.isValidIndex(i) {
			break
		}
		if allowStale || !c.isStale(i) {
			if !fn(i) {
				return
			}
		}
		if i == c.head {
			break
		}
		i = c.prev[i]
	}
}

// forEachRIndex iterates from head (LRU) to tail (MRU).
// fn returns false to stop iteration.
// TS source: *#rindexes generator (lines ~1739-1755)
func (c *LRUCache[K, V]) forEachRIndex(allowStale bool, fn func(index int) bool) {
	if c.size == 0 {
		return
	}
	for i := c.head; ; {
		if !c.isValidIndex(i) {
			break
		}
		if allowStale || !c.isStale(i) {
			if !fn(i) {
				return
			}
		}
		if i == c.tail {
			break
		}
		i = c.next[i]
	}
}

// ---------------------------------------------------------------------------
// Dispose queue helpers
// ---------------------------------------------------------------------------

// drainDisposed collects pending DisposeAfter tasks. Caller must hold the lock.
func (c *LRUCache[K, V]) drainDisposed() []disposeTask[K, V] {
	if !c.hasDisposeAfter || len(c.disposed) == 0 {
		return nil
	}
	tasks := make([]disposeTask[K, V], len(c.disposed))
	copy(tasks, c.disposed)
	c.disposed = c.disposed[:0]
	return tasks
}

// runDisposeTasks calls DisposeAfter for each pending task. Lock must NOT be held.
func (c *LRUCache[K, V]) runDisposeTasks(tasks []disposeTask[K, V]) {
	for _, t := range tasks {
		c.disposeAfter(t.value, t.key, t.reason)
	}
}

// ===========================================================================
// PUBLIC METHODS
// ===========================================================================

// ---------------------------------------------------------------------------
// Set
// TS source: set() method (lines ~2042-2188)
// ---------------------------------------------------------------------------

// Set adds or updates a value in the cache.
// Returns the cache for method chaining.
// If the value already exists, it is moved to the most recently used position.
func (c *LRUCache[K, V]) Set(k K, v V, opts ...SetOptions[K, V]) *LRUCache[K, V] {
	c.mu.Lock()

	// Resolve options
	// TS source: lines ~2081-2088
	var opt SetOptions[K, V]
	if len(opts) > 0 {
		opt = opts[0]
	}

	ttl := c.ttl
	if opt.TTL != nil {
		ttl = *opt.TTL
	}
	start := opt.Start
	noDisposeOnSet := c.noDisposeOnSet
	if opt.NoDisposeOnSet != nil {
		noDisposeOnSet = *opt.NoDisposeOnSet
	}
	sizeCalc := c.sizeCalculation
	if opt.SizeCalculation != nil {
		sizeCalc = opt.SizeCalculation
	}
	status := opt.Status
	noUpdateTTL := c.noUpdateTTL
	if opt.NoUpdateTTL != nil {
		noUpdateTTL = *opt.NoUpdateTTL
	}

	// Calculate size
	// TS source: lines ~2090-2095
	size := c.requireSize(k, v, opt.Size, sizeCalc)

	// Check maxEntrySize
	// TS source: lines ~2098-2106
	if c.maxEntrySize > 0 && size > c.maxEntrySize {
		if status != nil {
			status.Set = "miss"
			status.MaxEntrySizeExceeded = true
		}
		c.internalDelete(k, DisposeSet)
		tasks := c.drainDisposed()
		c.mu.Unlock()
		c.runDisposeTasks(tasks)
		return c
	}

	// Find existing index
	// TS source: lines ~2107-2108
	index := -1
	exists := false
	if c.size > 0 {
		index, exists = c.keyMap[k]
	}

	if !exists {
		// --- Addition ---
		// TS source: lines ~2109-2127

		// Allocate index
		// TS source: lines ~2110-2114
		if c.size == 0 {
			index = c.tail
		} else if len(c.free) > 0 {
			index = c.free[len(c.free)-1]
			c.free = c.free[:len(c.free)-1]
		} else if c.max > 0 && c.size == c.max {
			index = c.evict(false)
		} else {
			index = c.size
		}

		// Ensure arrays can accommodate this index (for max=0 dynamic growth)
		c.ensureIndex(index)

		// Store key and value
		kCopy := k
		vCopy := v
		c.keyList[index] = &kCopy
		c.valList[index] = &vCopy
		c.keyMap[k] = index

		// Link into doubly-linked list at tail (MRU position)
		// TS source: lines ~2118-2120
		c.next[c.tail] = index
		c.prev[index] = c.tail
		c.tail = index
		c.size++

		c.addItemSize(index, size, status)
		if status != nil {
			status.Set = "add"
		}
		noUpdateTTL = false // Always set TTL for new entries

		if c.hasOnInsert {
			c.onInsert(v, k, InsertAdd)
		}
	} else {
		// --- Update ---
		// TS source: lines ~2128-2170

		c.moveToTail(index)

		oldVal := c.valList[index]
		same := oldVal != nil && sameValue(v, *oldVal)

		if !same {
			// Value changed: dispose old, store new
			// TS source: lines ~2132-2162
			// (Skipping BackgroundFetch handling - JS-only pattern)
			if !noDisposeOnSet && oldVal != nil {
				if c.hasDispose {
					c.dispose(*oldVal, k, DisposeSet)
				}
				if c.hasDisposeAfter {
					c.disposed = append(c.disposed, disposeTask[K, V]{
						value: *oldVal, key: k, reason: DisposeSet,
					})
				}
			}
			c.removeItemSize(index)
			c.addItemSize(index, size, status)
			vCopy := v
			c.valList[index] = &vCopy
			if status != nil {
				status.Set = "replace"
				if oldVal != nil {
					oldV := *oldVal
					status.OldValue = &oldV
				}
			}
		} else if status != nil {
			status.Set = "update"
		}

		if c.hasOnInsert {
			reason := InsertReplace
			if same {
				reason = InsertUpdate
			}
			c.onInsert(v, k, reason)
		}
	}

	// Handle TTL
	// TS source: lines ~2171-2179
	if ttl != 0 && c.ttls == nil {
		c.initializeTTLTracking()
	}
	if c.ttls != nil {
		if !noUpdateTTL {
			c.setItemTTL(index, ttl, start)
		}
		if status != nil {
			c.statusTTL(status, index)
		}
	}

	// Drain dispose queue
	// TS source: lines ~2180-2186
	tasks := c.drainDisposed()
	c.mu.Unlock()
	c.runDisposeTasks(tasks)

	return c
}

// ---------------------------------------------------------------------------
// Get
// TS source: get() method (lines ~2764-2814)
// ---------------------------------------------------------------------------

// Get retrieves a value from the cache.
// Updates the recency of the entry (moves it to MRU position).
// Returns (value, true) if found, or (zero, false) if not found or stale.
func (c *LRUCache[K, V]) Get(k K, opts ...GetOptions[V]) (V, bool) {
	c.mu.Lock()

	var opt GetOptions[V]
	if len(opts) > 0 {
		opt = opts[0]
	}

	allowStale := c.allowStale
	if opt.AllowStale != nil {
		allowStale = *opt.AllowStale
	}
	updateAgeOnGet := c.updateAgeOnGet
	if opt.UpdateAgeOnGet != nil {
		updateAgeOnGet = *opt.UpdateAgeOnGet
	}
	noDeleteOnStaleGet := c.noDeleteOnStaleGet
	if opt.NoDeleteOnStaleGet != nil {
		noDeleteOnStaleGet = *opt.NoDeleteOnStaleGet
	}
	status := opt.Status

	var zero V

	index, exists := c.keyMap[k]
	if !exists {
		if status != nil {
			status.Get = "miss"
		}
		c.mu.Unlock()
		return zero, false
	}

	vp := c.valList[index]
	if vp == nil {
		if status != nil {
			status.Get = "miss"
		}
		c.mu.Unlock()
		return zero, false
	}

	value := *vp

	if status != nil {
		c.statusTTL(status, index)
	}

	if c.isStale(index) {
		// TS source: lines ~2776-2794
		if status != nil {
			status.Get = "stale"
		}
		// (Skipping BackgroundFetch handling - JS-only pattern)
		if !noDeleteOnStaleGet {
			c.internalDelete(k, DisposeExpire)
		}
		tasks := c.drainDisposed()
		c.mu.Unlock()
		c.runDisposeTasks(tasks)
		if allowStale {
			if status != nil {
				status.ReturnedStale = true
			}
			return value, true
		}
		return zero, false
	}

	// Not stale
	// TS source: lines ~2795-2810
	if status != nil {
		status.Get = "hit"
	}
	c.moveToTail(index)
	if updateAgeOnGet {
		c.updateItemAge(index)
	}
	c.mu.Unlock()
	return value, true
}

// ---------------------------------------------------------------------------
// Has
// TS source: has() method (lines ~2270-2298)
// ---------------------------------------------------------------------------

// Has checks if a key exists in the cache without updating recency.
// Returns false for stale items unless they're technically still stored.
func (c *LRUCache[K, V]) Has(k K, opts ...HasOptions[V]) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	var opt HasOptions[V]
	if len(opts) > 0 {
		opt = opts[0]
	}

	updateAgeOnHas := c.updateAgeOnHas
	if opt.UpdateAgeOnHas != nil {
		updateAgeOnHas = *opt.UpdateAgeOnHas
	}
	status := opt.Status

	index, exists := c.keyMap[k]
	if !exists {
		if status != nil {
			status.Has = "miss"
		}
		return false
	}

	if c.valList[index] == nil {
		if status != nil {
			status.Has = "miss"
		}
		return false
	}

	if !c.isStale(index) {
		if updateAgeOnHas {
			c.updateItemAge(index)
		}
		if status != nil {
			status.Has = "hit"
			c.statusTTL(status, index)
		}
		return true
	}

	// Stale
	if status != nil {
		status.Has = "stale"
		c.statusTTL(status, index)
	}
	return false
}

// ---------------------------------------------------------------------------
// Delete
// TS source: delete() method (lines ~2849-2851)
// ---------------------------------------------------------------------------

// Delete removes an item from the cache. Returns true if the item existed.
func (c *LRUCache[K, V]) Delete(k K) bool {
	c.mu.Lock()
	deleted := c.internalDelete(k, DisposeDelete)
	tasks := c.drainDisposed()
	c.mu.Unlock()
	c.runDisposeTasks(tasks)
	return deleted
}

// ---------------------------------------------------------------------------
// Clear
// TS source: clear() method (lines ~2909-2911)
// ---------------------------------------------------------------------------

// Clear removes all items from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	c.internalClear(DisposeDelete)
	tasks := c.drainDisposed()
	c.mu.Unlock()
	c.runDisposeTasks(tasks)
}

// ---------------------------------------------------------------------------
// Peek
// TS source: peek() method (lines ~2307-2316)
// ---------------------------------------------------------------------------

// Peek returns a value without updating the recency or deleting stale items.
// Returns (value, true) if found, or (zero, false) if not found.
func (c *LRUCache[K, V]) Peek(k K, opts ...PeekOptions) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var opt PeekOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	allowStale := c.allowStale
	if opt.AllowStale != nil {
		allowStale = *opt.AllowStale
	}

	var zero V

	index, exists := c.keyMap[k]
	if !exists {
		return zero, false
	}

	if !allowStale && c.isStale(index) {
		return zero, false
	}

	v := c.valList[index]
	if v == nil {
		return zero, false
	}
	return *v, true
}

// ---------------------------------------------------------------------------
// Pop
// TS source: pop() method (lines ~2194-2216)
// ---------------------------------------------------------------------------

// Pop removes and returns the least recently used item.
// Returns (value, true) if the cache was non-empty, or (zero, false) if empty.
func (c *LRUCache[K, V]) Pop() (V, bool) {
	c.mu.Lock()

	var zero V
	var result V
	found := false

	for c.size > 0 {
		val := c.valList[c.head]
		c.evict(true)
		// (Skipping BackgroundFetch handling - JS-only pattern)
		if val != nil {
			result = *val
			found = true
			break
		}
	}

	tasks := c.drainDisposed()
	c.mu.Unlock()
	c.runDisposeTasks(tasks)

	if found {
		return result, true
	}
	return zero, false
}

// ---------------------------------------------------------------------------
// Find
// TS source: find() method (lines ~1873-1885)
// ---------------------------------------------------------------------------

// Find returns the first value for which fn returns true, iterating from
// most recently used to least recently used. The matched item is moved
// to the MRU position (like Get).
// Returns (value, true) if found, or (zero, false) if not found.
func (c *LRUCache[K, V]) Find(fn func(v V, k K) bool, opts ...GetOptions[V]) (V, bool) {
	c.mu.Lock()

	var zero V
	var foundKey *K

	c.forEachIndex(false, func(index int) bool {
		v := c.valList[index]
		if v == nil {
			return true
		}
		k := c.keyList[index]
		if k == nil {
			return true
		}
		if fn(*v, *k) {
			foundKey = k
			return false // stop iteration
		}
		return true
	})

	if foundKey == nil {
		c.mu.Unlock()
		return zero, false
	}

	// Use Get to update recency (release lock first, Get will re-acquire)
	k := *foundKey
	c.mu.Unlock()
	return c.Get(k, opts...)
}

// ---------------------------------------------------------------------------
// ForEach / RForEach
// TS source: forEach/rforEach methods (lines ~1898-1924)
// ---------------------------------------------------------------------------

// ForEach calls fn for each item, from most recently used to least recently used.
// Does not update age or recency of use.
func (c *LRUCache[K, V]) ForEach(fn func(v V, k K)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.forEachIndex(false, func(index int) bool {
		v := c.valList[index]
		k := c.keyList[index]
		if v != nil && k != nil {
			fn(*v, *k)
		}
		return true
	})
}

// RForEach calls fn for each item, from least recently used to most recently used.
// Does not update age or recency of use.
func (c *LRUCache[K, V]) RForEach(fn func(v V, k K)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.forEachRIndex(false, func(index int) bool {
		v := c.valList[index]
		k := c.keyList[index]
		if v != nil && k != nil {
			fn(*v, *k)
		}
		return true
	})
}

// ---------------------------------------------------------------------------
// PurgeStale
// TS source: purgeStale() method (lines ~1930-1939)
// ---------------------------------------------------------------------------

// PurgeStale deletes all stale entries. Returns true if anything was removed.
func (c *LRUCache[K, V]) PurgeStale() bool {
	c.mu.Lock()
	deleted := false

	// Collect stale keys first to avoid modifying during iteration
	var staleKeys []K
	c.forEachRIndex(true, func(index int) bool {
		if c.isStale(index) {
			k := c.keyList[index]
			if k != nil {
				staleKeys = append(staleKeys, *k)
			}
		}
		return true
	})

	for _, k := range staleKeys {
		c.internalDelete(k, DisposeExpire)
		deleted = true
	}

	tasks := c.drainDisposed()
	c.mu.Unlock()
	c.runDisposeTasks(tasks)
	return deleted
}

// ---------------------------------------------------------------------------
// Info
// TS source: info() method (lines ~1953-1977)
// ---------------------------------------------------------------------------

// Info returns extended information about a cache entry.
// Returns nil if the key is not present.
// Always returns stale values if their info is in the cache.
func (c *LRUCache[K, V]) Info(k K) *Entry[V] {
	c.mu.Lock()
	defer c.mu.Unlock()

	index, exists := c.keyMap[k]
	if !exists {
		return nil
	}

	v := c.valList[index]
	if v == nil {
		return nil
	}

	entry := &Entry[V]{Value: *v}

	if c.ttls != nil && c.starts != nil {
		ttl := c.ttls[index]
		start := c.starts[index]
		if ttl > 0 && start > 0 {
			remain := ttl - (c.nowFn() - start)
			entry.TTL = remain
			entry.Start = time.Now().UnixMilli()
		}
	}

	if c.sizes != nil {
		entry.Size = c.sizes[index]
	}

	return entry
}

// ---------------------------------------------------------------------------
// GetRemainingTTL
// TS source: getRemainingTTL() method (lines ~1606-1618)
// ---------------------------------------------------------------------------

// GetRemainingTTL returns the remaining TTL in milliseconds for a key.
// Returns 0 if the key is not in the cache.
// Returns a large value (effectively infinity) if the item has no TTL.
func (c *LRUCache[K, V]) GetRemainingTTL(k K) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	index, exists := c.keyMap[k]
	if !exists {
		return 0
	}

	if c.ttls == nil || c.starts == nil {
		// No TTL tracking, item lives forever
		return int64(^uint64(0) >> 1) // max int64 (≈ infinity)
	}

	ttl := c.ttls[index]
	start := c.starts[index]
	if ttl == 0 || start == 0 {
		// No TTL for this specific item
		return int64(^uint64(0) >> 1) // max int64 (≈ infinity)
	}

	age := c.getNow() - start
	return ttl - age
}

// ---------------------------------------------------------------------------
// Size / Max / CalculatedSize accessors
// TS source: get size(), get max, get calculatedSize (various locations)
// ---------------------------------------------------------------------------

// Size returns the current number of items in the cache.
func (c *LRUCache[K, V]) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.size
}

// Max returns the maximum number of items the cache can hold.
func (c *LRUCache[K, V]) Max() int {
	return c.max
}

// CalculatedSize returns the total calculated size of all items in the cache.
func (c *LRUCache[K, V]) CalculatedSize() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calculatedSize
}

// ---------------------------------------------------------------------------
// Collection methods (snapshot the current state)
// TS source: entries/keys/values generators (lines ~1768-1860)
// ---------------------------------------------------------------------------

// Entries returns all [key, value] pairs from most recently used to least.
func (c *LRUCache[K, V]) Entries() [][2]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([][2]any, 0, c.size)
	c.forEachIndex(false, func(index int) bool {
		v := c.valList[index]
		k := c.keyList[index]
		if v != nil && k != nil {
			result = append(result, [2]any{*k, *v})
		}
		return true
	})
	return result
}

// Keys returns all keys from most recently used to least.
func (c *LRUCache[K, V]) Keys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]K, 0, c.size)
	c.forEachIndex(false, func(index int) bool {
		k := c.keyList[index]
		if k != nil && c.valList[index] != nil {
			result = append(result, *k)
		}
		return true
	})
	return result
}

// Values returns all values from most recently used to least.
func (c *LRUCache[K, V]) Values() []V {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]V, 0, c.size)
	c.forEachIndex(false, func(index int) bool {
		v := c.valList[index]
		if v != nil && c.keyList[index] != nil {
			result = append(result, *v)
		}
		return true
	})
	return result
}

// REntries returns all [key, value] pairs from least recently used to most.
func (c *LRUCache[K, V]) REntries() [][2]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([][2]any, 0, c.size)
	c.forEachRIndex(false, func(index int) bool {
		v := c.valList[index]
		k := c.keyList[index]
		if v != nil && k != nil {
			result = append(result, [2]any{*k, *v})
		}
		return true
	})
	return result
}

// RKeys returns all keys from least recently used to most.
func (c *LRUCache[K, V]) RKeys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]K, 0, c.size)
	c.forEachRIndex(false, func(index int) bool {
		k := c.keyList[index]
		if k != nil && c.valList[index] != nil {
			result = append(result, *k)
		}
		return true
	})
	return result
}

// RValues returns all values from least recently used to most.
func (c *LRUCache[K, V]) RValues() []V {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]V, 0, c.size)
	c.forEachRIndex(false, func(index int) bool {
		v := c.valList[index]
		if v != nil && c.keyList[index] != nil {
			result = append(result, *v)
		}
		return true
	})
	return result
}

// ---------------------------------------------------------------------------
// Dump / Load
// TS source: dump() (lines ~1992-2014) / load() (lines ~2025-2040)
// ---------------------------------------------------------------------------

// DumpEntry is a key-entry pair returned by Dump() and accepted by Load().
type DumpEntry[K comparable, V any] struct {
	Key   K
	Entry Entry[V]
}

// Dump returns all cache entries as a serializable slice.
// Order: most recently used to least recently used.
// TTL start times are relative to portable Unix timestamps (Date.now() in TS).
// Stale entries are always included.
func (c *LRUCache[K, V]) Dump() []DumpEntry[K, V] {
	c.mu.Lock()
	defer c.mu.Unlock()

	var result []DumpEntry[K, V]

	c.forEachIndex(true, func(index int) bool {
		k := c.keyList[index]
		v := c.valList[index]
		if k == nil || v == nil {
			return true
		}

		entry := Entry[V]{Value: *v}

		if c.ttls != nil && c.starts != nil {
			entry.TTL = c.ttls[index]
			if c.starts[index] > 0 {
				// Convert monotonic start to portable timestamp
				// TS source: lines ~2004-2006
				age := c.nowFn() - c.starts[index]
				entry.Start = time.Now().UnixMilli() - age
			}
		}
		if c.sizes != nil {
			entry.Size = c.sizes[index]
		}

		// Prepend (TS uses arr.unshift which prepends)
		result = append([]DumpEntry[K, V]{{Key: *k, Entry: entry}}, result...)
		return true
	})

	return result
}

// Load clears the cache and loads entries from a previous Dump().
// TTL start times are assumed to be portable Unix timestamps.
// TS source: load() method (lines ~2025-2040)
func (c *LRUCache[K, V]) Load(entries []DumpEntry[K, V]) {
	c.Clear()

	c.mu.Lock()
	for _, de := range entries {
		entry := de.Entry
		var setOpts SetOptions[K, V]
		if entry.TTL > 0 {
			setOpts.TTL = &entry.TTL
		}
		if entry.Start > 0 {
			// Convert portable timestamp to monotonic time
			// TS source: lines ~2033-2036
			age := time.Now().UnixMilli() - entry.Start
			start := c.nowFn() - age
			setOpts.Start = start
		}
		if entry.Size > 0 {
			setOpts.Size = entry.Size
		}
		c.mu.Unlock()
		c.Set(de.Key, entry.Value, setOpts)
		c.mu.Lock()
	}
	c.mu.Unlock()
}

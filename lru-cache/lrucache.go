package lrucache

// Go port of node-lru-cache
// TS source: https://github.com/isaacs/node-lru-cache/blob/main/src/index.ts

import (
	"context"
	"fmt"
	"os"
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
	// DisposeFetch means an inflight fetch resolved to undefined or failed.
	DisposeFetch DisposeReason = "fetch"
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
	// Fetch is the result of a Fetch() operation.
	Fetch string
	// FetchDispatched indicates that FetchMethod was invoked.
	FetchDispatched bool
	// FetchUpdated indicates that the fetched value updated the cache.
	FetchUpdated bool
	// FetchError stores a fetch rejection/abort reason.
	FetchError error
	// FetchAborted indicates that the fetch received an abort signal.
	FetchAborted bool
	// FetchAbortIgnored indicates that the abort signal was ignored.
	FetchAbortIgnored bool
	// FetchResolved indicates that the fetch resolved successfully.
	FetchResolved bool
	// FetchRejected indicates that the fetch rejected with an error.
	FetchRejected bool
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

	// FetchMethod implements Fetch/ForceFetch behavior.
	// staleValue is nil on a cache miss.
	FetchMethod func(key K, staleValue *V, opts FetcherOptions[K, V]) (value V, ok bool, err error)

	// MemoMethod implements Memo behavior.
	// staleValue is nil on a cache miss.
	MemoMethod func(key K, staleValue *V, opts MemoizerOptions[K, V]) V

	// NoDeleteOnFetchRejection preserves stale data when FetchMethod returns an error.
	NoDeleteOnFetchRejection bool

	// AllowStaleOnFetchRejection returns stale data when FetchMethod returns an error.
	AllowStaleOnFetchRejection bool

	// AllowStaleOnFetchAbort returns stale data when an inflight fetch is aborted.
	AllowStaleOnFetchAbort bool

	// IgnoreFetchAbort allows a fetch to continue updating the cache after abort.
	IgnoreFetchAbort bool

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

// FetchOptions overrides cache-level options for a single Fetch()/ForceFetch() call.
type FetchOptions[K comparable, V any] struct {
	AllowStale                 *bool
	UpdateAgeOnGet             *bool
	NoDeleteOnStaleGet         *bool
	TTL                        *int64
	NoDisposeOnSet             *bool
	NoUpdateTTL                *bool
	SizeCalculation            func(value V, key K) int
	Size                       int
	NoDeleteOnFetchRejection   *bool
	AllowStaleOnFetchRejection *bool
	AllowStaleOnFetchAbort     *bool
	IgnoreFetchAbort           *bool
	ForceRefresh               bool
	Context                    any
	Signal                     context.Context
	Status                     *Status[V]
}

// MemoOptions overrides cache-level options for a single Memo() call.
type MemoOptions[K comparable, V any] struct {
	AllowStale         *bool
	UpdateAgeOnGet     *bool
	NoDeleteOnStaleGet *bool
	TTL                *int64
	NoDisposeOnSet     *bool
	NoUpdateTTL        *bool
	SizeCalculation    func(value V, key K) int
	Size               int
	ForceRefresh       bool
	Context            any
	Status             *Status[V]
}

// ResolvedFetchOptions are passed to FetchMethod after cache defaults are applied.
type ResolvedFetchOptions[K comparable, V any] struct {
	AllowStale                 bool
	UpdateAgeOnGet             bool
	NoDeleteOnStaleGet         bool
	TTL                        int64
	NoDisposeOnSet             bool
	NoUpdateTTL                bool
	SizeCalculation            func(value V, key K) int
	Size                       int
	NoDeleteOnFetchRejection   bool
	AllowStaleOnFetchRejection bool
	AllowStaleOnFetchAbort     bool
	IgnoreFetchAbort           bool
	Status                     *Status[V]
	Signal                     context.Context
}

// ResolvedMemoOptions are passed to MemoMethod after cache defaults are applied.
type ResolvedMemoOptions[K comparable, V any] struct {
	AllowStale         bool
	UpdateAgeOnGet     bool
	NoDeleteOnStaleGet bool
	TTL                int64
	NoDisposeOnSet     bool
	NoUpdateTTL        bool
	SizeCalculation    func(value V, key K) int
	Size               int
	Start              int64
	Status             *Status[V]
}

// FetcherOptions are passed to Options.FetchMethod.
type FetcherOptions[K comparable, V any] struct {
	Signal  context.Context
	Options *ResolvedFetchOptions[K, V]
	Context any
}

// MemoizerOptions are passed to Options.MemoMethod.
type MemoizerOptions[K comparable, V any] struct {
	Options *ResolvedMemoOptions[K, V]
	Context any
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

type cacheValue[V any] struct {
	value *V
	fetch *backgroundFetch[V]
}

type fetchResult[V any] struct {
	value *V
	err   error
}

type backgroundFetch[V any] struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	stale *V

	mu        sync.Mutex
	done      chan struct{}
	completed bool
	result    fetchResult[V]
}

func newValueSlot[V any](v V) *cacheValue[V] {
	vv := v
	return &cacheValue[V]{value: &vv}
}

func newFetchSlot[V any](bf *backgroundFetch[V]) *cacheValue[V] {
	return &cacheValue[V]{fetch: bf}
}

func (cv *cacheValue[V]) isBackgroundFetch() bool {
	return cv != nil && cv.fetch != nil
}

func (cv *cacheValue[V]) visibleValue() *V {
	if cv == nil {
		return nil
	}
	if cv.fetch != nil {
		return cv.fetch.stale
	}
	return cv.value
}

func (cv *cacheValue[V]) actualValue() *V {
	if cv == nil || cv.fetch != nil {
		return nil
	}
	return cv.value
}

func newBackgroundFetch[V any](ctx context.Context, cancel context.CancelCauseFunc, stale *V) *backgroundFetch[V] {
	return &backgroundFetch[V]{
		ctx:    ctx,
		cancel: cancel,
		stale:  stale,
		done:   make(chan struct{}),
	}
}

func (bf *backgroundFetch[V]) complete(value *V, err error) bool {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	if bf.completed {
		return false
	}
	bf.completed = true
	bf.result = fetchResult[V]{value: value, err: err}
	close(bf.done)
	return true
}

func (bf *backgroundFetch[V]) wait() fetchResult[V] {
	<-bf.done
	bf.mu.Lock()
	defer bf.mu.Unlock()
	return bf.result
}

func clonePtr[V any](v V) *V {
	vv := v
	return &vv
}

var warned sync.Map

func warnOnce(code, warningType, msg string) {
	if _, loaded := warned.LoadOrStore(code, struct{}{}); loaded {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", code, warningType, msg)
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

	max                        int   // Maximum item count (0 = unbounded by count)
	maxSize                    int   // Maximum total size (0 = no size tracking)
	maxEntrySize               int   // Maximum single entry size (0 = no limit)
	ttl                        int64 // Default TTL in ms (0 = no TTL)
	ttlResolution              int64 // Time caching resolution in ms
	ttlAutopurge               bool  // Auto-delete expired items via timers
	updateAgeOnGet             bool
	updateAgeOnHas             bool
	allowStale                 bool
	noDisposeOnSet             bool
	noUpdateTTL                bool
	noDeleteOnFetchRejection   bool
	allowStaleOnFetchAbort     bool
	allowStaleOnFetchRejection bool
	ignoreFetchAbort           bool
	noDeleteOnStaleGet         bool
	sizeCalculation            func(V, K) int

	// --- Callbacks ---
	dispose      func(V, K, DisposeReason)
	disposeAfter func(V, K, DisposeReason)
	onInsert     func(V, K, InsertReason)
	fetchMethod  func(K, *V, FetcherOptions[K, V]) (V, bool, error)
	memoMethod   func(K, *V, MemoizerOptions[K, V]) V

	// --- Feature flags (cached from callbacks for fast checks) ---
	hasDispose      bool
	hasDisposeAfter bool
	hasOnInsert     bool
	hasFetchMethod  bool

	// --- Core data structure ---
	// Parallel arrays + index-based doubly-linked list.
	// TS source: constructor initialization (lines ~1410-1470)
	//
	// The linked list goes: head (LRU) → ... → tail (MRU)
	// next[i] = index of next more-recently-used item
	// prev[i] = index of next less-recently-used item
	keyMap  map[K]int        // key → array index (TS: #keyMap)
	keyList []*K             // index → key pointer, nil = empty slot (TS: #keyList)
	valList []*cacheValue[V] // index → value/background fetch, nil = empty slot
	next    []int            // forward linked list pointers (TS: #next)
	prev    []int            // backward linked list pointers (TS: #prev)
	head    int              // LRU end index (TS: #head)
	tail    int              // MRU end index (TS: #tail)
	free    []int            // stack of freed indices for reuse (TS: #free via Stack)
	size    int              // current item count (TS: #size)

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
		panic("cannot set sizeCalculation without setting maxSize or maxEntrySize")
	}

	// Must have at least one limiting factor
	// TS source: lines ~1395-1405
	if max == 0 && o.MaxSize == 0 && o.TTL == 0 {
		panic("at least one of max, maxSize, or ttl must be specified")
	}

	// TTL resolution defaults to 1
	// TS source: line ~1480
	ttlResolution := o.TTLResolution
	if ttlResolution == 0 {
		ttlResolution = 1
	}
	if ttlResolution < 0 {
		ttlResolution = 1
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
		max:                        max,
		maxSize:                    o.MaxSize,
		maxEntrySize:               maxEntrySize,
		ttl:                        o.TTL,
		ttlResolution:              ttlResolution,
		ttlAutopurge:               o.TTLAutopurge,
		updateAgeOnGet:             o.UpdateAgeOnGet,
		updateAgeOnHas:             o.UpdateAgeOnHas,
		allowStale:                 o.AllowStale,
		noDisposeOnSet:             o.NoDisposeOnSet,
		noUpdateTTL:                o.NoUpdateTTL,
		noDeleteOnFetchRejection:   o.NoDeleteOnFetchRejection,
		allowStaleOnFetchAbort:     o.AllowStaleOnFetchAbort,
		allowStaleOnFetchRejection: o.AllowStaleOnFetchRejection,
		ignoreFetchAbort:           o.IgnoreFetchAbort,
		noDeleteOnStaleGet:         o.NoDeleteOnStaleGet,
		sizeCalculation:            o.SizeCalculation,
		dispose:                    o.Dispose,
		disposeAfter:               o.DisposeAfter,
		onInsert:                   o.OnInsert,
		fetchMethod:                o.FetchMethod,
		memoMethod:                 o.MemoMethod,
		hasDispose:                 o.Dispose != nil,
		hasDisposeAfter:            o.DisposeAfter != nil,
		hasOnInsert:                o.OnInsert != nil,
		hasFetchMethod:             o.FetchMethod != nil,
		nowFn:                      nowFn,
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
	c.valList = make([]*cacheValue[V], cap)
	c.next = make([]int, cap)
	c.prev = make([]int, cap)
	c.free = make([]int, 0, cap)

	// Initialize TTL tracking if TTL is set
	// TS source: lines ~1500-1510
	if o.TTL > 0 {
		c.initializeTTLTracking()
	}

	// Initialize size tracking if size limits are set
	// TS source: lines ~1515-1520
	if maxEntrySize > 0 {
		c.initializeSizeTracking()
	}

	if !o.TTLAutopurge && max == 0 && o.MaxSize == 0 && o.TTL > 0 {
		warnOnce(
			"LRU_CACHE_UNBOUNDED",
			"UnboundedCacheWarning",
			"TTL caching without ttlAutopurge, max, or maxSize can result in unbounded memory consumption.",
		)
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
	c.valList = growSliceCacheValue[V](c.valList, newCap)
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

func growSliceCacheValue[V any](s []*cacheValue[V], newCap int) []*cacheValue[V] {
	grown := make([]*cacheValue[V], newCap)
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
//
// The TS version uses setTimeout to clear cachedNow after ttlResolution ms.
// In Go, time.AfterFunc uses real wall-clock time which doesn't work with
// mock clocks in tests. Instead, we use comparison-based invalidation:
// call nowFn() and check if the clock has advanced past the resolution period.
// This is semantically equivalent and works correctly with both real and mock clocks.
func (c *LRUCache[K, V]) getNow() int64 {
	if c.ttlResolution > 0 {
		cached := c.cachedNow.Load()
		if cached > 0 {
			// Check if the resolution period has passed by asking the clock.
			// TS equivalent: setTimeout(() => this.#cachedNow = 0, ttlResolution)
			now := c.nowFn()
			if now-cached < c.ttlResolution {
				return cached
			}
			// Resolution period elapsed, update cache with fresh value.
			c.cachedNow.Store(now)
			return now
		}
	}
	now := c.nowFn()
	if c.ttlResolution > 0 {
		c.cachedNow.Store(now)
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

	// Call dispose callbacks or abort an inflight fetch.
	if k != nil && v != nil {
		if v.isBackgroundFetch() {
			v.fetch.cancel(fmt.Errorf("evicted"))
		} else if c.hasDispose || c.hasDisposeAfter {
			value := v.actualValue()
			if value == nil {
				goto evictAfterCallbacks
			}
			if c.hasDispose {
				c.dispose(*value, *k, DisposeEvict)
			}
			if c.hasDisposeAfter {
				c.disposed = append(c.disposed, disposeTask[K, V]{
					value: *value, key: *k, reason: DisposeEvict,
				})
			}
		}
	}

evictAfterCallbacks:
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

	// Call dispose or abort an inflight fetch.
	v := c.valList[index]
	if v != nil {
		if v.isBackgroundFetch() {
			v.fetch.cancel(fmt.Errorf("deleted"))
		} else if c.hasDispose || c.hasDisposeAfter {
			value := v.actualValue()
			if value == nil {
				goto deleteAfterCallbacks
			}
			if c.hasDispose {
				c.dispose(*value, k, reason)
			}
			if c.hasDisposeAfter {
				c.disposed = append(c.disposed, disposeTask[K, V]{
					value: *value, key: k, reason: reason,
				})
			}
		}
	}

deleteAfterCallbacks:
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
			if v.isBackgroundFetch() {
				v.fetch.cancel(fmt.Errorf("deleted"))
			} else {
				value := v.actualValue()
				if value == nil {
					return true
				}
				if c.hasDispose {
					c.dispose(*value, *k, reason)
				}
				if c.hasDisposeAfter {
					c.disposed = append(c.disposed, disposeTask[K, V]{
						value: *value, key: *k, reason: reason,
					})
				}
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

func (c *LRUCache[K, V]) resolveFetchOptions(opt FetchOptions[K, V]) ResolvedFetchOptions[K, V] {
	resolved := ResolvedFetchOptions[K, V]{
		AllowStale:                 c.allowStale,
		UpdateAgeOnGet:             c.updateAgeOnGet,
		NoDeleteOnStaleGet:         c.noDeleteOnStaleGet,
		TTL:                        c.ttl,
		NoDisposeOnSet:             c.noDisposeOnSet,
		NoUpdateTTL:                c.noUpdateTTL,
		SizeCalculation:            c.sizeCalculation,
		NoDeleteOnFetchRejection:   c.noDeleteOnFetchRejection,
		AllowStaleOnFetchRejection: c.allowStaleOnFetchRejection,
		AllowStaleOnFetchAbort:     c.allowStaleOnFetchAbort,
		IgnoreFetchAbort:           c.ignoreFetchAbort,
		Status:                     opt.Status,
		Signal:                     opt.Signal,
	}
	if opt.AllowStale != nil {
		resolved.AllowStale = *opt.AllowStale
	}
	if opt.UpdateAgeOnGet != nil {
		resolved.UpdateAgeOnGet = *opt.UpdateAgeOnGet
	}
	if opt.NoDeleteOnStaleGet != nil {
		resolved.NoDeleteOnStaleGet = *opt.NoDeleteOnStaleGet
	}
	if opt.TTL != nil {
		resolved.TTL = *opt.TTL
	}
	if opt.NoDisposeOnSet != nil {
		resolved.NoDisposeOnSet = *opt.NoDisposeOnSet
	}
	if opt.NoUpdateTTL != nil {
		resolved.NoUpdateTTL = *opt.NoUpdateTTL
	}
	if opt.SizeCalculation != nil {
		resolved.SizeCalculation = opt.SizeCalculation
	}
	if opt.Size > 0 {
		resolved.Size = opt.Size
	}
	if opt.NoDeleteOnFetchRejection != nil {
		resolved.NoDeleteOnFetchRejection = *opt.NoDeleteOnFetchRejection
	}
	if opt.AllowStaleOnFetchRejection != nil {
		resolved.AllowStaleOnFetchRejection = *opt.AllowStaleOnFetchRejection
	}
	if opt.AllowStaleOnFetchAbort != nil {
		resolved.AllowStaleOnFetchAbort = *opt.AllowStaleOnFetchAbort
	}
	if opt.IgnoreFetchAbort != nil {
		resolved.IgnoreFetchAbort = *opt.IgnoreFetchAbort
	}
	return resolved
}

func (c *LRUCache[K, V]) resolveMemoOptions(opt MemoOptions[K, V]) ResolvedMemoOptions[K, V] {
	resolved := ResolvedMemoOptions[K, V]{
		AllowStale:         c.allowStale,
		UpdateAgeOnGet:     c.updateAgeOnGet,
		NoDeleteOnStaleGet: c.noDeleteOnStaleGet,
		TTL:                c.ttl,
		NoDisposeOnSet:     c.noDisposeOnSet,
		NoUpdateTTL:        c.noUpdateTTL,
		SizeCalculation:    c.sizeCalculation,
		Status:             opt.Status,
	}
	if opt.AllowStale != nil {
		resolved.AllowStale = *opt.AllowStale
	}
	if opt.UpdateAgeOnGet != nil {
		resolved.UpdateAgeOnGet = *opt.UpdateAgeOnGet
	}
	if opt.NoDeleteOnStaleGet != nil {
		resolved.NoDeleteOnStaleGet = *opt.NoDeleteOnStaleGet
	}
	if opt.TTL != nil {
		resolved.TTL = *opt.TTL
	}
	if opt.NoDisposeOnSet != nil {
		resolved.NoDisposeOnSet = *opt.NoDisposeOnSet
	}
	if opt.NoUpdateTTL != nil {
		resolved.NoUpdateTTL = *opt.NoUpdateTTL
	}
	if opt.SizeCalculation != nil {
		resolved.SizeCalculation = opt.SizeCalculation
	}
	if opt.Size > 0 {
		resolved.Size = opt.Size
	}
	return resolved
}

func (c *LRUCache[K, V]) allocIndexLocked() int {
	if c.size == 0 {
		return c.tail
	}
	if len(c.free) > 0 {
		index := c.free[len(c.free)-1]
		c.free = c.free[:len(c.free)-1]
		return index
	}
	if c.max > 0 && c.size == c.max {
		return c.evict(false)
	}
	return c.size
}

func (c *LRUCache[K, V]) setFetchPlaceholderLocked(
	k K,
	index int,
	exists bool,
	bf *backgroundFetch[V],
	options ResolvedFetchOptions[K, V],
) int {
	if !exists {
		index = c.allocIndexLocked()
		c.ensureIndex(index)
		kCopy := k
		c.keyList[index] = &kCopy
		c.valList[index] = newFetchSlot(bf)
		c.keyMap[k] = index
		c.next[c.tail] = index
		c.prev[index] = c.tail
		c.tail = index
		c.size++
		c.addItemSize(index, 0, nil)
		if options.TTL != 0 && c.ttls == nil {
			c.initializeTTLTracking()
		}
		if c.ttls != nil {
			c.setItemTTL(index, options.TTL, 0)
		}
		return index
	}
	c.valList[index] = newFetchSlot(bf)
	return index
}

func (c *LRUCache[K, V]) fetchSetOptions(options *ResolvedFetchOptions[K, V]) SetOptions[K, V] {
	ttl := options.TTL
	noDisposeOnSet := options.NoDisposeOnSet
	noUpdateTTL := options.NoUpdateTTL
	return SetOptions[K, V]{
		TTL:             &ttl,
		NoDisposeOnSet:  &noDisposeOnSet,
		NoUpdateTTL:     &noUpdateTTL,
		SizeCalculation: options.SizeCalculation,
		Size:            options.Size,
		Status:          options.Status,
	}
}

func (c *LRUCache[K, V]) handleFetchFailure(
	k K,
	index int,
	bf *backgroundFetch[V],
	options *ResolvedFetchOptions[K, V],
	err error,
	proceed bool,
) fetchResult[V] {
	aborted := bf.ctx.Err() != nil
	allowStaleAborted := aborted && options.AllowStaleOnFetchAbort
	allowStale := allowStaleAborted || options.AllowStaleOnFetchRejection
	noDelete := allowStale || options.NoDeleteOnFetchRejection

	c.mu.Lock()
	if index >= 0 && index < len(c.valList) {
		slot := c.valList[index]
		if slot != nil && slot.fetch == bf {
			del := !noDelete || (!proceed && bf.stale == nil)
			if del {
				c.internalDelete(k, DisposeFetch)
			} else if !allowStaleAborted && bf.stale != nil {
				c.valList[index] = newValueSlot(*bf.stale)
			}
		}
	}
	tasks := c.drainDisposed()
	c.mu.Unlock()
	c.runDisposeTasks(tasks)

	if allowStale {
		if options.Status != nil && bf.stale != nil {
			options.Status.ReturnedStale = true
		}
		return fetchResult[V]{value: bf.stale}
	}
	return fetchResult[V]{err: err}
}

func (c *LRUCache[K, V]) handleFetchSuccess(
	k K,
	index int,
	bf *backgroundFetch[V],
	options *ResolvedFetchOptions[K, V],
	value *V,
	updateCache bool,
) fetchResult[V] {
	aborted := bf.ctx.Err() != nil
	ignoreAbort := options.IgnoreFetchAbort && value != nil
	proceed := options.IgnoreFetchAbort || (options.AllowStaleOnFetchAbort && bf.stale != nil)

	if options.Status != nil {
		if aborted && !updateCache {
			options.Status.FetchAborted = true
			options.Status.FetchError = context.Cause(bf.ctx)
			if ignoreAbort {
				options.Status.FetchAbortIgnored = true
			}
		} else {
			options.Status.FetchResolved = true
		}
	}

	if aborted && !ignoreAbort && !updateCache {
		return c.handleFetchFailure(k, index, bf, options, context.Cause(bf.ctx), proceed)
	}

	shouldSet := false
	c.mu.Lock()
	if index >= 0 && index < len(c.valList) {
		slot := c.valList[index]
		if slot != nil && slot.fetch == bf {
			if value == nil {
				if bf.stale != nil {
					c.valList[index] = newValueSlot(*bf.stale)
				} else {
					c.internalDelete(k, DisposeFetch)
				}
			} else {
				shouldSet = true
			}
		} else if ignoreAbort && updateCache && slot == nil && value != nil {
			shouldSet = true
		}
	}
	tasks := c.drainDisposed()
	c.mu.Unlock()
	c.runDisposeTasks(tasks)

	if shouldSet && value != nil {
		if options.Status != nil {
			options.Status.FetchUpdated = true
		}
		c.Set(k, *value, c.fetchSetOptions(options))
	}

	return fetchResult[V]{value: value}
}

func (c *LRUCache[K, V]) startBackgroundFetch(
	k K,
	index int,
	exists bool,
	options ResolvedFetchOptions[K, V],
	userContext any,
) *backgroundFetch[V] {
	var stale *V
	if exists && index >= 0 && index < len(c.valList) && c.valList[index] != nil {
		if visible := c.valList[index].visibleValue(); visible != nil {
			stale = clonePtr(*visible)
		}
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	bf := newBackgroundFetch(ctx, cancel, stale)
	index = c.setFetchPlaceholderLocked(k, index, exists, bf, options)

	stopSignal := func() bool { return false }
	if options.Signal != nil {
		stopSignal = context.AfterFunc(options.Signal, func() {
			cancel(context.Cause(options.Signal))
		})
	}

	type outcome struct {
		value *V
		err   error
	}
	outcomeCh := make(chan outcome, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				outcomeCh <- outcome{err: fmt.Errorf("%v", r)}
			}
		}()
		value, ok, err := c.fetchMethod(k, stale, FetcherOptions[K, V]{
			Signal:  ctx,
			Options: &options,
			Context: userContext,
		})
		if ok {
			outcomeCh <- outcome{value: clonePtr(value)}
			return
		}
		outcomeCh <- outcome{err: err}
	}()

	if options.Status != nil {
		options.Status.FetchDispatched = true
	}

	go func() {
		defer stopSignal()

		abortedEarly := false
		updateCache := false
		abortCh := ctx.Done()

		for {
			select {
			case out := <-outcomeCh:
				var result fetchResult[V]
				if out.err != nil {
					if options.Status != nil {
						options.Status.FetchRejected = true
						options.Status.FetchError = out.err
					}
					result = c.handleFetchFailure(k, index, bf, &options, out.err, false)
				} else {
					result = c.handleFetchSuccess(k, index, bf, &options, out.value, updateCache)
				}
				if !abortedEarly {
					bf.complete(result.value, result.err)
				}
				return
			case <-abortCh:
				abortCh = nil
				if !options.IgnoreFetchAbort || options.AllowStaleOnFetchAbort {
					if options.Status != nil {
						options.Status.FetchAborted = true
						options.Status.FetchError = context.Cause(ctx)
					}
					proceed := options.IgnoreFetchAbort || (options.AllowStaleOnFetchAbort && bf.stale != nil)
					result := c.handleFetchFailure(k, index, bf, &options, context.Cause(ctx), proceed)
					bf.complete(result.value, result.err)
					abortedEarly = true
					updateCache = options.AllowStaleOnFetchAbort
					if !options.AllowStaleOnFetchAbort {
						return
					}
				}
			}
		}
	}()

	return bf
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
		c.keyList[index] = &kCopy
		c.valList[index] = newValueSlot(v)
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
		oldActual := oldVal.actualValue()
		same := oldActual != nil && sameValue(v, *oldActual)

		if !same {
			// Value changed: dispose old, store new
			// TS source: lines ~2132-2162
			if !noDisposeOnSet && oldVal != nil {
				if oldVal.isBackgroundFetch() {
					oldVal.fetch.cancel(fmt.Errorf("replaced"))
					if stale := oldVal.fetch.stale; stale != nil {
						if c.hasDispose {
							c.dispose(*stale, k, DisposeSet)
						}
						if c.hasDisposeAfter {
							c.disposed = append(c.disposed, disposeTask[K, V]{
								value: *stale, key: k, reason: DisposeSet,
							})
						}
					}
				} else if oldActual != nil {
					if c.hasDispose {
						c.dispose(*oldActual, k, DisposeSet)
					}
					if c.hasDisposeAfter {
						c.disposed = append(c.disposed, disposeTask[K, V]{
							value: *oldActual, key: k, reason: DisposeSet,
						})
					}
				}
			}
			c.removeItemSize(index)
			c.addItemSize(index, size, status)
			c.valList[index] = newValueSlot(v)
			if status != nil {
				status.Set = "replace"
				if oldVisible := oldVal.visibleValue(); oldVisible != nil {
					status.OldValue = clonePtr(*oldVisible)
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

// Fetch returns a cached value or loads it via FetchMethod.
// It matches the node-lru-cache fetch() behavior as closely as Go allows:
// a non-nil error represents a rejected fetch promise, while ok=false/error=nil
// represents a resolved undefined fetch.
func (c *LRUCache[K, V]) Fetch(k K, opts ...FetchOptions[K, V]) (V, bool, error) {
	var raw FetchOptions[K, V]
	if len(opts) > 0 {
		raw = opts[0]
	}
	resolved := c.resolveFetchOptions(raw)

	if !c.hasFetchMethod {
		if resolved.Status != nil {
			resolved.Status.Fetch = "get"
		}
		allowStale := resolved.AllowStale
		updateAgeOnGet := resolved.UpdateAgeOnGet
		noDeleteOnStaleGet := resolved.NoDeleteOnStaleGet
		v, ok := c.Get(k, GetOptions[V]{
			AllowStale:         &allowStale,
			UpdateAgeOnGet:     &updateAgeOnGet,
			NoDeleteOnStaleGet: &noDeleteOnStaleGet,
			Status:             resolved.Status,
		})
		return v, ok, nil
	}

	c.mu.Lock()
	index, exists := c.keyMap[k]
	if !exists {
		if resolved.Status != nil {
			resolved.Status.Fetch = "miss"
		}
		bf := c.startBackgroundFetch(k, -1, false, resolved, raw.Context)
		c.mu.Unlock()
		result := bf.wait()
		var zero V
		if result.err != nil {
			return zero, false, result.err
		}
		if result.value == nil {
			return zero, false, nil
		}
		return *result.value, true, nil
	}

	slot := c.valList[index]
	if slot != nil && slot.isBackgroundFetch() {
		stale := resolved.AllowStale && slot.visibleValue() != nil
		if resolved.Status != nil {
			resolved.Status.Fetch = "inflight"
			if stale {
				resolved.Status.ReturnedStale = true
			}
		}
		if stale {
			value := slot.visibleValue()
			c.mu.Unlock()
			return *value, true, nil
		}
		bf := slot.fetch
		c.mu.Unlock()
		result := bf.wait()
		var zero V
		if result.err != nil {
			return zero, false, result.err
		}
		if result.value == nil {
			return zero, false, nil
		}
		return *result.value, true, nil
	}

	isStale := c.isStale(index)
	if !raw.ForceRefresh && !isStale {
		if resolved.Status != nil {
			resolved.Status.Fetch = "hit"
		}
		c.moveToTail(index)
		if resolved.UpdateAgeOnGet {
			c.updateItemAge(index)
		}
		if resolved.Status != nil {
			c.statusTTL(resolved.Status, index)
		}
		value := slot.actualValue()
		c.mu.Unlock()
		if value == nil {
			var zero V
			return zero, false, nil
		}
		return *value, true, nil
	}

	bf := c.startBackgroundFetch(k, index, true, resolved, raw.Context)
	stale := bf.stale != nil && resolved.AllowStale
	if resolved.Status != nil {
		if isStale {
			resolved.Status.Fetch = "stale"
			if stale {
				resolved.Status.ReturnedStale = true
			}
		} else {
			resolved.Status.Fetch = "refresh"
		}
	}
	if stale {
		value := *bf.stale
		c.mu.Unlock()
		return value, true, nil
	}
	c.mu.Unlock()

	result := bf.wait()
	var zero V
	if result.err != nil {
		return zero, false, result.err
	}
	if result.value == nil {
		return zero, false, nil
	}
	return *result.value, true, nil
}

// ForceFetch is Fetch, but it rejects undefined resolutions.
func (c *LRUCache[K, V]) ForceFetch(k K, opts ...FetchOptions[K, V]) (V, error) {
	v, ok, err := c.Fetch(k, opts...)
	if err != nil {
		var zero V
		return zero, err
	}
	if !ok {
		var zero V
		return zero, fmt.Errorf("fetch() returned undefined")
	}
	return v, nil
}

// Memo returns a cached value or computes it via MemoMethod.
func (c *LRUCache[K, V]) Memo(k K, opts ...MemoOptions[K, V]) V {
	if c.memoMethod == nil {
		panic("no memoMethod provided to constructor")
	}

	var raw MemoOptions[K, V]
	if len(opts) > 0 {
		raw = opts[0]
	}
	resolved := c.resolveMemoOptions(raw)

	allowStale := resolved.AllowStale
	updateAgeOnGet := resolved.UpdateAgeOnGet
	noDeleteOnStaleGet := resolved.NoDeleteOnStaleGet
	v, ok := c.Get(k, GetOptions[V]{
		AllowStale:         &allowStale,
		UpdateAgeOnGet:     &updateAgeOnGet,
		NoDeleteOnStaleGet: &noDeleteOnStaleGet,
		Status:             resolved.Status,
	})
	if !raw.ForceRefresh && ok {
		return v
	}

	var stale *V
	if ok {
		stale = clonePtr(v)
	}
	value := c.memoMethod(k, stale, MemoizerOptions[K, V]{
		Options: &resolved,
		Context: raw.Context,
	})
	ttl := resolved.TTL
	noDisposeOnSet := resolved.NoDisposeOnSet
	noUpdateTTL := resolved.NoUpdateTTL
	c.Set(k, value, SetOptions[K, V]{
		TTL:             &ttl,
		NoDisposeOnSet:  &noDisposeOnSet,
		NoUpdateTTL:     &noUpdateTTL,
		SizeCalculation: resolved.SizeCalculation,
		Size:            resolved.Size,
		Status:          resolved.Status,
	})
	return value
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
	fetching := vp.isBackgroundFetch()

	if status != nil {
		c.statusTTL(status, index)
	}

	if c.isStale(index) {
		// TS source: lines ~2776-2794
		if status != nil {
			status.Get = "stale"
		}
		if !fetching && !noDeleteOnStaleGet {
			c.internalDelete(k, DisposeExpire)
		}
		tasks := c.drainDisposed()
		c.mu.Unlock()
		c.runDisposeTasks(tasks)
		if allowStale {
			value := vp.visibleValue()
			if value == nil {
				return zero, false
			}
			if status != nil {
				status.ReturnedStale = true
			}
			return *value, true
		}
		return zero, false
	}

	// Not stale
	// TS source: lines ~2795-2810
	if status != nil {
		status.Get = "hit"
	}
	if fetching {
		value := vp.visibleValue()
		c.mu.Unlock()
		if value == nil {
			return zero, false
		}
		return *value, true
	}
	c.moveToTail(index)
	if updateAgeOnGet {
		c.updateItemAge(index)
	}
	value := vp.actualValue()
	c.mu.Unlock()
	if value == nil {
		return zero, false
	}
	return *value, true
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
	if c.valList[index].isBackgroundFetch() && c.valList[index].visibleValue() == nil {
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
	value := v.visibleValue()
	if value == nil {
		return zero, false
	}
	return *value, true
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
		if val != nil {
			value := val.visibleValue()
			if value == nil {
				continue
			}
			result = *value
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
		value := v.visibleValue()
		if value == nil {
			return true
		}
		k := c.keyList[index]
		if k == nil {
			return true
		}
		if fn(*value, *k) {
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
			value := v.visibleValue()
			if value == nil {
				return true
			}
			fn(*value, *k)
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
			value := v.visibleValue()
			if value == nil {
				return true
			}
			fn(*value, *k)
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
	value := v.visibleValue()
	if value == nil {
		return nil
	}

	entry := &Entry[V]{Value: *value}

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
		if v != nil && k != nil && !v.isBackgroundFetch() {
			value := v.actualValue()
			if value != nil {
				result = append(result, [2]any{*k, *value})
			}
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
		if k != nil && c.valList[index] != nil && !c.valList[index].isBackgroundFetch() {
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
		if v != nil && c.keyList[index] != nil && !v.isBackgroundFetch() {
			value := v.actualValue()
			if value != nil {
				result = append(result, *value)
			}
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
		if v != nil && k != nil && !v.isBackgroundFetch() {
			value := v.actualValue()
			if value != nil {
				result = append(result, [2]any{*k, *value})
			}
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
		if k != nil && c.valList[index] != nil && !c.valList[index].isBackgroundFetch() {
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
		if v != nil && c.keyList[index] != nil && !v.isBackgroundFetch() {
			value := v.actualValue()
			if value != nil {
				result = append(result, *value)
			}
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
		value := v.visibleValue()
		if value == nil {
			return true
		}

		entry := Entry[V]{Value: *value}

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

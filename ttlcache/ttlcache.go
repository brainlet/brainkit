package ttlcache

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"sync"
	"time"
)

const (
	timerMaxMs         = int64(1<<31 - 1)
	infinityExpiration = int64(math.MaxInt64)
)

var (
	ErrMaxMustBePositive      = errors.New("max must be positive integer or Infinity")
	ErrTTLMustBePositive      = errors.New("ttl must be positive integer or Infinity")
	ErrTTLMustBePositiveIfSet = errors.New("ttl must be positive integer or Infinity if set")
)

type DisposeReason string

const (
	DisposeSet    DisposeReason = "set"
	DisposeDelete DisposeReason = "delete"
	DisposeStale  DisposeReason = "stale"
	DisposeEvict  DisposeReason = "evict"
)

type DisposeFunction[K comparable, V any] func(value V, key K, reason DisposeReason)

type TTLCacheOptions[K comparable, V any] struct {
	Max            *int
	TTL            *time.Duration
	UpdateAgeOnGet bool
	CheckAgeOnGet  bool
	UpdateAgeOnHas bool
	CheckAgeOnHas  bool
	NoUpdateTTL    bool
	Dispose        DisposeFunction[K, V]
	NoDisposeOnSet bool
}

type SetOptions[K comparable, V any] struct {
	TTL            *time.Duration
	NoUpdateTTL    *bool
	NoDisposeOnSet *bool
}

type GetOptions[K comparable, V any] struct {
	UpdateAgeOnGet *bool
	TTL            *time.Duration
	CheckAgeOnGet  *bool
}

type HasOptions[K comparable, V any] struct {
	UpdateAgeOnHas *bool
	TTL            *time.Duration
	CheckAgeOnHas  *bool
}

type Entry[K comparable, V any] struct {
	Key   K
	Value V
}

type timerHandle interface {
	Stop() bool
}

type disposal[K comparable, V any] struct {
	key    K
	value  V
	reason DisposeReason
}

// TTLCache is a Go port of @isaacs/ttlcache.
//
// Go-specific notes:
//   - nil TTL means "unset default TTL", matching JS undefined.
//   - a zero time.Duration means Infinity when provided explicitly.
//   - timers are synchronized because Go timers fire in separate goroutines.
type TTLCache[K comparable, V any] struct {
	mu sync.Mutex

	expirations     map[int64][]K
	expirationOrder []int64
	data            map[K]V
	expirationMap   map[K]int64

	ttl            *time.Duration
	max            int
	updateAgeOnGet bool
	updateAgeOnHas bool
	noUpdateTTL    bool
	noDisposeOnSet bool
	checkAgeOnGet  bool
	checkAgeOnHas  bool
	dispose        DisposeFunction[K, V]
	hasDispose     bool

	timer              timerHandle
	timerExpiration    int64
	hasTimerExpiration bool

	immortalKeys  map[K]struct{}
	immortalOrder []K

	nowFn     func() float64
	afterFunc func(time.Duration, func()) timerHandle
}

func New[K comparable, V any](opts ...TTLCacheOptions[K, V]) (*TTLCache[K, V], error) {
	var o TTLCacheOptions[K, V]
	if len(opts) > 0 {
		o = opts[0]
	}

	max := 0
	if o.Max != nil {
		if *o.Max <= 0 {
			return nil, ErrMaxMustBePositive
		}
		max = *o.Max
	}

	if o.TTL != nil {
		if err := validatePositiveTTL(*o.TTL, ErrTTLMustBePositiveIfSet); err != nil {
			return nil, err
		}
	}

	dispose := o.Dispose
	if dispose == nil {
		dispose = func(V, K, DisposeReason) {}
	}

	return &TTLCache[K, V]{
		expirations:    make(map[int64][]K),
		data:           make(map[K]V),
		expirationMap:  make(map[K]int64),
		ttl:            o.TTL,
		max:            max,
		updateAgeOnGet: o.UpdateAgeOnGet,
		updateAgeOnHas: o.UpdateAgeOnHas,
		noUpdateTTL:    o.NoUpdateTTL,
		noDisposeOnSet: o.NoDisposeOnSet,
		checkAgeOnGet:  o.CheckAgeOnGet,
		checkAgeOnHas:  o.CheckAgeOnHas,
		dispose:        dispose,
		hasDispose:     o.Dispose != nil,
		immortalKeys:   make(map[K]struct{}),
		nowFn:          defaultNow,
		afterFunc:      realAfterFunc,
	}, nil
}

func defaultNow() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Millisecond)
}

func realAfterFunc(delay time.Duration, fn func()) timerHandle {
	return time.AfterFunc(delay, fn)
}

func validatePositiveTTL(ttl time.Duration, err error) error {
	if ttl == 0 {
		return nil
	}
	if ttl < time.Millisecond {
		return err
	}
	if ttl%time.Millisecond != 0 {
		return err
	}
	return nil
}

func ttlMillis(ttl time.Duration) int64 {
	return int64(ttl / time.Millisecond)
}

func valuesDiffer[V any](a, b V) bool {
	av := any(a)
	bv := any(b)
	at := reflect.TypeOf(av)
	bt := reflect.TypeOf(bv)

	if at == nil || bt == nil {
		return at != bt
	}

	if at == bt && at.Comparable() {
		return av != bv
	}

	if at == bt {
		if ap, ok := referencePointer(reflect.ValueOf(av)); ok {
			if bp, ok := referencePointer(reflect.ValueOf(bv)); ok {
				return ap != bp
			}
		}
	}

	return !reflect.DeepEqual(av, bv)
}

func referencePointer(v reflect.Value) (uintptr, bool) {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		if v.IsNil() {
			return 0, true
		}
		return v.Pointer(), true
	default:
		return 0, false
	}
}

func (c *TTLCache[K, V]) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.data)
}

func (c *TTLCache[K, V]) SetTimer(expiration int64, ttlMs int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setTimerLocked(expiration, ttlMs)
}

func (c *TTLCache[K, V]) setTimerLocked(expiration int64, ttlMs int64) {
	if c.hasTimerExpiration && c.timerExpiration < expiration {
		return
	}

	if c.timer != nil {
		c.timer.Stop()
	}

	delayMs := ttlMs
	if delayMs < 0 {
		delayMs = 0
	}
	if delayMs > timerMaxMs {
		delayMs = timerMaxMs
	}

	c.timerExpiration = expiration
	c.hasTimerExpiration = true
	c.timer = c.afterFunc(time.Duration(delayMs)*time.Millisecond, func() {
		c.mu.Lock()
		c.timer = nil
		c.hasTimerExpiration = false
		c.mu.Unlock()

		c.PurgeStale()

		c.mu.Lock()
		nextExpiration, ok := c.firstExpirationLocked()
		if ok {
			remaining := nextExpiration - int64(c.nowFn())
			c.setTimerLocked(nextExpiration, remaining)
		}
		c.mu.Unlock()
	})
}

func (c *TTLCache[K, V]) CancelTimer() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cancelTimerLocked()
}

func (c *TTLCache[K, V]) CancelTimers() {
	c.CancelTimer()
}

func (c *TTLCache[K, V]) cancelTimerLocked() {
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
		c.hasTimerExpiration = false
	}
}

func (c *TTLCache[K, V]) SetTTL(key K, ttl ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var override *time.Duration
	if len(ttl) > 0 {
		t := ttl[0]
		override = &t
	}
	c.setTTLLocked(key, c.resolveTTL(override))
}

func (c *TTLCache[K, V]) resolveTTL(override *time.Duration) *time.Duration {
	if override != nil {
		return override
	}
	return c.ttl
}

func (c *TTLCache[K, V]) setTTLLocked(key K, ttl *time.Duration) {
	if current, ok := c.expirationMap[key]; ok {
		if current == infinityExpiration {
			c.removeImmortalLocked(key)
		} else {
			c.removeFromExpirationLocked(current, key)
		}
	}

	if ttl != nil && *ttl != 0 {
		expiration := int64(math.Floor(c.nowFn() + float64(ttlMillis(*ttl))))
		c.expirationMap[key] = expiration
		if _, ok := c.expirations[expiration]; !ok {
			c.expirations[expiration] = []K{}
			c.insertExpirationLocked(expiration)
			c.setTimerLocked(expiration, ttlMillis(*ttl))
		}
		c.expirations[expiration] = append(c.expirations[expiration], key)
		return
	}

	if _, ok := c.immortalKeys[key]; !ok {
		c.immortalKeys[key] = struct{}{}
		c.immortalOrder = append(c.immortalOrder, key)
	}
	c.expirationMap[key] = infinityExpiration
}

func (c *TTLCache[K, V]) insertExpirationLocked(expiration int64) {
	idx, found := slices.BinarySearch(c.expirationOrder, expiration)
	if found {
		return
	}
	c.expirationOrder = slices.Insert(c.expirationOrder, idx, expiration)
}

func (c *TTLCache[K, V]) removeFromExpirationLocked(expiration int64, key K) {
	keys, ok := c.expirations[expiration]
	if !ok {
		return
	}
	for i, existing := range keys {
		if existing == key {
			keys = slices.Delete(keys, i, i+1)
			break
		}
	}
	if len(keys) == 0 {
		delete(c.expirations, expiration)
		if idx, found := slices.BinarySearch(c.expirationOrder, expiration); found {
			c.expirationOrder = slices.Delete(c.expirationOrder, idx, idx+1)
		}
		return
	}
	c.expirations[expiration] = keys
}

func (c *TTLCache[K, V]) removeImmortalLocked(key K) {
	if _, ok := c.immortalKeys[key]; !ok {
		return
	}
	delete(c.immortalKeys, key)
	for i, existing := range c.immortalOrder {
		if existing == key {
			c.immortalOrder = slices.Delete(c.immortalOrder, i, i+1)
			return
		}
	}
}

func (c *TTLCache[K, V]) Set(key K, value V, opts ...SetOptions[K, V]) *TTLCache[K, V] {
	var opt *SetOptions[K, V]
	if len(opts) > 0 {
		opt = &opts[0]
	}

	ttl := c.ttl
	noUpdateTTL := c.noUpdateTTL
	noDisposeOnSet := c.noDisposeOnSet

	if opt != nil {
		if opt.TTL != nil {
			ttl = opt.TTL
		}
		if opt.NoUpdateTTL != nil {
			noUpdateTTL = *opt.NoUpdateTTL
		}
		if opt.NoDisposeOnSet != nil {
			noDisposeOnSet = *opt.NoDisposeOnSet
		}
	}

	if ttl != nil {
		if err := validatePositiveTTL(*ttl, ErrTTLMustBePositive); err != nil {
			panic(err)
		}
	}

	c.mu.Lock()

	var pending *disposal[K, V]
	if _, exists := c.expirationMap[key]; exists {
		if !noUpdateTTL {
			c.setTTLLocked(key, ttl)
		}

		oldValue, hadValue := c.data[key]
		if !hadValue || valuesDiffer(oldValue, value) {
			c.data[key] = value
			if hadValue && !noDisposeOnSet {
				pending = &disposal[K, V]{
					key:    key,
					value:  oldValue,
					reason: DisposeSet,
				}
			}
		}
	} else {
		c.setTTLLocked(key, ttl)
		c.data[key] = value
	}

	c.mu.Unlock()

	if pending != nil {
		c.dispose(pending.value, pending.key, pending.reason)
	}

	if c.max > 0 {
		for {
			before := c.Size()
			if before <= c.max {
				break
			}
			c.PurgeToCapacity()
			if after := c.Size(); after >= before {
				break
			}
		}
	}

	return c
}

func (c *TTLCache[K, V]) Has(key K, opts ...HasOptions[K, V]) bool {
	var opt *HasOptions[K, V]
	if len(opts) > 0 {
		opt = &opts[0]
	}

	checkAgeOnHas := c.checkAgeOnHas
	updateAgeOnHas := c.updateAgeOnHas
	ttl := c.ttl

	if opt != nil {
		if opt.CheckAgeOnHas != nil {
			checkAgeOnHas = *opt.CheckAgeOnHas
		}
		if opt.UpdateAgeOnHas != nil {
			updateAgeOnHas = *opt.UpdateAgeOnHas
		}
		if opt.TTL != nil {
			ttl = opt.TTL
		}
	}

	c.mu.Lock()
	_, exists := c.data[key]
	if !exists {
		c.mu.Unlock()
		return false
	}

	if checkAgeOnHas && c.getRemainingTTLLocked(key) == 0 {
		removed, hadRemoval := c.deleteLocked(key, DisposeDelete)
		c.mu.Unlock()
		if hadRemoval {
			c.dispose(removed.value, removed.key, removed.reason)
			c.cancelTimerIfEmpty()
		}
		return false
	}

	if updateAgeOnHas {
		c.setTTLLocked(key, ttl)
	}

	c.mu.Unlock()
	return true
}

func (c *TTLCache[K, V]) GetRemainingTTL(key K) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getRemainingTTLLocked(key)
}

func (c *TTLCache[K, V]) getRemainingTTLLocked(key K) int64 {
	expiration, ok := c.expirationMap[key]
	if !ok {
		return 0
	}
	if expiration == infinityExpiration {
		return math.MaxInt64
	}

	remaining := float64(expiration) - c.nowFn()
	if remaining <= 0 {
		return 0
	}
	return int64(math.Ceil(remaining))
}

func (c *TTLCache[K, V]) Get(key K, opts ...GetOptions[K, V]) (V, bool) {
	var opt *GetOptions[K, V]
	if len(opts) > 0 {
		opt = &opts[0]
	}

	updateAgeOnGet := c.updateAgeOnGet
	checkAgeOnGet := c.checkAgeOnGet
	ttl := c.ttl

	if opt != nil {
		if opt.UpdateAgeOnGet != nil {
			updateAgeOnGet = *opt.UpdateAgeOnGet
		}
		if opt.CheckAgeOnGet != nil {
			checkAgeOnGet = *opt.CheckAgeOnGet
		}
		if opt.TTL != nil {
			ttl = opt.TTL
		}
	}

	c.mu.Lock()
	value, exists := c.data[key]
	if !exists {
		c.mu.Unlock()
		var zero V
		return zero, false
	}

	if checkAgeOnGet && c.getRemainingTTLLocked(key) == 0 {
		removed, hadRemoval := c.deleteLocked(key, DisposeDelete)
		c.mu.Unlock()
		if hadRemoval {
			c.dispose(removed.value, removed.key, removed.reason)
			c.cancelTimerIfEmpty()
		}
		var zero V
		return zero, false
	}

	if updateAgeOnGet {
		c.setTTLLocked(key, ttl)
	}

	c.mu.Unlock()
	return value, true
}

func (c *TTLCache[K, V]) Delete(key K) bool {
	c.mu.Lock()
	removed, ok := c.deleteLocked(key, DisposeDelete)
	c.mu.Unlock()
	if !ok {
		return false
	}

	c.dispose(removed.value, removed.key, removed.reason)
	c.cancelTimerIfEmpty()
	return true
}

func (c *TTLCache[K, V]) deleteLocked(key K, reason DisposeReason) (disposal[K, V], bool) {
	current, exists := c.expirationMap[key]
	if !exists {
		var zero disposal[K, V]
		return zero, false
	}

	value, ok := c.data[key]
	if !ok {
		var zero V
		value = zero
	}

	delete(c.data, key)
	delete(c.expirationMap, key)
	if current == infinityExpiration {
		c.removeImmortalLocked(key)
	} else {
		c.removeFromExpirationLocked(current, key)
	}

	return disposal[K, V]{
		key:    key,
		value:  value,
		reason: reason,
	}, true
}

func (c *TTLCache[K, V]) cancelTimerIfEmpty() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.data) == 0 {
		c.cancelTimerLocked()
	}
}

func (c *TTLCache[K, V]) Clear() {
	c.mu.Lock()
	var entries []Entry[K, V]
	if c.hasDispose {
		entries = c.snapshotEntriesLocked()
	}

	c.data = make(map[K]V)
	c.expirationMap = make(map[K]int64)
	c.expirations = make(map[int64][]K)
	c.expirationOrder = nil
	c.immortalKeys = make(map[K]struct{})
	c.immortalOrder = nil
	c.cancelTimerLocked()
	c.mu.Unlock()

	for _, entry := range entries {
		c.dispose(entry.Value, entry.Key, DisposeDelete)
	}
}

func (c *TTLCache[K, V]) PurgeToCapacity() {
	if c.max <= 0 {
		return
	}

	for {
		c.mu.Lock()
		expiration, ok := c.firstExpirationLocked()
		if !ok {
			c.mu.Unlock()
			return
		}

		keys := append([]K(nil), c.expirations[expiration]...)
		if len(keys) == 0 {
			delete(c.expirations, expiration)
			c.expirationOrder = c.expirationOrder[1:]
			c.mu.Unlock()
			continue
		}

		if len(c.data)-len(keys) >= c.max {
			delete(c.expirations, expiration)
			c.expirationOrder = c.expirationOrder[1:]
			entries := c.collectDisposalsLocked(keys, DisposeEvict)
			c.mu.Unlock()
			c.runDisposals(entries)
			continue
		}

		removeCount := len(c.data) - c.max
		if removeCount <= 0 {
			c.mu.Unlock()
			return
		}
		if removeCount > len(keys) {
			removeCount = len(keys)
		}

		removed := append([]K(nil), keys[:removeCount]...)
		remaining := append([]K(nil), keys[removeCount:]...)
		if len(remaining) == 0 {
			delete(c.expirations, expiration)
			c.expirationOrder = c.expirationOrder[1:]
		} else {
			c.expirations[expiration] = remaining
		}

		entries := c.collectDisposalsLocked(removed, DisposeEvict)
		c.mu.Unlock()
		c.runDisposals(entries)
		return
	}
}

func (c *TTLCache[K, V]) PurgeStale() {
	for {
		c.mu.Lock()
		expiration, ok := c.firstExpirationLocked()
		if !ok {
			if len(c.data) == 0 {
				c.cancelTimerLocked()
			}
			c.mu.Unlock()
			return
		}

		now := int64(math.Ceil(c.nowFn()))
		if expiration > now {
			if len(c.data) == 0 {
				c.cancelTimerLocked()
			}
			c.mu.Unlock()
			return
		}

		keys := append([]K(nil), c.expirations[expiration]...)
		delete(c.expirations, expiration)
		c.expirationOrder = c.expirationOrder[1:]
		entries := c.collectDisposalsLocked(keys, DisposeStale)
		c.mu.Unlock()
		c.runDisposals(entries)
	}
}

func (c *TTLCache[K, V]) firstExpirationLocked() (int64, bool) {
	for len(c.expirationOrder) > 0 {
		expiration := c.expirationOrder[0]
		if keys, ok := c.expirations[expiration]; ok && len(keys) > 0 {
			return expiration, true
		}
		c.expirationOrder = c.expirationOrder[1:]
		delete(c.expirations, expiration)
	}
	return 0, false
}

func (c *TTLCache[K, V]) collectDisposalsLocked(keys []K, reason DisposeReason) []disposal[K, V] {
	entries := make([]disposal[K, V], 0, len(keys))
	for _, key := range keys {
		value, ok := c.data[key]
		if !ok {
			var zero V
			value = zero
		}
		delete(c.data, key)
		delete(c.expirationMap, key)
		entries = append(entries, disposal[K, V]{
			key:    key,
			value:  value,
			reason: reason,
		})
	}
	return entries
}

func (c *TTLCache[K, V]) runDisposals(entries []disposal[K, V]) {
	for _, entry := range entries {
		c.dispose(entry.value, entry.key, entry.reason)
	}
}

func (c *TTLCache[K, V]) Entries() <-chan Entry[K, V] {
	c.mu.Lock()
	entries := c.snapshotEntriesLocked()
	c.mu.Unlock()

	ch := make(chan Entry[K, V], len(entries))
	for _, entry := range entries {
		ch <- entry
	}
	close(ch)
	return ch
}

func (c *TTLCache[K, V]) snapshotEntriesLocked() []Entry[K, V] {
	keys := c.snapshotKeysLocked()
	entries := make([]Entry[K, V], 0, len(keys))
	for _, key := range keys {
		value, ok := c.data[key]
		if !ok {
			var zero V
			value = zero
		}
		entries = append(entries, Entry[K, V]{Key: key, Value: value})
	}
	return entries
}

func (c *TTLCache[K, V]) Keys() <-chan K {
	c.mu.Lock()
	keys := c.snapshotKeysLocked()
	c.mu.Unlock()

	ch := make(chan K, len(keys))
	for _, key := range keys {
		ch <- key
	}
	close(ch)
	return ch
}

func (c *TTLCache[K, V]) snapshotKeysLocked() []K {
	keys := make([]K, 0, len(c.expirationMap))
	for _, expiration := range c.expirationOrder {
		keys = append(keys, c.expirations[expiration]...)
	}
	keys = append(keys, c.immortalOrder...)
	return keys
}

func (c *TTLCache[K, V]) Values() <-chan V {
	c.mu.Lock()
	values := c.snapshotValuesLocked()
	c.mu.Unlock()

	ch := make(chan V, len(values))
	for _, value := range values {
		ch <- value
	}
	close(ch)
	return ch
}

func (c *TTLCache[K, V]) snapshotValuesLocked() []V {
	keys := c.snapshotKeysLocked()
	values := make([]V, 0, len(keys))
	for _, key := range keys {
		value, ok := c.data[key]
		if !ok {
			var zero V
			value = zero
		}
		values = append(values, value)
	}
	return values
}

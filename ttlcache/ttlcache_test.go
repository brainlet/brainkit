package ttlcache

import (
	"testing"
	"time"
)

func TestBasicOperation(t *testing.T) {
	c, err := New[int, int](TTLCacheOptions[int, int]{
		TTL: durationPtr(1000 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	c.Set(1, 2)

	if !c.Has(1) {
		t.Error("Expected has(1) to be true")
	}

	val, ok := c.Get(1)
	if !ok || val != 2 {
		t.Errorf("Expected get(1)=2, got %v, %v", val, ok)
	}

	time.Sleep(1100 * time.Millisecond)
	c.PurgeStale()

	if c.Has(1) {
		t.Error("Expected has(1) to be false after expiration")
	}
}

func TestUpdateAgeOnGet(t *testing.T) {
	c, err := New[int, int](TTLCacheOptions[int, int]{
		TTL:            durationPtr(1000 * time.Millisecond),
		UpdateAgeOnGet: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	c.Set(1, 2)

	ttl1 := c.GetRemainingTTL(1)
	if ttl1 <= 0 {
		t.Errorf("Expected positive TTL initially, got %d", ttl1)
	}

	time.Sleep(50 * time.Millisecond)

	c.Get(1)

	ttl2 := c.GetRemainingTTL(1)
	if ttl2 <= 0 {
		t.Errorf("Expected positive TTL after get with updateAgeOnGet, got %d", ttl2)
	}
}

func TestBadValues(t *testing.T) {
	_, err := New[int, int](TTLCacheOptions[int, int]{
		Max: intPtr(-1),
	})
	if err != ErrMaxMustBePositive {
		t.Errorf("Expected ErrMaxMustBePositive, got %v", err)
	}

	_, err = New[int, int](TTLCacheOptions[int, int]{
		TTL: durationPtr(-1 * time.Millisecond),
	})
	if err != ErrTTLMustBePositive {
		t.Errorf("Expected ErrTTLMustBePositive, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	c, err := New[int, int](TTLCacheOptions[int, int]{
		TTL: durationPtr(10 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	c.Set(1, 1)
	c.Set(2, 2)

	if !c.Delete(1) {
		t.Error("Expected delete(1) to return true")
	}

	if c.Has(1) {
		t.Error("Expected has(1) to return false after delete")
	}

	if !c.Has(2) {
		t.Error("Expected has(2) to still return true")
	}
}

func TestClear(t *testing.T) {
	var disposals int

	dispose := func(value int, key int, reason DisposeReason) {
		disposals++
	}

	c, err := New[int, int](TTLCacheOptions[int, int]{
		TTL:     durationPtr(10 * time.Millisecond),
		Dispose: dispose,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	c.Set(1, 1)
	c.Set(2, 2)

	c.Clear()

	if c.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", c.Size())
	}

	if disposals != 2 {
		t.Errorf("Expected 2 disposals on clear, got %d", disposals)
	}
}

func TestIterators(t *testing.T) {
	c, err := New[int, int](TTLCacheOptions[int, int]{
		TTL: durationPtr(10 * time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	c.Set(1, 10)
	c.Set(2, 20)
	c.Set(3, 30)

	keys := 0
	for range c.Keys() {
		keys++
	}
	if keys != 3 {
		t.Errorf("Expected 3 keys, got %d", keys)
	}

	values := 0
	for range c.Values() {
		values++
	}
	if values != 3 {
		t.Errorf("Expected 3 values, got %d", values)
	}

	entries := 0
	for range c.Entries() {
		entries++
	}
	if entries != 3 {
		t.Errorf("Expected 3 entries, got %d", entries)
	}
}

func TestMaxCapacity(t *testing.T) {
	var disposals int

	dispose := func(value int, key int, reason DisposeReason) {
		disposals++
	}

	c, err := New[int, int](TTLCacheOptions[int, int]{
		Max:     intPtr(2),
		TTL:     durationPtr(10 * time.Millisecond),
		Dispose: dispose,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	c.Set(1, 1)
	c.Set(2, 2)

	if c.Size() != 2 {
		t.Errorf("Expected size 2, got %d", c.Size())
	}

	c.Set(3, 3)

	if c.Size() != 2 {
		t.Errorf("Expected size 2 after eviction, got %d", c.Size())
	}

	if disposals != 1 {
		t.Errorf("Expected 1 disposal, got %d", disposals)
	}
}

func TestPurgeStale(t *testing.T) {
	var disposals int

	dispose := func(value int, key int, reason DisposeReason) {
		disposals++
	}

	c, err := New[int, int](TTLCacheOptions[int, int]{
		TTL:     durationPtr(10 * time.Millisecond),
		Dispose: dispose,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	c.Set(1, 1)
	c.Set(2, 2)

	time.Sleep(20 * time.Millisecond)

	c.PurgeStale()

	if c.Size() != 0 {
		t.Errorf("Expected size 0 after PurgeStale, got %d", c.Size())
	}

	if disposals != 2 {
		t.Errorf("Expected 2 disposals, got %d", disposals)
	}
}

func TestInfinityTTL(t *testing.T) {
	c, err := New[int, int](TTLCacheOptions[int, int]{})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Set with TTL=0 to make it immortal (represents Infinity in TS)
	c.Set(1, 11, SetOptions[int, int]{
		TTL: durationPtr(0),
	})

	if !c.Has(1) {
		t.Error("Expected has(1) to be true for immortal key")
	}

	time.Sleep(20 * time.Millisecond)

	if !c.Has(1) {
		t.Error("Expected immortal key to still exist")
	}
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func intPtr(i int) *int {
	return &i
}

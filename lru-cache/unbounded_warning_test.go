package lrucache

import (
	"sync"
	"testing"
)

func TestUnboundedWarning_TTLOnlyWarns(t *testing.T) {
	warned = syncMapForTests()
	defer func() { warned = syncMapForTests() }()

	c := New[int, int](Options[int, int]{TTL: 100})
	if c == nil {
		t.Fatal("TTL-only cache should be created")
	}
	if _, ok := warned.Load("LRU_CACHE_UNBOUNDED"); !ok {
		t.Fatal("expected unbounded warning to be recorded")
	}
}

func TestUnboundedWarning_Deduplicated(t *testing.T) {
	warned = syncMapForTests()
	defer func() { warned = syncMapForTests() }()

	New[int, int](Options[int, int]{TTL: 100})
	New[string, string](Options[string, string]{TTL: 100})

	count := 0
	warned.Range(func(_, _ any) bool {
		count++
		return true
	})
	if count != 1 {
		t.Fatalf("expected 1 warning code, got %d", count)
	}
}

func TestUnboundedWarning_TTLWithAutopurgeOK(t *testing.T) {
	c := New[int, int](Options[int, int]{
		TTL:          100,
		TTLAutopurge: true,
	})
	if c == nil {
		t.Fatal("TTL + ttlAutopurge should create a valid cache")
	}
}

func TestUnboundedWarning_TTLWithMaxOK(t *testing.T) {
	c := New[int, int](Options[int, int]{
		TTL: 100,
		Max: 10,
	})
	if c == nil {
		t.Fatal("TTL + max should create a valid cache")
	}
}

func TestUnboundedWarning_TTLWithMaxSizeOK(t *testing.T) {
	c := New[int, int](Options[int, int]{
		TTL:     100,
		MaxSize: 1000,
	})
	if c == nil {
		t.Fatal("TTL + maxSize should create a valid cache")
	}
}

func syncMapForTests() sync.Map {
	return sync.Map{}
}

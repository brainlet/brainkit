package brainkit

import (
	"testing"

	"github.com/brainlet/brainkit/bus"
)

func TestInstanceManager_SpawnAndKillPool(t *testing.T) {
	im := NewInstanceManager()

	err := im.SpawnPool("workers", PoolConfig{
		Base:         Config{Namespace: "worker"},
		InitialCount: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	info, err := im.PoolInfo("workers")
	if err != nil {
		t.Fatal(err)
	}
	if info.Current != 3 {
		t.Errorf("expected 3 instances, got %d", info.Current)
	}

	pools := im.Pools()
	if len(pools) != 1 {
		t.Errorf("pools = %v", pools)
	}

	err = im.KillPool("workers")
	if err != nil {
		t.Fatal(err)
	}

	pools = im.Pools()
	if len(pools) != 0 {
		t.Errorf("expected 0 pools after kill, got %d", len(pools))
	}
}

func TestInstanceManager_ScaleUpAndDown(t *testing.T) {
	im := NewInstanceManager()

	im.SpawnPool("scalable", PoolConfig{
		Base:         Config{Namespace: "scale"},
		InitialCount: 2,
	})
	defer im.KillPool("scalable")

	err := im.Scale("scalable", 3)
	if err != nil {
		t.Fatal(err)
	}

	info, _ := im.PoolInfo("scalable")
	if info.Current != 5 {
		t.Errorf("expected 5 after scale-up, got %d", info.Current)
	}

	err = im.Scale("scalable", -2)
	if err != nil {
		t.Fatal(err)
	}

	info, _ = im.PoolInfo("scalable")
	if info.Current != 3 {
		t.Errorf("expected 3 after scale-down, got %d", info.Current)
	}
}

func TestInstanceManager_DuplicatePoolError(t *testing.T) {
	im := NewInstanceManager()

	im.SpawnPool("dup", PoolConfig{
		Base:         Config{Namespace: "dup"},
		InitialCount: 1,
	})
	defer im.KillPool("dup")

	err := im.SpawnPool("dup", PoolConfig{
		Base:         Config{Namespace: "dup"},
		InitialCount: 1,
	})
	if err == nil {
		t.Fatal("expected error for duplicate pool name")
	}
}

func TestStaticStrategy_RestoreTarget(t *testing.T) {
	strategy := NewStaticStrategy(3)

	decision := strategy.Evaluate(bus.BusMetrics{}, PoolInfo{Current: 1, Min: 1})
	if decision.Action != "scale-up" || decision.Delta != 2 {
		t.Errorf("expected scale-up by 2, got %+v", decision)
	}

	decision = strategy.Evaluate(bus.BusMetrics{}, PoolInfo{Current: 3, Min: 1})
	if decision.Action != "none" {
		t.Errorf("expected none, got %+v", decision)
	}

	decision = strategy.Evaluate(bus.BusMetrics{}, PoolInfo{Current: 5, Min: 1})
	if decision.Action != "scale-down" || decision.Delta != 2 {
		t.Errorf("expected scale-down by 2, got %+v", decision)
	}
}

func TestThresholdStrategy_ScaleUpOnHighPending(t *testing.T) {
	strategy := NewThresholdStrategy(100, 10)

	// High pending → scale up
	decision := strategy.Evaluate(bus.BusMetrics{}, PoolInfo{Current: 2, Pending: 150, Min: 1, Max: 10})
	if decision.Action != "scale-up" {
		t.Errorf("expected scale-up, got %+v", decision)
	}

	// Low pending → scale down
	decision = strategy.Evaluate(bus.BusMetrics{}, PoolInfo{Current: 5, Pending: 5, Min: 1, Max: 10})
	if decision.Action != "scale-down" {
		t.Errorf("expected scale-down, got %+v", decision)
	}

	// Normal pending → none
	decision = strategy.Evaluate(bus.BusMetrics{}, PoolInfo{Current: 3, Pending: 50, Min: 1, Max: 10})
	if decision.Action != "none" {
		t.Errorf("expected none, got %+v", decision)
	}

	// At max → no more scale up
	decision = strategy.Evaluate(bus.BusMetrics{}, PoolInfo{Current: 10, Pending: 200, Min: 1, Max: 10})
	if decision.Action != "none" {
		t.Errorf("expected none at max, got %+v", decision)
	}

	// At min → no more scale down
	decision = strategy.Evaluate(bus.BusMetrics{}, PoolInfo{Current: 1, Pending: 0, Min: 1, Max: 10})
	if decision.Action != "none" {
		t.Errorf("expected none at min, got %+v", decision)
	}
}

func TestInstanceManager_EvaluateAndScale(t *testing.T) {
	im := NewInstanceManager()

	im.SpawnPool("auto", PoolConfig{
		Base:         Config{Namespace: "auto"},
		InitialCount: 2,
		Strategy:     NewStaticStrategy(5),
	})
	defer im.KillPool("auto")

	im.EvaluateAndScale()

	info, _ := im.PoolInfo("auto")
	if info.Current != 5 {
		t.Errorf("expected 5 after auto-scale, got %d", info.Current)
	}
}

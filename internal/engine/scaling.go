package engine

import (
	"context"
	"fmt"
	"log/slog"
	"github.com/brainlet/brainkit/internal/syncx"

	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
)

// PoolConfig configures a Kit pool.
type PoolConfig struct {
	Base         NodeConfig
	InitialCount int
	Min          int
	Max          int
	Strategy     ScalingStrategy
}

type pool struct {
	name        string
	config      PoolConfig
	instances   []*Node
	sharedTools *tools.ToolRegistry
	mu          syncx.Mutex
}

// InstanceManager manages pools of Kit instances with shared tool registries.
type InstanceManager struct {
	mu    syncx.Mutex
	pools map[string]*pool
}

func NewInstanceManager() *InstanceManager {
	return &InstanceManager{
		pools: make(map[string]*pool),
	}
}

// SpawnPool creates a new pool with the given number of Kit instances.
func (im *InstanceManager) SpawnPool(name string, cfg PoolConfig) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if _, exists := im.pools[name]; exists {
		return &sdk.AlreadyExistsError{Resource: "pool", Name: name}
	}

	count := cfg.InitialCount
	if count <= 0 {
		count = 1
	}

	sharedTools := tools.New()

	p := &pool{
		name:        name,
		config:      cfg,
		sharedTools: sharedTools,
	}

	for i := 0; i < count; i++ {
		kit, err := im.spawnInstance(p, i)
		if err != nil {
			for _, k := range p.instances {
				k.Close()
			}
			return fmt.Errorf("scaling: spawn pool %q instance %d: %w", name, i, err)
		}
		p.instances = append(p.instances, kit)
	}

	im.pools[name] = p
	slog.Info("pool spawned", slog.String("pool", name), slog.Int("instances", count))
	return nil
}

// Scale adjusts the number of instances in a pool.
func (im *InstanceManager) Scale(name string, delta int) error {
	im.mu.Lock()
	p, ok := im.pools[name]
	im.mu.Unlock()

	if !ok {
		return &sdk.NotFoundError{Resource: "pool", Name: name}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if delta > 0 {
		for i := 0; i < delta; i++ {
			idx := len(p.instances)
			node, err := im.spawnInstance(p, idx)
			if err != nil {
				return fmt.Errorf("scaling: scale-up pool %q: %w", name, err)
			}
			p.instances = append(p.instances, node)
		}
		slog.Info("pool scaled up", slog.String("pool", name), slog.Int("delta", delta), slog.Int("total", len(p.instances)))
	} else if delta < 0 {
		remove := -delta
		if remove > len(p.instances) {
			remove = len(p.instances)
		}
		for i := 0; i < remove; i++ {
			idx := len(p.instances) - 1
			p.instances[idx].Close()
			p.instances = p.instances[:idx]
		}
		slog.Info("pool scaled down", slog.String("pool", name), slog.Int("delta", remove), slog.Int("total", len(p.instances)))
	}

	return nil
}

// KillPool shuts down all instances and removes the pool.
func (im *InstanceManager) KillPool(name string) error {
	im.mu.Lock()
	p, ok := im.pools[name]
	if !ok {
		im.mu.Unlock()
		return &sdk.NotFoundError{Resource: "pool", Name: name}
	}
	delete(im.pools, name)
	im.mu.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, kit := range p.instances {
		kit.Close()
	}
	p.instances = nil

	slog.Info("pool killed", slog.String("pool", name))
	return nil
}

// PoolInfo returns information about a pool.
func (im *InstanceManager) PoolInfo(name string) (PoolInfo, error) {
	im.mu.Lock()
	p, ok := im.pools[name]
	im.mu.Unlock()

	if !ok {
		return PoolInfo{}, &sdk.NotFoundError{Resource: "pool", Name: name}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	min := p.config.Min
	if min <= 0 {
		min = 1
	}

	return PoolInfo{
		Name:    name,
		Current: len(p.instances),
		Min:     min,
		Max:     p.config.Max,
	}, nil
}

// Pools returns the names of all pools.
func (im *InstanceManager) Pools() []string {
	im.mu.Lock()
	defer im.mu.Unlock()

	names := make([]string, 0, len(im.pools))
	for name := range im.pools {
		names = append(names, name)
	}
	return names
}

// EvaluateAndScale checks all pools with strategies and applies scaling decisions.
func (im *InstanceManager) EvaluateAndScale() {
	im.mu.Lock()
	poolsCopy := make(map[string]*pool, len(im.pools))
	for k, v := range im.pools {
		poolsCopy[k] = v
	}
	im.mu.Unlock()

	for name, p := range poolsCopy {
		if p.config.Strategy == nil {
			continue
		}

		info, err := im.PoolInfo(name)
		if err != nil {
			continue
		}

		metrics := transport.MetricsSnapshot{
			Published: make(map[string]int),
			Handled:   make(map[string]int),
			Errors:    make(map[string]int),
		}
		decision := p.config.Strategy.Evaluate(metrics, info)

		switch decision.Action {
		case "scale-up":
			if decision.Delta > 0 {
				slog.Info("scaling decision", slog.String("pool", name), slog.String("reason", decision.Reason))
				im.Scale(name, decision.Delta)
			}
		case "scale-down":
			if decision.Delta > 0 {
				slog.Info("scaling decision", slog.String("pool", name), slog.String("reason", decision.Reason))
				im.Scale(name, -decision.Delta)
			}
		}
	}
}

func (im *InstanceManager) spawnInstance(p *pool, idx int) (*Node, error) {
	cfg := p.config.Base
	cfg.Kernel.Namespace = fmt.Sprintf("%s-%s-%d", p.name, cfg.Kernel.Namespace, idx)
	cfg.Kernel.SharedTools = p.sharedTools

	node, err := NewNode(cfg)
	if err != nil {
		return nil, err
	}
	if err := node.Start(context.Background()); err != nil {
		_ = node.Close()
		return nil, err
	}
	return node, nil
}




// StaticStrategy maintains a fixed instance count.
type StaticStrategy struct {
	Target int
}

func NewStaticStrategy(target int) *StaticStrategy {
	return &StaticStrategy{Target: target}
}

func (s *StaticStrategy) Evaluate(_ transport.MetricsSnapshot, pool PoolInfo) ScalingDecision {
	if pool.Current < s.Target {
		return ScalingDecision{
			Action: "scale-up",
			Delta:  s.Target - pool.Current,
			Reason: "static: restore target count",
		}
	}
	if pool.Current > s.Target {
		return ScalingDecision{
			Action: "scale-down",
			Delta:  pool.Current - s.Target,
			Reason: "static: reduce to target count",
		}
	}
	return ScalingDecision{Action: "none"}
}

// ThresholdStrategy scales based on pending message count.
type ThresholdStrategy struct {
	ScaleUpThreshold   int
	ScaleDownThreshold int
	ScaleUpStep        int
	ScaleDownStep      int
}

func NewThresholdStrategy(scaleUp, scaleDown int) *ThresholdStrategy {
	return &ThresholdStrategy{
		ScaleUpThreshold:   scaleUp,
		ScaleDownThreshold: scaleDown,
		ScaleUpStep:        1,
		ScaleDownStep:      1,
	}
}

func (s *ThresholdStrategy) Evaluate(_ transport.MetricsSnapshot, pool PoolInfo) ScalingDecision {
	pending := pool.Pending

	if pending > s.ScaleUpThreshold {
		if pool.Max > 0 && pool.Current >= pool.Max {
			return ScalingDecision{Action: "none", Reason: "threshold: at max instances"}
		}
		delta := s.ScaleUpStep
		if pool.Max > 0 && pool.Current+delta > pool.Max {
			delta = pool.Max - pool.Current
		}
		if delta <= 0 {
			return ScalingDecision{Action: "none"}
		}
		return ScalingDecision{
			Action: "scale-up",
			Delta:  delta,
			Reason: fmt.Sprintf("threshold: pending=%d > threshold=%d", pending, s.ScaleUpThreshold),
		}
	}

	if pending < s.ScaleDownThreshold && pool.Current > pool.Min {
		delta := s.ScaleDownStep
		if pool.Current-delta < pool.Min {
			delta = pool.Current - pool.Min
		}
		if delta <= 0 {
			return ScalingDecision{Action: "none"}
		}
		return ScalingDecision{
			Action: "scale-down",
			Delta:  delta,
			Reason: fmt.Sprintf("threshold: pending=%d < threshold=%d", pending, s.ScaleDownThreshold),
		}
	}

	return ScalingDecision{Action: "none"}
}

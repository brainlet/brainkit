package brainkit

import (
	"fmt"
	"log"
	"sync"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

// PoolConfig configures a Kit pool.
type PoolConfig struct {
	Base         Config
	InitialCount int
	Min          int // minimum instances (default: 1)
	Max          int // maximum instances (0 = unlimited)
	Strategy     ScalingStrategy
}

type pool struct {
	name        string
	config      PoolConfig
	instances   []*Kit
	sharedBus   *bus.Bus
	sharedTools *registry.ToolRegistry
	mu          sync.Mutex
}

// InstanceManager manages pools of Kit instances with shared buses.
type InstanceManager struct {
	mu    sync.Mutex
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
		return fmt.Errorf("scaling: pool %q already exists", name)
	}

	count := cfg.InitialCount
	if count <= 0 {
		count = 1
	}

	sharedBus := bus.NewBus(bus.NewInProcessTransport())
	sharedTools := registry.New()

	p := &pool{
		name:        name,
		config:      cfg,
		sharedBus:   sharedBus,
		sharedTools: sharedTools,
	}

	for i := 0; i < count; i++ {
		kit, err := im.spawnInstance(p, i)
		if err != nil {
			for _, k := range p.instances {
				k.Close()
			}
			sharedBus.Close()
			return fmt.Errorf("scaling: spawn pool %q instance %d: %w", name, i, err)
		}
		p.instances = append(p.instances, kit)
	}

	im.pools[name] = p
	log.Printf("[scaling] pool %q spawned with %d instances", name, count)
	return nil
}

// Scale adjusts the number of instances in a pool.
func (im *InstanceManager) Scale(name string, delta int) error {
	im.mu.Lock()
	p, ok := im.pools[name]
	im.mu.Unlock()

	if !ok {
		return fmt.Errorf("scaling: pool %q not found", name)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if delta > 0 {
		for i := 0; i < delta; i++ {
			idx := len(p.instances)
			kit, err := im.spawnInstance(p, idx)
			if err != nil {
				return fmt.Errorf("scaling: scale-up pool %q: %w", name, err)
			}
			p.instances = append(p.instances, kit)
		}
		log.Printf("[scaling] pool %q scaled up by %d (now %d)", name, delta, len(p.instances))
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
		log.Printf("[scaling] pool %q scaled down by %d (now %d)", name, remove, len(p.instances))
	}

	return nil
}

// KillPool shuts down all instances and removes the pool.
func (im *InstanceManager) KillPool(name string) error {
	im.mu.Lock()
	p, ok := im.pools[name]
	if !ok {
		im.mu.Unlock()
		return fmt.Errorf("scaling: pool %q not found", name)
	}
	delete(im.pools, name)
	im.mu.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, kit := range p.instances {
		kit.Close()
	}
	p.sharedBus.Close()
	p.instances = nil

	log.Printf("[scaling] pool %q killed", name)
	return nil
}

// PoolInfo returns information about a pool.
func (im *InstanceManager) PoolInfo(name string) (PoolInfo, error) {
	im.mu.Lock()
	p, ok := im.pools[name]
	im.mu.Unlock()

	if !ok {
		return PoolInfo{}, fmt.Errorf("scaling: pool %q not found", name)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	metrics := p.sharedBus.Metrics()
	pending := 0
	for _, wg := range metrics.Transport.Workers {
		pending += wg.Pending
	}

	min := p.config.Min
	if min <= 0 {
		min = 1
	}

	return PoolInfo{
		Name:    name,
		Current: len(p.instances),
		Min:     min,
		Max:     p.config.Max,
		Pending: pending,
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

		metrics := p.sharedBus.Metrics()
		decision := p.config.Strategy.Evaluate(metrics, info)

		switch decision.Action {
		case "scale-up":
			if decision.Delta > 0 {
				log.Printf("[scaling] %s: %s", name, decision.Reason)
				im.Scale(name, decision.Delta)
			}
		case "scale-down":
			if decision.Delta > 0 {
				log.Printf("[scaling] %s: %s", name, decision.Reason)
				im.Scale(name, -decision.Delta)
			}
		}
	}
}

func (im *InstanceManager) spawnInstance(p *pool, idx int) (*Kit, error) {
	cfg := p.config.Base
	cfg.Name = fmt.Sprintf("%s-%s-%d", p.name, cfg.Name, idx)
	cfg.SharedBus = p.sharedBus
	cfg.SharedTools = p.sharedTools
	cfg.WorkerGroup = p.name

	kit, err := New(cfg)
	if err != nil {
		return nil, err
	}
	return kit, nil
}

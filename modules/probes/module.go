// Package probes runs periodic health probes against a Kit's registered
// AI providers, vector stores, and storage backends. Construct with
// probes.New(probes.Config{...}) and include in brainkit.Config.Modules.
//
// The on-demand per-resource probe helpers (*Kit).ProbeAll is always
// available in core — this module adds the periodic ticker.
package probes

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
)

// Config configures periodic probing.
type Config struct {
	// Interval between probe sweeps. Zero disables the repeating ticker but
	// still allows the initial Mount probe when ProbeOnRegister is enabled.
	// 60s is a reasonable default.
	Interval time.Duration
	// ProbeOnRegister, when true, runs an initial probe sweep as soon as
	// Mount returns. Default: true.
	ProbeOnRegister bool

	probeOnRegisterSet bool
}

// Module runs the periodic probe loop.
type Module struct {
	cfg Config

	mu                sync.Mutex
	runner            bkmodule.ProbeRunner
	cancel            context.CancelFunc
	done              chan struct{}
	closing           bool
	closed            bool
	loopRunning       bool
	lastSweepStarted  time.Time
	lastSweepFinished time.Time

	activeSweeps atomic.Int64
}

// New builds a probes module.
func New(cfg Config) *Module {
	return &Module{cfg: cfg}
}

// YAML is the config shape decoded by the registry factory.
// ProbeOnRegister is a pointer so "absent" differs from "explicit
// false" — the factory defaults absent to true.
type YAML struct {
	Interval        time.Duration `yaml:"interval"`
	ProbeOnRegister *bool         `yaml:"probe_on_register"`
}

// Factory is the registered ModuleFactory for probes.
type Factory struct{}

// Build decodes YAML and returns the module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	probeOnRegister := true
	if y.ProbeOnRegister != nil {
		probeOnRegister = *y.ProbeOnRegister
	}
	return New(Config{
		Interval:           y.Interval,
		ProbeOnRegister:    probeOnRegister,
		probeOnRegisterSet: y.ProbeOnRegister != nil,
	}), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "probes",
		Status:  bkmodule.StatusBeta,
		Summary: "Periodic health probes of providers, vector stores, and storages.",
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[bkmodule.ProbeRunner](bkmodule.CapabilityProbeAll),
			bkmodule.OptionalCapabilityOf[bkmodule.LifecycleDebugRegistry](bkmodule.CapabilityLifecycleDebugRegistry),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindScheduler, "probes.loop", "Periodic provider/storage/vector probe loop."),
		},
	}
}

func init() { bkmodule.Register("probes", Factory{}) }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "probes" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	runner, err := bkmodule.RequireCapability[bkmodule.ProbeRunner](host, bkmodule.CapabilityProbeAll)
	if err != nil {
		return fmt.Errorf("probes: %w", err)
	}
	m.start(runner)
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindScheduler, "probes.loop", "Periodic provider/storage/vector probe loop."))
	host.Scope().Defer(func(closeCtx context.Context) error { return m.CloseContext(closeCtx) })

	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	if lifecycleDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "probes", func() any {
			return m.DebugSnapshot()
		})
		if err != nil {
			return fmt.Errorf("probes: lifecycle debug: %w", err)
		}
		host.Scope().Defer(handle.Close)
	}
	return nil
}

func (m *Module) start(runner bkmodule.ProbeRunner) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	probeOnRegister := m.probeOnRegister()
	interval := m.cfg.Interval

	m.mu.Lock()
	m.runner = runner
	m.cancel = cancel
	m.done = done
	m.closing = false
	m.closed = false
	m.loopRunning = true
	m.lastSweepStarted = time.Time{}
	m.lastSweepFinished = time.Time{}
	m.mu.Unlock()

	go m.run(ctx, runner, probeOnRegister, interval, done)
}

// Close stops the periodic loop.
func (m *Module) Close() error {
	return m.CloseContext(context.Background())
}

// CloseContext cancels and joins the periodic probe loop under caller-owned
// lifecycle cancellation.
func (m *Module) CloseContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.mu.Lock()
	cancel := m.cancel
	done := m.done
	m.closing = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.closing = false
		m.mu.Unlock()
	}()

	if cancel != nil {
		cancel()
	}
	if done == nil {
		m.mu.Lock()
		m.closed = true
		m.runner = nil
		m.cancel = nil
		m.mu.Unlock()
		return nil
	}
	select {
	case <-done:
		m.mu.Lock()
		if m.done == done {
			m.closed = true
			m.runner = nil
			m.cancel = nil
			m.done = nil
		}
		m.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Module) run(ctx context.Context, runner bkmodule.ProbeRunner, probeOnRegister bool, interval time.Duration, done chan struct{}) {
	defer func() {
		m.mu.Lock()
		if m.done == done {
			m.loopRunning = false
		}
		m.mu.Unlock()
		close(done)
	}()

	if probeOnRegister {
		m.runSweep(ctx, runner)
	}
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.runSweep(ctx, runner)
		case <-ctx.Done():
			return
		}
	}
}

func (m *Module) runSweep(ctx context.Context, runner bkmodule.ProbeRunner) {
	if runner == nil || ctx.Err() != nil {
		return
	}
	m.activeSweeps.Add(1)
	m.mu.Lock()
	m.lastSweepStarted = time.Now()
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.lastSweepFinished = time.Now()
		m.mu.Unlock()
		m.activeSweeps.Add(-1)
	}()
	runner.ProbeAll(ctx)
}

func (m *Module) probeOnRegister() bool {
	if !m.cfg.probeOnRegisterSet && !m.cfg.ProbeOnRegister {
		return true
	}
	return m.cfg.ProbeOnRegister
}

// Package probes runs periodic health probes against a Kit's registered
// AI providers, vector stores, and storage backends. Construct with
// probes.New(probes.Config{...}) and include in brainkit.Config.Modules.
//
// The on-demand per-resource probe helpers (*Kit).ProbeAll is always
// available in core — this module adds the periodic ticker.
package probes

import (
	"context"
	"sync/atomic"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
)

// Config configures periodic probing.
type Config struct {
	// Interval between probe sweeps. Zero disables the periodic probe; the
	// module's Mount becomes a no-op. 60s is a reasonable default.
	Interval time.Duration
	// ProbeOnRegister, when true, runs an initial probe sweep as soon as
	// Mount returns. Default: true.
	ProbeOnRegister bool
}

// Module runs the periodic probe loop.
type Module struct {
	cfg    Config
	probe  func()
	closed atomic.Bool
	stop   chan struct{}
}

// New builds a probes module.
func New(cfg Config) *Module {
	return &Module{cfg: cfg, stop: make(chan struct{})}
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
		Interval:        y.Interval,
		ProbeOnRegister: probeOnRegister,
	}), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "probes",
		Status:  bkmodule.StatusBeta,
		Summary: "Periodic health probes of providers, vector stores, and storages.",
	}
}

func init() { bkmodule.Register("probes", Factory{}) }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "probes" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	probe, ok := bkmodule.Capability[func()](host, bkmodule.CapabilityProbeAll)
	if !ok {
		return nil
	}
	m.start(probe)
	host.Scope().Defer(func(context.Context) error { return m.Close() })
	return nil
}

func (m *Module) start(probe func()) {
	m.probe = probe
	m.stop = make(chan struct{})
	m.closed.Store(false)
	probeOnRegister := m.cfg.ProbeOnRegister
	// Default to true when the user didn't explicitly pick a value.
	if !probeOnRegister {
		// Allow explicit opt-out by leaving ProbeOnRegister=false; but because
		// Go zero values make "unset" indistinguishable from "false", treat
		// zero as a signal to probe once.
		probeOnRegister = true
	}
	if probeOnRegister {
		go probe()
	}

	if m.cfg.Interval > 0 {
		go m.loop()
	}
}

// Close stops the periodic loop.
func (m *Module) Close() error {
	if m.closed.CompareAndSwap(false, true) && m.stop != nil {
		close(m.stop)
	}
	return nil
}

func (m *Module) loop() {
	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if m.probe != nil {
				m.probe()
			}
		case <-m.stop:
			return
		}
	}
}

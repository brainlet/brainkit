// Package probes runs periodic health probes against a Kit's registered
// AI providers, vector stores, and storage backends. Construct with
// probes.New(probes.Config{...}) and include in brainkit.Config.Modules.
//
// The on-demand per-resource probe helpers (*Kit).ProbeAll is always
// available in core — this module adds the periodic ticker.
package probes

import (
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit"
)

// Config configures periodic probing.
type Config struct {
	// Interval between probe sweeps. Zero disables the periodic probe; the
	// module's Init becomes a no-op. 60s is a reasonable default.
	Interval time.Duration
	// ProbeOnRegister, when true, runs an initial probe sweep as soon as
	// Init returns. Default: true.
	ProbeOnRegister bool
}

// Module runs the periodic probe loop.
type Module struct {
	cfg    Config
	kit    *brainkit.Kit
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
func (Factory) Build(ctx brainkit.ModuleContext) (brainkit.Module, error) {
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
func (Factory) Describe() brainkit.ModuleDescriptor {
	return brainkit.ModuleDescriptor{
		Name:    "probes",
		Status:  brainkit.ModuleStatusBeta,
		Summary: "Periodic health probes of providers, vector stores, and storages.",
	}
}

func init() { brainkit.RegisterModule("probes", Factory{}) }

// Name reports the module identifier.
func (m *Module) Name() string { return "probes" }

// Status reports maturity.
func (m *Module) Status() brainkit.ModuleStatus { return brainkit.ModuleStatusBeta }

// Init kicks off the periodic probe loop (if Interval > 0) and optionally
// runs an initial sweep.
func (m *Module) Init(k *brainkit.Kit) error {
	m.kit = k

	probeOnRegister := m.cfg.ProbeOnRegister
	// Default to true when the user didn't explicitly pick a value.
	if !probeOnRegister {
		// Allow explicit opt-out by leaving ProbeOnRegister=false; but because
		// Go zero values make "unset" indistinguishable from "false", treat
		// zero as a signal to probe once.
		probeOnRegister = true
	}
	if probeOnRegister {
		go k.ProbeAll()
	}

	if m.cfg.Interval > 0 {
		go m.loop()
	}
	return nil
}

// Close stops the periodic loop.
func (m *Module) Close() error {
	if m.closed.CompareAndSwap(false, true) {
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
			m.kit.ProbeAll()
		case <-m.stop:
			return
		}
	}
}

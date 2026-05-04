package harness

import (
	"context"
	"fmt"
	"sync"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	harnesscap "github.com/brainlet/brainkit/modulecap/harness"
)

// Module is the bkmodule.Module wrapper around the harness Instance.
// Marked WIP — the Harness surface is in flux while multi-consumer
// validation catches up; only the Instance interface declared in
// instance.go is frozen.
type Module struct {
	cfg      Config
	mu       sync.RWMutex
	instance *Harness
}

// Config builds the harness Module. Wraps HarnessConfig so additional
// module-level knobs can be added later without breaking the inner
// Harness constructor.
type Config struct {
	Harness HarnessConfig
}

// NewModule builds the harness Module. Pass it to brainkit.Config.Modules.
// Mount creates the inner Harness when the Kit boots.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) ID() string              { return "harness" }
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusWIP }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	harnessRuntime, err := bkmodule.RequireCapability[harnesscap.Runtime](host, bkmodule.CapabilityHarnessRuntime)
	if err != nil {
		return fmt.Errorf("harness: %w", err)
	}
	if err := m.start(harnessRuntime); err != nil {
		return err
	}
	m.mu.RLock()
	instanceAttached := m.instance != nil
	m.mu.RUnlock()
	if instanceAttached {
		host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindRuntime, "harness.instance", "Experimental harness instance."))
	}
	host.Scope().Defer(m.CloseContext)
	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	if lifecycleDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "harness", func() any {
			return m.DebugSnapshot()
		})
		if err != nil {
			return err
		}
		host.Scope().Defer(handle.Close)
	}
	return nil
}

func (m *Module) start(rt Runtime) error {
	if rt == nil {
		return nil // Harness cannot run without a JS runtime.
	}
	h, err := Init(rt, m.cfg.Harness)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.instance = h
	m.mu.Unlock()
	return nil
}

// Close shuts down the inner Harness.
func (m *Module) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.CloseContext(ctx)
}

// CloseContext shuts down the inner Harness under caller-owned lifecycle
// cancellation. If shutdown times out, the instance stays attached so a later
// close can retry the wait.
func (m *Module) CloseContext(ctx context.Context) error {
	m.mu.RLock()
	instance := m.instance
	m.mu.RUnlock()
	if instance == nil {
		return nil
	}
	err := instance.CloseContext(ctx)
	if err == nil {
		m.mu.Lock()
		if m.instance == instance {
			m.instance = nil
		}
		m.mu.Unlock()
	}
	return err
}

// Instance returns the Harness as the frozen Instance surface.
// Returns nil when Init hasn't produced a Harness yet (e.g. when the
// Kit is built without a JS runtime).
func (m *Module) Instance() Instance {
	m.mu.RLock()
	instance := m.instance
	m.mu.RUnlock()
	if instance == nil {
		return nil
	}
	return (*instanceAdapter)(instance)
}

// YAML is the config shape decoded by the registry factory. The
// harness surface is rich (Modes, Subagents, StateSchema, …) — the
// YAML shape exposes only the simple scalar knobs and leaves richer
// configuration to code via `brainkit.Config.Modules`. The factory is
// primarily useful as a "include harness, use defaults" switch.
type YAML struct {
	ID         string   `yaml:"id"`
	ResourceID string   `yaml:"resource_id"`
	Tools      []string `yaml:"tools"`
}

// Factory is the registered ModuleFactory for harness.
type Factory struct{}

// Build decodes YAML and returns a harness module with minimal
// config. Advanced configuration (Modes, Subagents, StateSchema,
// etc.) remains programmatic — wire those via code in a custom
// binary instead of YAML.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return NewModule(Config{Harness: HarnessConfig{
		ID:         y.ID,
		ResourceID: y.ResourceID,
		Tools:      y.Tools,
	}}), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "harness",
		Status:  bkmodule.StatusWIP,
		Summary: "Experimental multi-mode JS harness (tools, subagents, state).",
		Requires: []string{
			"jsruntime",
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[harnesscap.Runtime](bkmodule.CapabilityHarnessRuntime),
			bkmodule.OptionalCapabilityOf[bkmodule.LifecycleDebugRegistry](bkmodule.CapabilityLifecycleDebugRegistry),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindRuntime, "harness.instance", "Experimental harness instance."),
		},
	}
}

func init() { bkmodule.Register("harness", Factory{}) }

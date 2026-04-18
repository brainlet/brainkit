package harness

import (
	"fmt"

	"github.com/brainlet/brainkit"
)

// Module is the brainkit.Module wrapper around the harness Instance.
// Marked WIP — the Harness surface is in flux while multi-consumer
// validation catches up; only the Instance interface declared in
// instance.go is frozen.
type Module struct {
	cfg      Config
	instance *Harness
}

// Config builds the harness Module. Wraps HarnessConfig so additional
// module-level knobs can be added later without breaking the inner
// Harness constructor.
type Config struct {
	Harness HarnessConfig
}

// NewModule builds the harness Module. Pass it to brainkit.Config.Modules.
// Init creates the inner Harness when the Kit boots.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) Name() string                   { return "harness" }
func (m *Module) Status() brainkit.ModuleStatus  { return brainkit.ModuleStatusWIP }

// Init constructs the underlying Harness from the Kit's JS runtime.
// Harness needs a Runtime (BridgeEval access); the Kit's bridge
// satisfies it via brainkit.HarnessRuntime.
func (m *Module) Init(k *brainkit.Kit) error {
	raw := k.HarnessRuntime()
	if raw == nil {
		return nil // Harness cannot run without a JS runtime.
	}
	rt, ok := raw.(Runtime)
	if !ok {
		return fmt.Errorf("harness: Kit.HarnessRuntime() returned %T which does not satisfy harness.Runtime", raw)
	}
	h, err := Init(rt, m.cfg.Harness)
	if err != nil {
		return err
	}
	m.instance = h
	return nil
}

// Close shuts down the inner Harness.
func (m *Module) Close() error {
	if m.instance == nil {
		return nil
	}
	return m.instance.Close()
}

// Instance returns the Harness as the frozen Instance surface.
// Returns nil when Init hasn't produced a Harness yet (e.g. when the
// Kit is built without a JS runtime).
func (m *Module) Instance() Instance {
	if m.instance == nil {
		return nil
	}
	return (*instanceAdapter)(m.instance)
}

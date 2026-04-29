// Package health owns the kit.health command as a hot-mountable Kit module.
package health

import (
	"context"
	"encoding/json"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
)

// Module exposes kit.health. Construct via New and include in
// brainkit.Config.Modules when the runtime should expose health over the bus.
type Module struct {
	snapshot func(context.Context) any
}

// New creates the health module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "health" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers kit.health.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	snapshot, err := bkmodule.RequireCapability[func(context.Context) any](host, bkmodule.CapabilityHealthSnapshot)
	if err != nil {
		return fmt.Errorf("health: %w", err)
	}
	m.snapshot = snapshot
	host.Scope().Defer(func(context.Context) error {
		m.snapshot = nil
		return nil
	})
	if _, err := host.Commands().Handle(bkmodule.Command(m.Get)); err != nil {
		return err
	}
	return nil
}

// Close detaches the module from Kit capabilities.
func (m *Module) Close() error {
	m.snapshot = nil
	return nil
}

// Factory is the registered ModuleFactory for health.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the health module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "health",
		Status:  bkmodule.StatusStable,
		Summary: "Runtime health bus command (kit.health).",
	}
}

func init() { bkmodule.Register("health", Factory{}) }

// Get handles kit.health.
func (m *Module) Get(ctx context.Context, _ KitHealthMsg) (*KitHealthResp, error) {
	if m.snapshot == nil {
		return nil, fmt.Errorf("health: module is not mounted")
	}
	data, _ := json.Marshal(m.snapshot(ctx))
	return &KitHealthResp{Health: data}, nil
}

// Package metrics owns the metrics.get command as a hot-mountable Kit module.
package metrics

import (
	"context"
	"encoding/json"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
)

// Module exposes metrics.get. Construct via New and include in
// brainkit.Config.Modules when the runtime should expose bus metrics.
type Module struct {
	snapshot func() any
}

// New creates the metrics module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "metrics" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers metrics.get.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	snapshot, err := bkmodule.RequireCapability[func() any](host, bkmodule.CapabilityMetricsSnapshot)
	if err != nil {
		return fmt.Errorf("metrics: %w", err)
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

// Factory is the registered ModuleFactory for metrics.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the metrics module.
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
		Name:    "metrics",
		Status:  bkmodule.StatusStable,
		Summary: "Runtime metrics bus command (metrics.get).",
	}
}

func init() { bkmodule.Register("metrics", Factory{}) }

// Get handles metrics.get.
func (m *Module) Get(_ context.Context, _ MetricsGetMsg) (*MetricsGetResp, error) {
	if m.snapshot == nil {
		return nil, fmt.Errorf("metrics: module is not mounted")
	}
	data, _ := json.Marshal(m.snapshot())
	return &MetricsGetResp{Metrics: data}, nil
}

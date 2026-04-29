// Package jsruntime owns hot-mount activation of the embedded JS/TS runtime.
package jsruntime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/engine"
	runtimejs "github.com/brainlet/brainkit/internal/jsruntime"
	bkmodule "github.com/brainlet/brainkit/module"
)

// Module enables the embedded JS/TS runtime on mount.
type Module struct{}

// New creates the JS runtime module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "jsruntime" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

// Mount starts the embedded JS/TS runtime if it is not already active.
func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	kernel, err := bkmodule.RequireCapability[*engine.Kernel](host, bkmodule.CapabilityKernel)
	if err != nil {
		return fmt.Errorf("jsruntime: %w", err)
	}
	if err := runtimejs.Enable(ctx, kernel); err != nil {
		return err
	}
	for name, value := range map[string]any{
		bkmodule.CapabilityEnableJSRuntime: func(ctx context.Context) error {
			return runtimejs.Enable(ctx, kernel)
		},
		bkmodule.CapabilityHasJSRuntime: func() bool {
			return kernel.HasJSRuntime()
		},
		bkmodule.CapabilityDeployer:    engine.Deployer(kernel),
		bkmodule.CapabilityTSRunner:    engine.TSRunner(kernel),
		bkmodule.CapabilityEvalRuntime: kernel,
		bkmodule.CapabilityCallJS: func(ctx context.Context, fn string, args any) (json.RawMessage, error) {
			return kernel.CallJS(ctx, fn, args)
		},
		bkmodule.CapabilityHarnessRuntime: func() any {
			return kernel.HarnessRuntime()
		},
	} {
		if _, err := host.Capabilities().Provide(ctx, name, value); err != nil {
			return err
		}
	}
	return nil
}

// Close leaves the runtime running. The kernel owns runtime shutdown; unmounting
// this module only releases the module lease, not the shared JS heap.
func (m *Module) Close() error { return nil }

// Factory is the registered ModuleFactory for jsruntime.
type Factory struct{}

// YAML is reserved for future runtime options.
type YAML struct{}

// Build decodes YAML and returns the module.
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
		Name:    "jsruntime",
		Status:  bkmodule.StatusBeta,
		Summary: "Embedded JS/TS runtime activation.",
	}
}

func init() { bkmodule.Register("jsruntime", Factory{}) }

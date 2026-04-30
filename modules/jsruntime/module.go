// Package jsruntime owns hot-mount activation of the embedded JS/TS runtime.
package jsruntime

import (
	"context"
	"encoding/json"
	"fmt"

	runtimejs "github.com/brainlet/brainkit/internal/jsruntime"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modulecap/runtime"
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
	runtimeHost, err := bkmodule.RequireCapability[runtimecap.Host](host, bkmodule.CapabilityJSRuntimeHost)
	if err != nil {
		return fmt.Errorf("jsruntime: %w", err)
	}
	if err := runtimejs.Enable(ctx, runtimeHost); err != nil {
		return err
	}
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindRuntime, "jsruntime.heap", "Embedded JS/TS runtime activation lease."))
	for name, value := range map[string]any{
		bkmodule.CapabilityEnableJSRuntime: func(ctx context.Context) error {
			return runtimejs.Enable(ctx, runtimeHost)
		},
		bkmodule.CapabilityHasJSRuntime: func() bool {
			return runtimeHost.HasJSRuntime()
		},
		bkmodule.CapabilityDeployer:    runtimecap.Deployer(runtimeHost),
		bkmodule.CapabilityTSRunner:    runtimecap.TSRunner(runtimeHost),
		bkmodule.CapabilityEvalRuntime: runtimecap.EvalRuntime(runtimeHost),
		bkmodule.CapabilityCallJS: func(ctx context.Context, fn string, args any) (json.RawMessage, error) {
			return runtimeHost.CallJS(ctx, fn, args)
		},
		bkmodule.CapabilityHarnessRuntime: func() any {
			return runtimeHost.HarnessRuntime()
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
		Provides: []string{
			"jsruntime",
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.ProvidedCapabilityOf[func(context.Context) error](bkmodule.CapabilityEnableJSRuntime),
			bkmodule.ProvidedCapabilityOf[func() bool](bkmodule.CapabilityHasJSRuntime),
			bkmodule.ProvidedCapabilityOf[runtimecap.Deployer](bkmodule.CapabilityDeployer),
			bkmodule.ProvidedCapabilityOf[runtimecap.TSRunner](bkmodule.CapabilityTSRunner),
			bkmodule.ProvidedCapabilityOf[runtimecap.EvalRuntime](bkmodule.CapabilityEvalRuntime),
			bkmodule.ProvidedCapabilityOf[func(context.Context, string, any) (json.RawMessage, error)](bkmodule.CapabilityCallJS),
			bkmodule.ProvidedCapabilityOf[func() any](bkmodule.CapabilityHarnessRuntime),
			bkmodule.RequiredCapabilityOf[runtimecap.Host](bkmodule.CapabilityJSRuntimeHost),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindRuntime, "jsruntime.heap", "Embedded JS/TS runtime activation lease."),
		},
	}
}

func init() { bkmodule.Register("jsruntime", Factory{}) }

// Package jsruntime owns hot-mount activation of the embedded JavaScript runtime.
package jsruntime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/embed/typescript"
	runtimejs "github.com/brainlet/brainkit/internal/jsruntime"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	harnesscap "github.com/brainlet/brainkit/modulecap/harness"
	"github.com/brainlet/brainkit/modulecap/runtime"
)

// Module enables the embedded JavaScript runtime on mount.
type Module struct{}

type artifactDeployer struct {
	deployer runtimecap.SourceDeployer
}

type evalRuntime struct {
	runtime interface {
		runtimecap.SourceDeployer
		runtimecap.JSRunner
		EvalModule(ctx context.Context, source, code string) (string, error)
	}
}

type testingRuntime struct {
	artifact runtimecap.ArtifactDeployer
	js       runtimecap.JSRunner
}

func prepareTypeScriptSource(source, code string) (string, error) {
	return typescript.TranspileTS(code, source)
}

func (d artifactDeployer) DeployArtifact(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	opts = append(opts, types.WithNormalizedJS())
	return d.deployer.Deploy(ctx, source, code, opts...)
}

func (d artifactDeployer) Teardown(ctx context.Context, source string) (int, error) {
	return d.deployer.Teardown(ctx, source)
}

func (d artifactDeployer) ListDeployments() []runtimecap.DeploymentInfo {
	return d.deployer.ListDeployments()
}

func (d artifactDeployer) ResourcesFrom(source string) ([]types.ResourceInfo, error) {
	type resourcer interface {
		ResourcesFrom(source string) ([]types.ResourceInfo, error)
	}
	if r, ok := d.deployer.(resourcer); ok {
		return r.ResourcesFrom(source)
	}
	return nil, nil
}

func (r evalRuntime) EvalJS(ctx context.Context, source, code string) (string, error) {
	return r.runtime.EvalJS(ctx, source, code)
}

func (r evalRuntime) EvalModule(ctx context.Context, source, code string) (string, error) {
	return r.runtime.EvalModule(ctx, source, code)
}

func (r evalRuntime) EvalScript(ctx context.Context, source, code string) (string, error) {
	if _, err := r.runtime.Deploy(ctx, source, code); err != nil {
		return "", err
	}
	defer r.runtime.Teardown(ctx, source)
	return r.runtime.EvalJS(ctx, "__read_eval.js", `return globalThis.__module_result || "null";`)
}

func (r testingRuntime) EvalJS(ctx context.Context, source, code string) (string, error) {
	return r.js.EvalJS(ctx, source, code)
}

func (r testingRuntime) DeployArtifact(ctx context.Context, source, code string) ([]types.ResourceInfo, error) {
	return r.artifact.DeployArtifact(ctx, source, code)
}

func (r testingRuntime) Teardown(ctx context.Context, source string) (int, error) {
	return r.artifact.Teardown(ctx, source)
}

// New creates the JS runtime module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "jsruntime" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

// Mount starts the embedded JavaScript runtime if it is not already active.
func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	runtimeHost, err := bkmodule.RequireCapability[runtimecap.Host](host, bkmodule.CapabilityJSRuntimeHost)
	if err != nil {
		return fmt.Errorf("jsruntime: %w", err)
	}
	if err := runtimejs.Enable(ctx, runtimeHost, runtimejs.WithSourcePreparer(prepareTypeScriptSource)); err != nil {
		return err
	}
	host.Scope().Defer(func(ctx context.Context) error {
		return runtimeHost.DisableJSRuntime(ctx)
	})
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindRuntime, "jsruntime.heap", "Embedded JavaScript runtime activation lease."))
	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	runtimeDebug, _ := runtimeHost.(runtimecap.DebugSnapshotter)
	if lifecycleDebug != nil && runtimeDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "jsruntime", func() any {
			return runtimeDebug.JSRuntimeDebugSnapshot()
		})
		if err != nil {
			return fmt.Errorf("jsruntime: lifecycle debug: %w", err)
		}
		host.Scope().Defer(handle.Close)
	}
	harnessRuntime, ok := runtimeHost.HarnessRuntime().(harnesscap.Runtime)
	if !ok || harnessRuntime == nil {
		return fmt.Errorf("jsruntime: harness runtime is unavailable")
	}
	artifact := artifactDeployer{deployer: runtimeHost}
	for name, value := range map[string]any{
		bkmodule.CapabilityEnableJSRuntime: func(ctx context.Context) error {
			return runtimejs.Enable(ctx, runtimeHost, runtimejs.WithSourcePreparer(prepareTypeScriptSource))
		},
		bkmodule.CapabilityHasJSRuntime: func() bool {
			return runtimeHost.HasJSRuntime()
		},
		bkmodule.CapabilityArtifactDeployer: artifact,
		bkmodule.CapabilityEvalRuntime:      evalRuntime{runtime: runtimeHost},
		bkmodule.CapabilityTestRuntime: testingRuntime{
			artifact: artifact,
			js:       runtimeHost,
		},
		bkmodule.CapabilityCallJS: func(ctx context.Context, fn string, args any) (json.RawMessage, error) {
			return runtimeHost.CallJS(ctx, fn, args)
		},
		bkmodule.CapabilityHarnessRuntime: harnessRuntime,
	} {
		if _, err := host.Capabilities().Provide(ctx, name, value); err != nil {
			return err
		}
	}
	return nil
}

// Close is a no-op because runtime teardown is owned by the module scope defer
// registered during Mount.
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

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "jsruntime",
		Status:  bkmodule.StatusBeta,
		Summary: "Embedded JavaScript runtime activation.",
		Provides: []string{
			"jsruntime",
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.ProvidedCapabilityOf[func(context.Context) error](bkmodule.CapabilityEnableJSRuntime),
			bkmodule.ProvidedCapabilityOf[func() bool](bkmodule.CapabilityHasJSRuntime),
			bkmodule.ProvidedCapabilityOf[runtimecap.ArtifactDeployer](bkmodule.CapabilityArtifactDeployer),
			bkmodule.ProvidedCapabilityOf[runtimecap.EvalRuntime](bkmodule.CapabilityEvalRuntime),
			bkmodule.ProvidedCapabilityOf[runtimecap.TestRuntime](bkmodule.CapabilityTestRuntime),
			bkmodule.ProvidedCapabilityOf[func(context.Context, string, any) (json.RawMessage, error)](bkmodule.CapabilityCallJS),
			bkmodule.ProvidedCapabilityOf[harnesscap.Runtime](bkmodule.CapabilityHarnessRuntime),
			bkmodule.RequiredCapabilityOf[runtimecap.Host](bkmodule.CapabilityJSRuntimeHost),
			bkmodule.OptionalCapabilityOf[bkmodule.LifecycleDebugRegistry](bkmodule.CapabilityLifecycleDebugRegistry),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindRuntime, "jsruntime.heap", "Embedded JavaScript runtime activation lease."),
		},
	}
}

func init() { bkmodule.Register("jsruntime", Factory{}) }

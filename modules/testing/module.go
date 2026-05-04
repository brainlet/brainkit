// Package testing owns the test.run command as a hot-mountable Kit module.
package testing

import (
	"context"
	"encoding/json"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modulecap/runtime"
	braintest "github.com/brainlet/brainkit/modules/testing/internal/braintest"
	"github.com/brainlet/brainkit/modules/testing/testingmsg"
)

// Module exposes test.run. Construct via New and include in
// brainkit.Config.Modules when the runtime should execute .test.ts suites.
type Module struct {
	runtime runtimecap.TestRuntime
}

// New creates the testing module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "testing" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

// Mount registers test.run.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	runtime, err := bkmodule.RequireCapability[runtimecap.TestRuntime](host, bkmodule.CapabilityTestRuntime)
	if err != nil {
		return fmt.Errorf("testing: %w", err)
	}
	m.runtime = runtime
	host.Scope().Defer(func(context.Context) error {
		m.runtime = nil
		return nil
	})
	if _, err := host.Commands().Handle(bkmodule.Command(m.Run)); err != nil {
		return err
	}
	return nil
}

// Close detaches the module from the Kit capabilities.
func (m *Module) Close() error {
	m.runtime = nil
	return nil
}

// Factory is the registered ModuleFactory for testing.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the testing module.
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
		Name:    "testing",
		Status:  bkmodule.StatusBeta,
		Summary: "Runs .test.ts suites through the test.run bus command.",
		Requires: []string{
			"jsruntime",
		},
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[testingmsg.TestRunMsg, testingmsg.TestRunResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[runtimecap.TestRuntime](bkmodule.CapabilityTestRuntime),
		},
	}
}

func init() { bkmodule.Register("testing", Factory{}) }

// testRuntime adapts runtime capabilities to braintest.Runtime.
type testRuntime struct {
	runtime runtimecap.TestRuntime
}

func (r *testRuntime) EvalTS(ctx context.Context, source, code string) (string, error) {
	return r.runtime.EvalTS(ctx, source, code)
}

func (r *testRuntime) Deploy(ctx context.Context, source, code string, kind braintest.DeployKind) error {
	if kind == braintest.DeployNormalizedJS {
		_, err := r.runtime.DeployArtifact(ctx, source, code)
		return err
	}
	_, err := r.runtime.DeploySource(ctx, source, code)
	return err
}

func (r *testRuntime) Teardown(ctx context.Context, source string) error {
	_, err := r.runtime.Teardown(ctx, source)
	return err
}

// Run handles test.run.
func (m *Module) Run(ctx context.Context, req testingmsg.TestRunMsg) (*testingmsg.TestRunResp, error) {
	if m.runtime == nil {
		return nil, fmt.Errorf("testing: runtime is not configured")
	}
	runner := braintest.NewTestRunner(&testRuntime{runtime: m.runtime}, braintest.TestRunnerConfig{
		TestDir: req.Dir,
		Pattern: req.Pattern,
		SkipAI:  req.SkipAI,
	})

	result, err := runner.Run(ctx)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(result)
	return &testingmsg.TestRunResp{Results: data}, nil
}

package brainkit

import (
	"context"
	"encoding/json"

	braintest "github.com/brainlet/brainkit/internal/braintest"
	"github.com/brainlet/brainkit/sdk/messages"
)

// TestingDomain handles test.run bus commands.
type TestingDomain struct {
	kit *Kernel
}

func newTestingDomain(k *Kernel) *TestingDomain {
	return &TestingDomain{kit: k}
}

// kernelTestRuntime adapts Kernel to the testing.Runtime interface.
type kernelTestRuntime struct {
	kernel *Kernel
}

func (r *kernelTestRuntime) EvalTS(ctx context.Context, source, code string) (string, error) {
	return r.kernel.EvalTS(ctx, source, code)
}

func (r *kernelTestRuntime) Deploy(ctx context.Context, source, code string) error {
	_, err := r.kernel.Deploy(ctx, source, code)
	return err
}

func (r *kernelTestRuntime) Teardown(ctx context.Context, source string) error {
	_, err := r.kernel.Teardown(ctx, source)
	return err
}

func (d *TestingDomain) Run(ctx context.Context, req messages.TestRunMsg) (*messages.TestRunResp, error) {
	runner := braintest.NewTestRunner(&kernelTestRuntime{kernel: d.kit}, braintest.TestRunnerConfig{
		TestDir: req.Dir,
		Pattern: req.Pattern,
	})

	result, err := runner.Run(ctx)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(result)
	return &messages.TestRunResp{Results: data}, nil
}

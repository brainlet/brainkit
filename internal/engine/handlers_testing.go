package engine

import (
	"context"
	"encoding/json"

	braintest "github.com/brainlet/brainkit/internal/braintest"
	"github.com/brainlet/brainkit/sdk"
)

// TestingDomain handles test.run bus commands.
type TestingDomain struct {
	deployer Deployer
	tsRunner TSRunner
}

func newTestingDomain(deployer Deployer, tsRunner TSRunner) *TestingDomain {
	return &TestingDomain{deployer: deployer, tsRunner: tsRunner}
}

// testRuntime adapts Deployer + TSRunner to the braintest.Runtime interface.
type testRuntime struct {
	deployer Deployer
	tsRunner TSRunner
}

func (r *testRuntime) EvalTS(ctx context.Context, source, code string) (string, error) {
	return r.tsRunner.EvalTS(ctx, source, code)
}

func (r *testRuntime) Deploy(ctx context.Context, source, code string) error {
	_, err := r.deployer.Deploy(ctx, source, code)
	return err
}

func (r *testRuntime) Teardown(ctx context.Context, source string) error {
	_, err := r.deployer.Teardown(ctx, source)
	return err
}

func (d *TestingDomain) Run(ctx context.Context, req sdk.TestRunMsg) (*sdk.TestRunResp, error) {
	runner := braintest.NewTestRunner(&testRuntime{deployer: d.deployer, tsRunner: d.tsRunner}, braintest.TestRunnerConfig{
		TestDir: req.Dir,
		Pattern: req.Pattern,
	})

	result, err := runner.Run(ctx)
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(result)
	return &sdk.TestRunResp{Results: data}, nil
}

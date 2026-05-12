package testing

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit/internal/types"
)

type testingRecordingRuntime struct {
	cfg types.DeployConfig
}

func (r *testingRecordingRuntime) EvalJS(context.Context, string, string) (string, error) {
	return "", nil
}

func (r *testingRecordingRuntime) DeployArtifact(_ context.Context, _ string, _ string) ([]types.ResourceInfo, error) {
	var cfg types.DeployConfig
	types.WithNormalizedJS()(&cfg)
	r.cfg = cfg
	return nil, nil
}

func (r *testingRecordingRuntime) Teardown(context.Context, string) (int, error) { return 0, nil }

func TestTestRuntimeAlwaysDeploysNormalizedJS(t *testing.T) {
	runtime := &testingRecordingRuntime{}
	rt := &testRuntime{runtime: runtime}

	if err := rt.Deploy(context.Background(), "__test_suite.ts", "test('ok', () => {})"); err != nil {
		t.Fatalf("deploy normalized test: %v", err)
	}
	if runtime.cfg.EffectiveArtifactKind() != types.DeployArtifactNormalizedJS {
		t.Fatalf("bundled test deploy artifact kind = %q, want normalized_js", runtime.cfg.EffectiveArtifactKind())
	}
}

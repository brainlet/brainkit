package braintest

import (
	"context"
	"testing"
)

type deployKindRuntime struct {
	deployKinds []DeployKind
}

func (r *deployKindRuntime) EvalTS(context.Context, string, string) (string, error) {
	return "[]", nil
}

func (r *deployKindRuntime) Deploy(_ context.Context, _ string, _ string, kind DeployKind) error {
	r.deployKinds = append(r.deployKinds, kind)
	return nil
}

func (r *deployKindRuntime) Teardown(context.Context, string) error { return nil }

func TestRunCodeDeploysRawSource(t *testing.T) {
	rt := &deployKindRuntime{}
	runner := NewTestRunner(rt, TestRunnerConfig{})

	if _, err := runner.RunCode(context.Background(), "inline.ts", "const x: number = 1;"); err != nil {
		t.Fatalf("run code: %v", err)
	}
	if len(rt.deployKinds) != 1 || rt.deployKinds[0] != DeploySource {
		t.Fatalf("deploy kinds = %#v, want [source]", rt.deployKinds)
	}
}

func TestExecuteBundledTestCodeDeploysNormalizedJS(t *testing.T) {
	rt := &deployKindRuntime{}
	runner := NewTestRunner(rt, TestRunnerConfig{})

	if _, err := runner.executeTestCode(context.Background(), "suite.test.ts", "test('ok', () => {});", DeployNormalizedJS); err != nil {
		t.Fatalf("execute test code: %v", err)
	}
	if len(rt.deployKinds) != 1 || rt.deployKinds[0] != DeployNormalizedJS {
		t.Fatalf("deploy kinds = %#v, want [normalized_js]", rt.deployKinds)
	}
}

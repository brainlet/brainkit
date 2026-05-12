package braintest

import (
	"context"
	"strings"
	"testing"
)

type deployKindRuntime struct {
	deployed []string
}

func (r *deployKindRuntime) EvalJS(context.Context, string, string) (string, error) {
	return "[]", nil
}

func (r *deployKindRuntime) Deploy(_ context.Context, _ string, code string) error {
	r.deployed = append(r.deployed, code)
	return nil
}

func (r *deployKindRuntime) Teardown(context.Context, string) error { return nil }

func TestRunCodeNormalizesInlineTypeScript(t *testing.T) {
	rt := &deployKindRuntime{}
	runner := NewTestRunner(rt, TestRunnerConfig{})

	if _, err := runner.RunCode(context.Background(), "inline.ts", "const x: number = 1;"); err != nil {
		t.Fatalf("run code: %v", err)
	}
	if len(rt.deployed) != 1 {
		t.Fatalf("deployed code count = %d, want 1", len(rt.deployed))
	}
	if strings.Contains(rt.deployed[0], ": number") {
		t.Fatalf("deployed code still contains TypeScript annotation: %s", rt.deployed[0])
	}
}

func TestExecuteBundledTestCodeDeploysJavaScript(t *testing.T) {
	rt := &deployKindRuntime{}
	runner := NewTestRunner(rt, TestRunnerConfig{})

	if _, err := runner.executeTestCode(context.Background(), "suite.test.ts", "test('ok', () => {});"); err != nil {
		t.Fatalf("execute test code: %v", err)
	}
	if len(rt.deployed) != 1 || rt.deployed[0] == "" {
		t.Fatalf("deployed code = %#v, want one non-empty artifact", rt.deployed)
	}
}

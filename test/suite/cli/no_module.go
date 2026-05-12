package cli

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	"github.com/brainlet/brainkit/test/suite"
)

// testKitEvalNoModuleCommandsAbsent verifies that kit.eval is only available
// when modules/eval is wired.
func testKitEvalNoModuleCommandsAbsent(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-eval-no-module",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := evalmsg.CallKitEval(k, ctx, evalmsg.KitEvalMsg{Mode: "js", Code: `return "ok"`}); err == nil {
		t.Fatal("expected kit.eval to fail without eval module")
	}
}

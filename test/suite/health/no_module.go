package health

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	healthmod "github.com/brainlet/brainkit/modules/health"
	"github.com/brainlet/brainkit/test/suite"
)

// testNoModuleCommandsAbsent verifies that kit.health is only available as a
// bus command when modules/health is mounted. The root kernel can still compute
// health internally for direct callers and gateway checks.
func testNoModuleCommandsAbsent(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-health-no-module",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := healthmod.CallKitHealth(k, ctx, healthmod.KitHealthMsg{}); err == nil {
		t.Fatal("expected kit.health to fail without health module")
	}
}

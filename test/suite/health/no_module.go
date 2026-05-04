package health

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	healthmod "github.com/brainlet/brainkit/modules/health"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
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

	pr, err := protocol.Publish(k, ctx, healthmod.KitHealthMsg{})
	if err != nil {
		t.Fatalf("publish kit.health: %v", err)
	}

	ch := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	select {
	case m := <-ch:
		t.Fatalf("expected no reply for kit.health without health module, got payload=%s", string(m.Payload))
	case <-ctx.Done():
		// Expected: no command handler registered.
	}
}

package secrets

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
)

// testNoModuleCommandsAbsent verifies that secrets.* bus commands are only
// available when modules/secrets is wired.
func testNoModuleCommandsAbsent(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-secrets-no-module",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pr, err := protocol.Publish(k, ctx, secretmsg.SecretsListMsg{})
	if err != nil {
		t.Fatalf("publish secrets.list: %v", err)
	}

	ch := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	select {
	case m := <-ch:
		t.Fatalf("expected no reply for secrets.list without secrets module, got payload=%s", string(m.Payload))
	case <-ctx.Done():
		// Expected: no command handler registered.
	}
}

package cli

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/google/uuid"
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

	replyTo := "kit.eval.reply.no-module." + uuid.NewString()
	ch := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, replyTo, func(m sdk.Message) { ch <- m })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	_, err = protocol.Publish(k, ctx, evalmsg.KitEvalMsg{Mode: "ts", Code: `return "ok"`}, protocol.WithReplyTo(replyTo))
	if err != nil {
		t.Fatalf("publish kit.eval: %v", err)
	}

	select {
	case m := <-ch:
		t.Fatalf("expected no reply for kit.eval without eval module, got payload=%s", string(m.Payload))
	case <-ctx.Done():
		// Expected: no command handler registered.
	}
}

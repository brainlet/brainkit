package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	messagingmod "github.com/brainlet/brainkit/modules/messaging"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/google/uuid"
)

// testMessagingNoModuleCommandsAbsent verifies that kit.send is only
// available when modules/messaging is wired. Raw publish/subscribe still works
// without the module; only the command bridge is absent.
func testMessagingNoModuleCommandsAbsent(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-messaging-no-module",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	replyTo := "kit.send.reply.no-module." + uuid.NewString()
	ch := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, replyTo, func(m sdk.Message) { ch <- m })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	_, err = protocol.Publish(k, ctx, messagingmod.KitSendMsg{
		Topic:   "missing.service",
		Payload: json.RawMessage(`{}`),
	}, protocol.WithReplyTo(replyTo))
	if err != nil {
		t.Fatalf("publish kit.send: %v", err)
	}

	select {
	case m := <-ch:
		t.Fatalf("expected no reply for kit.send without messaging module, got payload=%s", string(m.Payload))
	case <-ctx.Done():
		// Expected: no command handler registered.
	}
}

package tools

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
)

// testNoModuleCommandsAbsent verifies that tools.* bus commands are only
// available when modules/tools is wired. The root kernel can still keep the
// tool registry for JS registration and direct Go registration.
func testNoModuleCommandsAbsent(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-tools-no-module",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pr, err := protocol.Publish(k, ctx, toolmsg.ToolListMsg{})
	if err != nil {
		t.Fatalf("publish tools.list: %v", err)
	}

	ch := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	select {
	case m := <-ch:
		t.Fatalf("expected no reply for tools.list without tools module, got payload=%s", string(m.Payload))
	case <-ctx.Done():
		// Expected: no command handler registered.
	}
}

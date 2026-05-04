package bus

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/reference/referencemsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/google/uuid"
)

func testReferenceCommands(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	list, err := sdk.Call[referencemsg.KitReferenceListMsg, referencemsg.KitReferenceListResp](env.Kit, ctx, referencemsg.KitReferenceListMsg{})
	if err != nil {
		t.Fatalf("kit.reference.list: %v", err)
	}
	if len(list.References) == 0 {
		t.Fatal("expected reference catalog entries")
	}

	resp, err := sdk.Call[referencemsg.KitReferenceMsg, referencemsg.KitReferenceResp](env.Kit, ctx, referencemsg.KitReferenceMsg{Name: "tool-author"})
	if err != nil {
		t.Fatalf("kit.reference: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("expected non-empty reference content")
	}
}

// testReferenceNoModuleCommandsAbsent verifies that kit.reference commands are
// only available when modules/reference is wired. The root Reference() Go API
// remains available without the module.
func testReferenceNoModuleCommandsAbsent(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-reference-no-module",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	if _, err := brainkit.Reference("tool-author"); err != nil {
		t.Fatalf("root Reference API should remain available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	replyTo := "kit.reference.reply.no-module." + uuid.NewString()
	ch := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, replyTo, func(m sdk.Message) { ch <- m })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer unsub()

	_, err = protocol.Publish(k, ctx, referencemsg.KitReferenceListMsg{}, protocol.WithReplyTo(replyTo))
	if err != nil {
		t.Fatalf("publish kit.reference.list: %v", err)
	}

	select {
	case m := <-ch:
		t.Fatalf("expected no reply for kit.reference.list without reference module, got payload=%s", string(m.Payload))
	case <-ctx.Done():
		// Expected: no command handler registered.
	}
}

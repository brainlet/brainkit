package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/google/uuid"
)

func publishAndWaitMessage(t *testing.T, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (protocol.PublishResult, sdk.Message, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	replyTo := msg.BusTopic() + ".reply." + uuid.NewString()
	ch := make(chan sdk.Message, 1)
	unsub, err := rt.SubscribeRaw(ctx, replyTo, func(m sdk.Message) {
		select {
		case ch <- m:
		default:
		}
	})
	if err != nil {
		t.Logf("subscribe %s: %v", replyTo, err)
		return protocol.PublishResult{}, sdk.Message{}, false
	}
	defer unsub()

	pr, err := protocol.Publish(rt, ctx, msg, protocol.WithReplyTo(replyTo))
	if err != nil {
		t.Logf("publish %s: %v", msg.BusTopic(), err)
		return protocol.PublishResult{}, sdk.Message{}, false
	}

	select {
	case got := <-ch:
		return pr, got, true
	case <-ctx.Done():
		t.Logf("timeout waiting for %s reply", msg.BusTopic())
		return pr, sdk.Message{}, false
	}
}

func publishAndWaitPayload(t *testing.T, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, bool) {
	t.Helper()
	_, got, ok := publishAndWaitMessage(t, rt, msg, timeout)
	if !ok {
		return nil, false
	}
	return json.RawMessage(got.Payload), true
}

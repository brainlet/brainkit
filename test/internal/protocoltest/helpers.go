package protocoltest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/google/uuid"
)

type PublishResult = protocol.PublishResult

// PublishAndWaitMessageErr publishes a message with a private reply topic and
// returns the raw reply message. It is for envelope/protocol assertions in tests.
func PublishAndWaitMessageErr(t testing.TB, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (PublishResult, sdk.Message, error) {
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
		return PublishResult{}, sdk.Message{}, fmt.Errorf("subscribe %s: %w", replyTo, err)
	}
	defer unsub()

	pr, err := protocol.Publish(rt, ctx, msg, protocol.WithReplyTo(replyTo))
	if err != nil {
		return pr, sdk.Message{}, fmt.Errorf("publish %s: %w", msg.BusTopic(), err)
	}

	select {
	case got := <-ch:
		return pr, got, nil
	case <-ctx.Done():
		return pr, sdk.Message{}, fmt.Errorf("wait for %s reply: %w", msg.BusTopic(), ctx.Err())
	}
}

func PublishAndWaitMessage(t testing.TB, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (PublishResult, sdk.Message, bool) {
	t.Helper()
	pr, got, err := PublishAndWaitMessageErr(t, rt, msg, timeout)
	if err != nil {
		t.Log(err)
		return pr, sdk.Message{}, false
	}
	return pr, got, true
}

func PublishAndWaitPayloadErr(t testing.TB, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, error) {
	t.Helper()
	_, got, err := PublishAndWaitMessageErr(t, rt, msg, timeout)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(got.Payload), nil
}

func PublishAndWaitPayload(t testing.TB, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, bool) {
	t.Helper()
	payload, err := PublishAndWaitPayloadErr(t, rt, msg, timeout)
	if err != nil {
		t.Log(err)
		return nil, false
	}
	return payload, true
}

// PublishAndWaitDecodedErr publishes a message, unwraps a successful envelope
// when present, and decodes the reply into Resp. Error envelopes are returned as
// a zero Resp with the raw sdk.Message so callers can assert the envelope shape.
func PublishAndWaitDecodedErr[Resp any](t testing.TB, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (Resp, sdk.Message, error) {
	t.Helper()
	var resp Resp
	_, got, err := PublishAndWaitMessageErr(t, rt, msg, timeout)
	if err != nil {
		return resp, got, err
	}

	payload := got.Payload
	if got.Metadata["envelope"] == "true" {
		env, err := sdk.DecodeEnvelope(payload)
		if err != nil {
			return resp, got, err
		}
		if !env.Ok {
			return resp, got, nil
		}
		payload = env.Data
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return resp, got, err
	}
	return resp, got, nil
}

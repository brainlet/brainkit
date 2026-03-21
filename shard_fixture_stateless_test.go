package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
	"github.com/stretchr/testify/require"
)

func TestShardFixture_StatelessEcho(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/stateless-echo.ts", "echo")

	require.Equal(t, "stateless", desc.Mode)
	require.Contains(t, desc.Handlers, "test.echo")
	require.Equal(t, "handleEcho", desc.Handlers["test.echo"])

	result := injectEvent(t, kit, "echo", "test.echo", map[string]string{"msg": "hello"})
	require.Equal(t, `{"msg":"hello"}`, result.ReplyPayload)
}

func TestShardFixture_StatelessLogTopic(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-log-topic.ts", "topic-check")

	result := injectEvent(t, kit, "topic-check", "test.topic-check", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"receivedTopic":"test.topic-check"`)
}

func TestShardFixture_StatelessMultiHandler(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/stateless-multi-handler.ts", "multi")

	require.Equal(t, "stateless", desc.Mode)
	require.Len(t, desc.Handlers, 2)
	require.Contains(t, desc.Handlers, "test.ping")
	require.Contains(t, desc.Handlers, "test.pong")

	result := injectEvent(t, kit, "multi", "test.ping", map[string]string{})
	require.Equal(t, `{"handler":"ping"}`, result.ReplyPayload)

	result = injectEvent(t, kit, "multi", "test.pong", map[string]string{})
	require.Equal(t, `{"handler":"pong"}`, result.ReplyPayload)
}

func TestShardFixture_StatelessWildcard(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/stateless-wildcard.ts", "wildcard")

	require.Equal(t, "stateless", desc.Mode)
	require.Contains(t, desc.Handlers, "events.*")

	result := injectEvent(t, kit, "wildcard", "events.order", map[string]string{"id": "123"})
	require.Contains(t, result.ReplyPayload, `"matchedTopic":"events.order"`)
	require.Contains(t, result.ReplyPayload, `"id":"123"`)

	result = injectEvent(t, kit, "wildcard", "events.payment", map[string]string{"id": "456"})
	require.Contains(t, result.ReplyPayload, `"matchedTopic":"events.payment"`)
}

func TestShardFixture_StatelessFireAndForget(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-fire-and-forget.ts", "forwarder")

	received := make(chan string, 1)
	kit.Bus.On("test.forwarded", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- string(msg.Payload)
	})

	result, err := kit.InjectWASMEvent("forwarder", "test.forward", json.RawMessage(`{"data":"forwarded"}`))
	require.NoError(t, err)
	require.Empty(t, result.Error)
	require.Empty(t, result.ReplyPayload)

	select {
	case msg := <-received:
		require.Contains(t, msg, `"data":"forwarded"`)
	case <-time.After(2 * time.Second):
		t.Fatal("forwarded message not received on bus")
	}
}

func TestShardFixture_StatelessReplyJSON(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-reply-json.ts", "transformer")

	result := injectEvent(t, kit, "transformer", "test.transform", map[string]string{"name": "david"})
	require.Contains(t, result.ReplyPayload, `"greeting":"hello david"`)
	require.Contains(t, result.ReplyPayload, `"original"`)
}

func TestShardFixture_StatelessNoReply(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-no-reply.ts", "silent")

	result := injectEvent(t, kit, "silent", "test.silent", map[string]string{"msg": "shh"})
	require.Empty(t, result.ReplyPayload, "handler that doesn't call reply() should have empty reply")
}

func TestShardFixture_StatelessAskAsync(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/echo", ShortName: "echo",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"echoed":true}`), nil
			},
		},
	})

	deployShard(t, kit, "testdata/as/shard/stateless-ask-async.ts", "asker")

	result, err := kit.InjectWASMEvent("asker", "test.ask", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Empty(t, result.Error)
}

package protocol_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/ctxkeys"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type protocolRuntime struct {
	topic   string
	payload json.RawMessage
	ctx     context.Context
}

func (r *protocolRuntime) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	r.ctx = ctx
	r.topic = topic
	r.payload = append(json.RawMessage(nil), payload...)
	return "msg-1", nil
}

func (r *protocolRuntime) SubscribeRaw(context.Context, string, func(sdk.Message)) (func(), error) {
	return func() {}, nil
}

func (r *protocolRuntime) Close() error { return nil }

type protocolMsg struct {
	Value string `json:"value"`
}

func (protocolMsg) BusTopic() string { return "proto.test" }

func TestPublishStampsExplicitReplyMetadata(t *testing.T) {
	rt := &protocolRuntime{}

	result, err := protocol.Publish(rt, context.Background(), protocolMsg{Value: "ok"}, protocol.WithReplyTo("custom.reply"))
	require.NoError(t, err)

	require.Equal(t, "msg-1", result.MessageID)
	require.Equal(t, "proto.test", result.Topic)
	require.Equal(t, "custom.reply", result.ReplyTo)
	require.NotEmpty(t, result.CorrelationID)
	require.Equal(t, "proto.test", rt.topic)
	require.JSONEq(t, `{"value":"ok"}`, string(rt.payload))
	require.Equal(t, result.CorrelationID, rt.ctx.Value(ctxkeys.CorrelationID))
	require.Equal(t, "custom.reply", rt.ctx.Value(ctxkeys.ReplyTo))
}

func TestSendToServiceResolvesServiceTopic(t *testing.T) {
	rt := &protocolRuntime{}

	result, err := protocol.SendToService(rt, context.Background(), "nested/svc.ts", "ask", map[string]string{"q": "hello"})
	require.NoError(t, err)

	require.Equal(t, "ts.nested.svc.ask", result.Topic)
	require.Equal(t, "ts.nested.svc.ask", rt.topic)
	require.Contains(t, result.ReplyTo, "ts.nested.svc.ask.reply.")
	require.JSONEq(t, `{"q":"hello"}`, string(rt.payload))
}

func TestResolveServiceTopic(t *testing.T) {
	tests := []struct {
		service  string
		topic    string
		expected string
	}{
		{"my-agent.ts", "ask", "ts.my-agent.ask"},
		{"my-agent", "ask", "ts.my-agent.ask"},
		{"nested/svc.ts", "rpc", "ts.nested.svc.rpc"},
		{"simple", "ping", "ts.simple.ping"},
	}
	for _, tt := range tests {
		t.Run(tt.service+"/"+tt.topic, func(t *testing.T) {
			assert.Equal(t, tt.expected, protocol.ResolveServiceTopic(tt.service, tt.topic))
		})
	}
}

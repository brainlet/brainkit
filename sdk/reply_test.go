package sdk

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReplier struct {
	lastReplyTo       string
	lastCorrelationID string
	lastPayload       json.RawMessage
	lastDone          bool
}

func (m *mockReplier) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return "msg-id", nil
}

func (m *mockReplier) SubscribeRaw(ctx context.Context, topic string, handler func(Message)) (func(), error) {
	return func() {}, nil
}

func (m *mockReplier) Close() error { return nil }

func (m *mockReplier) ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error {
	m.lastReplyTo = replyTo
	m.lastCorrelationID = correlationID
	m.lastPayload = payload
	m.lastDone = done
	return nil
}

func TestReply(t *testing.T) {
	rt := &mockReplier{}
	msg := Message{
		Metadata: map[string]string{
			"replyTo":       "test.reply.topic",
			"correlationId": "corr-123",
		},
	}

	err := Reply(rt, context.Background(), msg, map[string]bool{"approved": true})
	require.NoError(t, err)
	assert.Equal(t, "test.reply.topic", rt.lastReplyTo)
	assert.Equal(t, "corr-123", rt.lastCorrelationID)
	assert.True(t, rt.lastDone)
	assert.JSONEq(t, `{"approved":true}`, string(rt.lastPayload))
}

func TestSendChunk(t *testing.T) {
	rt := &mockReplier{}
	msg := Message{
		Metadata: map[string]string{
			"replyTo":       "test.reply.topic",
			"correlationId": "corr-456",
		},
	}

	err := SendChunk(rt, context.Background(), msg, map[string]int{"chunk": 1})
	require.NoError(t, err)
	assert.Equal(t, "test.reply.topic", rt.lastReplyTo)
	assert.False(t, rt.lastDone)
}

func TestReply_MissingReplyTo(t *testing.T) {
	rt := &mockReplier{}
	msg := Message{Metadata: map[string]string{}}

	err := Reply(rt, context.Background(), msg, "data")
	assert.ErrorContains(t, err, "no replyTo")
}

func TestReply_NotReplier(t *testing.T) {
	rt := &nonReplierRuntime{}
	msg := Message{
		Metadata: map[string]string{"replyTo": "test.topic"},
	}

	err := Reply(rt, context.Background(), msg, "data")
	assert.ErrorContains(t, err, "does not implement Replier")
}

type nonReplierRuntime struct{}

func (r *nonReplierRuntime) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return "", nil
}
func (r *nonReplierRuntime) SubscribeRaw(ctx context.Context, topic string, handler func(Message)) (func(), error) {
	return func() {}, nil
}
func (r *nonReplierRuntime) Close() error { return nil }

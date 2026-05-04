package sdk_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

type requestCallerMsg struct {
	Value string `json:"value"`
}

func (requestCallerMsg) BusTopic() string { return "demo.request" }

type requestCallerResp struct {
	OK bool `json:"ok"`
}

type requestCallerChunk struct {
	Value int `json:"value"`
}

type fakeRequestCaller struct {
	topic   string
	payload json.RawMessage
	config  sdk.CallerConfig
	reply   json.RawMessage
	chunks  []json.RawMessage
}

func (f *fakeRequestCaller) Call(_ context.Context, topic string, payload json.RawMessage, config sdk.CallerConfig) (json.RawMessage, error) {
	f.topic = topic
	f.payload = append(json.RawMessage(nil), payload...)
	f.config = config
	for _, chunk := range f.chunks {
		if err := config.StreamHandler(sdk.Message{Payload: chunk}); err != nil {
			return nil, err
		}
	}
	return f.reply, nil
}

func TestCallWithCallerUsesNarrowRequestCaller(t *testing.T) {
	caller := &fakeRequestCaller{reply: json.RawMessage(`{"ok":true}`)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := sdk.CallWithCaller[requestCallerMsg, requestCallerResp](
		caller,
		ctx,
		requestCallerMsg{Value: "hello"},
		sdk.WithCallTo("peer-a"),
		sdk.WithCallMeta(map[string]string{"trace": "abc"}),
	)
	require.NoError(t, err)
	require.True(t, resp.OK)
	require.Equal(t, "demo.request", caller.topic)
	require.JSONEq(t, `{"value":"hello"}`, string(caller.payload))
	require.Equal(t, "peer-a", caller.config.TargetNamespace)
	require.Equal(t, map[string]string{"trace": "abc"}, caller.config.Metadata)
	require.Nil(t, caller.config.StreamHandler)
}

func TestCallStreamWithCallerUsesNarrowRequestCaller(t *testing.T) {
	caller := &fakeRequestCaller{
		reply:  json.RawMessage(`{"ok":true}`),
		chunks: []json.RawMessage{json.RawMessage(`{"value":1}`), json.RawMessage(`{"value":2}`)},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var chunks []int
	resp, err := sdk.CallStreamWithCaller[requestCallerMsg, requestCallerChunk, requestCallerResp](
		caller,
		ctx,
		requestCallerMsg{Value: "stream"},
		func(chunk requestCallerChunk) error {
			chunks = append(chunks, chunk.Value)
			return nil
		},
		sdk.WithCallBuffer(7),
		sdk.WithCallBufferPolicy(sdk.BufferDropOldest),
	)
	require.NoError(t, err)
	require.True(t, resp.OK)
	require.Equal(t, []int{1, 2}, chunks)
	require.Equal(t, 7, caller.config.BufferSize)
	require.Equal(t, sdk.BufferDropOldest, caller.config.BufferPolicy)
	require.NotNil(t, caller.config.StreamHandler)
}

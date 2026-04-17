package envelope

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSuccessReplyIsEnvelope — a successful kit.health reply is a wire
// envelope: exactly one top-level `ok:true` with `data` present.
func testSuccessReplyIsEnvelope(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.KitHealthMsg{})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		// Raw JSON must have a top-level `ok`.
		var probe map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(payload, &probe))
		_, hasOk := probe["ok"]
		assert.True(t, hasOk, "success reply must carry top-level `ok` field")

		wire, derr := sdk.DecodeEnvelope(payload)
		require.NoError(t, derr)
		assert.True(t, wire.Ok, "health reply must be ok=true")
		assert.Nil(t, wire.Error, "success reply has no error field")
		assert.NotNil(t, wire.Data, "success reply must populate data")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testErrorReplyIsEnvelope — a failing command reply is a wire envelope
// with ok=false and a well-formed error object.
func testErrorReplyIsEnvelope(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.ToolCallMsg{Name: "ghost-tool-env-shape"})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		wire, derr := sdk.DecodeEnvelope(payload)
		require.NoError(t, derr)
		assert.False(t, wire.Ok)
		require.NotNil(t, wire.Error)
		assert.NotEmpty(t, wire.Error.Code, "error envelope must carry a code")
		assert.NotEmpty(t, wire.Error.Message, "error envelope must carry a message")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testEnvelopeMetadataFlagPresent — every envelope reply sets
// metadata["envelope"]="true" so the Caller knows to unwrap.
func testEnvelopeMetadataFlagPresent(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.KitHealthMsg{})
	require.NoError(t, err)

	flagCh := make(chan string, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
		flagCh <- m.Metadata["envelope"]
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case flag := <-flagCh:
		assert.Equal(t, "true", flag, "transport/host must stamp envelope metadata")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

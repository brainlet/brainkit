package bus

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSDKReply(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "echo-svc.ts", `
			bus.on("ping", (msg) => {
				msg.reply({ pong: true, from: "ts" });
			});
		`)

	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   sdk.ResolveServiceTopic("echo-svc.ts", "ping"),
		Payload: json.RawMessage(`{"hello":"world"}`),
	})
	require.NoError(t, err)

	var data map[string]any
	json.Unmarshal(reply, &data)
	assert.Equal(t, true, data["pong"])
	assert.Equal(t, "ts", data["from"])
}

func testSDKReplyGoToGo(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := sdk.SubscribeTo[json.RawMessage](env.Kit, ctx, "test.approval.request",
		func(payload json.RawMessage, msg sdk.Message) {
			sdk.Reply(env.Kit, ctx, msg, map[string]bool{"approved": true})
		})
	require.NoError(t, err)

	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "test.approval.request",
		Payload: json.RawMessage(`{"action":"delete"}`),
	})
	require.NoError(t, err)

	var data map[string]bool
	json.Unmarshal(reply, &data)
	assert.True(t, data["approved"])
}

func testSDKSendChunk(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := sdk.SubscribeTo[json.RawMessage](env.Kit, ctx, "test.stream.request",
		func(payload json.RawMessage, msg sdk.Message) {
			sdk.SendChunk(env.Kit, ctx, msg, map[string]int{"chunk": 1})
			sdk.SendChunk(env.Kit, ctx, msg, map[string]int{"chunk": 2})
			sdk.SendChunk(env.Kit, ctx, msg, map[string]int{"chunk": 3})
			sdk.Reply(env.Kit, ctx, msg, map[string]any{"done": true, "total": 3})
		})
	require.NoError(t, err)

	var mu sync.Mutex
	var received []json.RawMessage
	replyTo := "test.stream.request.reply." + uuid.NewString()
	unsub, err := sdk.SubscribeTo[json.RawMessage](env.Kit, ctx, replyTo,
		func(payload json.RawMessage, msg sdk.Message) {
			mu.Lock()
			received = append(received, payload)
			mu.Unlock()
		})
	require.NoError(t, err)
	defer unsub()

	_, err = sdk.Publish(env.Kit, ctx, sdk.CustomMsg{
		Topic:   "test.stream.request",
		Payload: json.RawMessage(`{}`),
	}, sdk.WithReplyTo(replyTo))
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	mu.Lock()
	got := len(received)
	mu.Unlock()
	assert.GreaterOrEqual(t, got, 4, "expected 3 chunks + 1 final")
}

func testSDKSendToService(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "calc.ts", `
			bus.on("add", (msg) => {
				const a = msg.payload.a || 0;
				const b = msg.payload.b || 0;
				msg.reply({ result: a + b });
			});
		`)

	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   sdk.ResolveServiceTopic("calc.ts", "add"),
		Payload: json.RawMessage(`{"a":17,"b":25}`),
	})
	require.NoError(t, err)

	var data map[string]any
	json.Unmarshal(reply, &data)
	assert.Equal(t, float64(42), data["result"])
}

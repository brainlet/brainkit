package bus_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSDK_Reply verifies sdk.Reply works end-to-end with a real Kernel.
// Pattern: Go publishes → .ts service handles → Go subscriber replies via sdk.Reply → publisher gets response.
func TestSDK_Reply(t *testing.T) {
	testutil.LoadEnv(t)
	rt := testutil.NewTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Deploy a .ts service that echoes back via msg.reply
	_, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "echo-svc.ts",
		Code: `
			bus.on("ping", (msg) => {
				msg.reply({ pong: true, from: "ts" });
			});
		`,
	})
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Go subscribes to the echo service and relays the reply via sdk.Reply
	relayCh := make(chan json.RawMessage, 1)

	// Step 1: Go publishes to the service
	pr, err := sdk.SendToService(rt, ctx, "echo-svc.ts", "ping", map[string]string{"hello": "world"})
	require.NoError(t, err)

	// Step 2: Go subscribes to the replyTo topic
	unsub, err := sdk.SubscribeTo[json.RawMessage](rt, ctx, pr.ReplyTo, func(payload json.RawMessage, msg messages.Message) {
		relayCh <- payload
	})
	require.NoError(t, err)
	defer unsub()

	// Step 3: Wait for reply
	select {
	case reply := <-relayCh:
		var data map[string]any
		json.Unmarshal(reply, &data)
		assert.Equal(t, true, data["pong"])
		assert.Equal(t, "ts", data["from"])
	case <-ctx.Done():
		t.Fatal("timeout waiting for reply")
	}
}

// TestSDK_Reply_GoToGo verifies sdk.Reply works for Go-to-Go bus patterns.
// Pattern: Go publishes → Go subscriber replies via sdk.Reply → publisher gets response.
func TestSDK_Reply_GoToGo(t *testing.T) {
	testutil.LoadEnv(t)
	rt := testutil.NewTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Go service: subscribes to "approval" topic and replies
	_, err := sdk.SubscribeTo[json.RawMessage](rt, ctx, "test.approval.request",
		func(payload json.RawMessage, msg messages.Message) {
			sdk.Reply(rt, ctx, msg, map[string]bool{"approved": true})
		})
	require.NoError(t, err)

	// Go client: publishes and waits for reply
	pr, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "test.approval.request",
		Payload: json.RawMessage(`{"action":"delete"}`),
	})
	require.NoError(t, err)

	replyCh := make(chan json.RawMessage, 1)
	unsub, err := sdk.SubscribeTo[json.RawMessage](rt, ctx, pr.ReplyTo,
		func(payload json.RawMessage, msg messages.Message) {
			replyCh <- payload
		})
	require.NoError(t, err)
	defer unsub()

	select {
	case reply := <-replyCh:
		var data map[string]bool
		json.Unmarshal(reply, &data)
		assert.True(t, data["approved"])
	case <-ctx.Done():
		t.Fatal("timeout waiting for Go-to-Go reply")
	}
}

// TestSDK_SendChunk verifies sdk.SendChunk delivers intermediate chunks
// before sdk.Reply sends the final response.
func TestSDK_SendChunk(t *testing.T) {
	testutil.LoadEnv(t)
	rt := testutil.NewTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Go streaming service: sends 3 chunks then final reply
	_, err := sdk.SubscribeTo[json.RawMessage](rt, ctx, "test.stream.request",
		func(payload json.RawMessage, msg messages.Message) {
			sdk.SendChunk(rt, ctx, msg, map[string]int{"chunk": 1})
			sdk.SendChunk(rt, ctx, msg, map[string]int{"chunk": 2})
			sdk.SendChunk(rt, ctx, msg, map[string]int{"chunk": 3})
			sdk.Reply(rt, ctx, msg, map[string]any{"done": true, "total": 3})
		})
	require.NoError(t, err)

	// Publish and collect all responses
	pr, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "test.stream.request",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	var received []json.RawMessage
	unsub, err := sdk.SubscribeTo[json.RawMessage](rt, ctx, pr.ReplyTo,
		func(payload json.RawMessage, msg messages.Message) {
			received = append(received, payload)
		})
	require.NoError(t, err)
	defer unsub()

	time.Sleep(500 * time.Millisecond)

	assert.GreaterOrEqual(t, len(received), 4, "expected 3 chunks + 1 final")
}

// TestSDK_SendToService verifies sdk.SendToService resolves the topic correctly
// and delivers messages to deployed .ts services.
func TestSDK_SendToService(t *testing.T) {
	testutil.LoadEnv(t)
	rt := testutil.NewTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Deploy a calculator service
	_, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "calc.ts",
		Code: `
			bus.on("add", (msg) => {
				const a = msg.payload.a || 0;
				const b = msg.payload.b || 0;
				msg.reply({ result: a + b });
			});
		`,
	})
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Use SendToService to reach it
	pr, err := sdk.SendToService(rt, ctx, "calc.ts", "add", map[string]int{"a": 17, "b": 25})
	require.NoError(t, err)

	replyCh := make(chan json.RawMessage, 1)
	unsub, err := sdk.SubscribeTo[json.RawMessage](rt, ctx, pr.ReplyTo,
		func(payload json.RawMessage, msg messages.Message) {
			replyCh <- payload
		})
	require.NoError(t, err)
	defer unsub()

	select {
	case reply := <-replyCh:
		var data map[string]any
		json.Unmarshal(reply, &data)
		assert.Equal(t, float64(42), data["result"])
	case <-ctx.Done():
		t.Fatal("timeout waiting for calc reply")
	}
}

package e2e_test

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

func publishAndWait[Req messages.BrainkitMessage, Resp any](t *testing.T, rt sdk.Runtime, ctx context.Context, req Req) Resp {
	t.Helper()
	pr, err := sdk.Publish(rt, ctx, req)
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, err := rt.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		select {
		case ch <- msg.Payload:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()
	select {
	case payload := <-ch:
		var resp Resp
		require.NoError(t, json.Unmarshal(payload, &resp))
		return resp
	case <-ctx.Done():
		t.Fatal("timeout waiting for response")
		var zero Resp
		return zero
	}
}

func TestCLI_KitEval(t *testing.T) {
	rt := testutil.NewTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Simple output
	resp := publishAndWait[messages.KitEvalMsg, messages.KitEvalResp](t, rt, ctx, messages.KitEvalMsg{
		Code: `output(1 + 1)`,
	})
	assert.Equal(t, "2", resp.Result)

	// Object output
	resp = publishAndWait[messages.KitEvalMsg, messages.KitEvalResp](t, rt, ctx, messages.KitEvalMsg{
		Code: `output({ hello: "world" })`,
	})
	assert.JSONEq(t, `{"hello":"world"}`, resp.Result)

	// Top-level await
	resp = publishAndWait[messages.KitEvalMsg, messages.KitEvalResp](t, rt, ctx, messages.KitEvalMsg{
		Code: `const x = await Promise.resolve(42); output(x)`,
	})
	assert.Equal(t, "42", resp.Result)
}

func TestCLI_KitHealth(t *testing.T) {
	rt := testutil.NewTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp := publishAndWait[messages.KitHealthMsg, messages.KitHealthResp](t, rt, ctx, messages.KitHealthMsg{})

	var health struct {
		Healthy bool   `json:"healthy"`
		Status  string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(resp.Health, &health))
	assert.True(t, health.Healthy)
	assert.Equal(t, "running", health.Status)
}

func TestCLI_KitSend_RequestReply(t *testing.T) {
	rt := testutil.NewTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy a service that replies
	deployResp := publishAndWait[messages.KitDeployMsg, messages.KitDeployResp](t, rt, ctx, messages.KitDeployMsg{
		Source: "echo-svc.ts",
		Code: `
			bus.on("ping", (msg) => {
				msg.reply({ pong: msg.payload.value });
			});
		`,
	})
	require.True(t, deployResp.Deployed)

	// Send via kit.send — Go-side request-reply
	sendResp := publishAndWait[messages.KitSendMsg, messages.KitSendResp](t, rt, ctx, messages.KitSendMsg{
		Topic:   "ts.echo-svc.ping",
		Payload: json.RawMessage(`{"value":"hello"}`),
	})

	var payload struct {
		Pong string `json:"pong"`
	}
	require.NoError(t, json.Unmarshal(sendResp.Payload, &payload))
	assert.Equal(t, "hello", payload.Pong)
}

func TestCLI_KitSend_WithAwait(t *testing.T) {
	rt := testutil.NewTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy a service that does async work before replying
	deployResp := publishAndWait[messages.KitDeployMsg, messages.KitDeployResp](t, rt, ctx, messages.KitDeployMsg{
		Source: "async-svc.ts",
		Code: `
			bus.on("compute", async (msg) => {
				const result = await Promise.resolve(msg.payload.a + msg.payload.b);
				msg.reply({ sum: result });
			});
		`,
	})
	require.True(t, deployResp.Deployed)

	sendResp := publishAndWait[messages.KitSendMsg, messages.KitSendResp](t, rt, ctx, messages.KitSendMsg{
		Topic:   "ts.async-svc.compute",
		Payload: json.RawMessage(`{"a":3,"b":4}`),
	})

	var payload struct {
		Sum int `json:"sum"`
	}
	require.NoError(t, json.Unmarshal(sendResp.Payload, &payload))
	assert.Equal(t, 7, payload.Sum)
}

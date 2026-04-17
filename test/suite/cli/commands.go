package cli

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

func publishAndWait[Req sdk.BrainkitMessage, Resp any](t *testing.T, rt sdk.Runtime, ctx context.Context, req Req) Resp {
	t.Helper()
	pr, err := sdk.Publish(rt, ctx, req)
	require.NoError(t, err)
	ch := make(chan json.RawMessage, 1)
	unsub, err := rt.SubscribeRaw(ctx, pr.ReplyTo, func(msg sdk.Message) {
		select {
		case ch <- suite.ResponseDataFromMsg(msg):
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

func testKitEval(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp := publishAndWait[sdk.KitEvalMsg, sdk.KitEvalResp](t, env.Kit, ctx, sdk.KitEvalMsg{
		Code: `output(1 + 1)`,
	})
	assert.Equal(t, "2", resp.Result)

	resp = publishAndWait[sdk.KitEvalMsg, sdk.KitEvalResp](t, env.Kit, ctx, sdk.KitEvalMsg{
		Code: `output({ hello: "world" })`,
	})
	assert.JSONEq(t, `{"hello":"world"}`, resp.Result)

	resp = publishAndWait[sdk.KitEvalMsg, sdk.KitEvalResp](t, env.Kit, ctx, sdk.KitEvalMsg{
		Code: `const x = await Promise.resolve(42); output(x)`,
	})
	assert.Equal(t, "42", resp.Result)
}

func testKitHealth(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp := publishAndWait[sdk.KitHealthMsg, sdk.KitHealthResp](t, env.Kit, ctx, sdk.KitHealthMsg{})

	var health struct {
		Healthy bool   `json:"healthy"`
		Status  string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(resp.Health, &health))
	assert.True(t, health.Healthy)
	assert.Equal(t, "running", health.Status)
}

func testKitSendRequestReply(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	manifest1, _ := json.Marshal(map[string]string{"name": "echo-svc-cmd", "entry": "echo-svc-cmd.ts"})
	deployResp := publishAndWait[sdk.PackageDeployMsg, sdk.PackageDeployResp](t, env.Kit, ctx, sdk.PackageDeployMsg{
		Manifest: manifest1,
		Files: map[string]string{"echo-svc-cmd.ts": `
			bus.on("ping", (msg) => {
				msg.reply({ pong: msg.payload.value });
			});
		`},
	})
	require.True(t, deployResp.Deployed)

	sendResp := publishAndWait[sdk.KitSendMsg, sdk.KitSendResp](t, env.Kit, ctx, sdk.KitSendMsg{
		Topic:   "ts.echo-svc-cmd.ping",
		Payload: json.RawMessage(`{"value":"hello"}`),
	})

	var payload struct{ Pong string `json:"pong"` }
	require.NoError(t, json.Unmarshal(suite.ResponseData(sendResp.Payload), &payload))
	assert.Equal(t, "hello", payload.Pong)
}

func testKitSendWithAwait(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	manifest2, _ := json.Marshal(map[string]string{"name": "async-svc-cmd", "entry": "async-svc-cmd.ts"})
	deployResp := publishAndWait[sdk.PackageDeployMsg, sdk.PackageDeployResp](t, env.Kit, ctx, sdk.PackageDeployMsg{
		Manifest: manifest2,
		Files: map[string]string{"async-svc-cmd.ts": `
			bus.on("compute", async (msg) => {
				const result = await Promise.resolve(msg.payload.a + msg.payload.b);
				msg.reply({ sum: result });
			});
		`},
	})
	require.True(t, deployResp.Deployed)

	sendResp := publishAndWait[sdk.KitSendMsg, sdk.KitSendResp](t, env.Kit, ctx, sdk.KitSendMsg{
		Topic:   "ts.async-svc-cmd.compute",
		Payload: json.RawMessage(`{"a":3,"b":4}`),
	})

	var payload struct{ Sum int `json:"sum"` }
	require.NoError(t, json.Unmarshal(suite.ResponseData(sendResp.Payload), &payload))
	assert.Equal(t, 7, payload.Sum)
}

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	healthmod "github.com/brainlet/brainkit/modules/health"
	messagingmod "github.com/brainlet/brainkit/modules/messaging"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func publishAndWait[Req sdk.BrainkitMessage, Resp any](t *testing.T, rt sdk.Runtime, ctx context.Context, req Req) Resp {
	t.Helper()
	callerRT, ok := rt.(sdk.CallerRuntime)
	require.True(t, ok, "runtime must expose sdk.Caller for %s", req.BusTopic())
	resp, err := sdk.Call[Req, Resp](callerRT, ctx, req)
	require.NoError(t, err)
	return resp
}

func serviceTopic(service, topic string) string {
	name := strings.TrimSuffix(service, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}

func callService[Resp any](t *testing.T, rt sdk.Runtime, ctx context.Context, service, topic string, payload any) Resp {
	t.Helper()
	callerRT, ok := rt.(sdk.CallerRuntime)
	require.True(t, ok, "runtime must expose sdk.Caller for %s/%s", service, topic)
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	resp, err := sdk.Call[sdk.CustomMsg, Resp](callerRT, ctx, sdk.CustomMsg{
		Topic:   serviceTopic(service, topic),
		Payload: data,
	})
	require.NoError(t, err, fmt.Sprintf("call service %s/%s", service, topic))
	return resp
}

func testKitEval(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp := publishAndWait[evalmsg.KitEvalMsg, evalmsg.KitEvalResp](t, env.Kit, ctx, evalmsg.KitEvalMsg{
		Code: `output(1 + 1)`,
	})
	assert.Equal(t, "2", resp.Result)

	resp = publishAndWait[evalmsg.KitEvalMsg, evalmsg.KitEvalResp](t, env.Kit, ctx, evalmsg.KitEvalMsg{
		Code: `output({ hello: "world" })`,
	})
	assert.JSONEq(t, `{"hello":"world"}`, resp.Result)

	resp = publishAndWait[evalmsg.KitEvalMsg, evalmsg.KitEvalResp](t, env.Kit, ctx, evalmsg.KitEvalMsg{
		Code: `const x = await Promise.resolve(42); output(x)`,
	})
	assert.Equal(t, "42", resp.Result)
}

func testKitHealth(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp := publishAndWait[healthmod.KitHealthMsg, healthmod.KitHealthResp](t, env.Kit, ctx, healthmod.KitHealthMsg{})

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
	deployResp := publishAndWait[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](t, env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: manifest1,
		Files: map[string]string{"echo-svc-cmd.ts": `
			bus.on("ping", (msg) => {
				msg.reply({ pong: msg.payload.value });
			});
		`},
	})
	require.True(t, deployResp.Deployed)

	sendResp := publishAndWait[messagingmod.KitSendMsg, messagingmod.KitSendResp](t, env.Kit, ctx, messagingmod.KitSendMsg{
		Topic:   "ts.echo-svc-cmd.ping",
		Payload: json.RawMessage(`{"value":"hello"}`),
	})

	var payload struct {
		Pong string `json:"pong"`
	}
	require.NoError(t, json.Unmarshal(suite.ResponseData(sendResp.Payload), &payload))
	assert.Equal(t, "hello", payload.Pong)
}

func testKitSendWithAwait(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	manifest2, _ := json.Marshal(map[string]string{"name": "async-svc-cmd", "entry": "async-svc-cmd.ts"})
	deployResp := publishAndWait[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](t, env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: manifest2,
		Files: map[string]string{"async-svc-cmd.ts": `
			bus.on("compute", async (msg) => {
				const result = await Promise.resolve(msg.payload.a + msg.payload.b);
				msg.reply({ sum: result });
			});
		`},
	})
	require.True(t, deployResp.Deployed)

	sendResp := publishAndWait[messagingmod.KitSendMsg, messagingmod.KitSendResp](t, env.Kit, ctx, messagingmod.KitSendMsg{
		Topic:   "ts.async-svc-cmd.compute",
		Payload: json.RawMessage(`{"a":3,"b":4}`),
	})

	var payload struct {
		Sum int `json:"sum"`
	}
	require.NoError(t, json.Unmarshal(suite.ResponseData(sendResp.Payload), &payload))
	assert.Equal(t, 7, payload.Sum)
}

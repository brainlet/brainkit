package bus

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

// deployAndSendDiag deploys a .ts service with bus.on("test", handler), sends a
// message, and returns the reply payload. Shared helper for the async diag tests.
func deployAndSendDiag(t *testing.T, env *suite.TestEnv, source, code string, timeout time.Duration) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.KitDeployMsg{Source: source, Code: code})
	require.NoError(t, err)
	ch := make(chan sdk.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[sdk.KitDeployResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.KitDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("deploy timeout")
	}
	time.Sleep(100 * time.Millisecond)

	pr2, err := sdk.SendToService(env.Kit, ctx, source, "test", json.RawMessage(`{}`))
	require.NoError(t, err)
	replyCh := make(chan sdk.Message, 1)
	unsub2, _ := env.Kit.SubscribeRaw(ctx, pr2.ReplyTo, func(msg sdk.Message) {
		if msg.Metadata["done"] == "true" {
			replyCh <- msg
		}
	})
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]any
		json.Unmarshal(suite.ResponseData(msg.Payload), &result)
		return result
	case <-ctx.Done():
		t.Fatal("timeout waiting for reply")
		return nil
	}
}

func testDiagBusOnAwaitPromiseResolve(t *testing.T, env *suite.TestEnv) {
	result := deployAndSendDiag(t, env, "diag-promise-resolve.ts", `
		bus.on("test", async (msg) => {
			const val = await Promise.resolve("micro");
			msg.reply({ result: val });
		});
	`, 15*time.Second)
	assert.Equal(t, "micro", result["result"])
}

func testDiagBusOnAwaitSetTimeout(t *testing.T, env *suite.TestEnv) {
	result := deployAndSendDiag(t, env, "diag-settimeout.ts", `
		bus.on("test", async (msg) => {
			await new Promise(resolve => setTimeout(resolve, 50));
			msg.reply({ result: "delayed" });
		});
	`, 15*time.Second)
	assert.Equal(t, "delayed", result["result"])
}

func testDiagBusOnAwaitToolsCall(t *testing.T, env *suite.TestEnv) {
	result := deployAndSendDiag(t, env, "diag-tools-call.ts", `
		bus.on("test", async (msg) => {
			const result = await tools.call("echo", { message: "from-bus-handler" });
			msg.reply({ result: result });
		});
	`, 15*time.Second)
	inner, _ := result["result"].(map[string]any)
	assert.Equal(t, "from-bus-handler", inner["echoed"])
}

func testDiagBusOnAwaitFetch(t *testing.T, env *suite.TestEnv) {
	result := deployAndSendDiag(t, env, "diag-fetch.ts", `
		bus.on("test", async (msg) => {
			try {
				const resp = await fetch("https://httpbin.org/get");
				const text = await resp.text();
				msg.reply({ status: resp.status, bodyLen: text.length });
			} catch (e) {
				msg.reply({ error: e.message || String(e) });
			}
		});
	`, 30*time.Second)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("fetch inside bus.on returned error: %v", errMsg)
	}
	assert.Equal(t, float64(200), result["status"])
}

func testDiagBusOnAwaitGenerateText(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)
	result := deployAndSendDiag(t, env, "diag-generatetext.ts", `
		bus.on("test", async (msg) => {
			try {
				const result = await generateText({
					model: model("openai", "gpt-4o-mini"),
					prompt: "Say hi",
					maxTokens: 5,
				});
				msg.reply({ text: result.text });
			} catch (e) {
				msg.reply({ error: e.message || String(e) });
			}
		});
	`, 60*time.Second)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("generateText inside bus.on returned error: %v", errMsg)
	}
	assert.NotEmpty(t, result["text"])
}

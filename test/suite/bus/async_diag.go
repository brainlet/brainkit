package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
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

	testutil.Deploy(t, env.Kit, source, code)
	data, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   protocol.ResolveServiceTopic(source, "test"),
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	var result map[string]any
	json.Unmarshal(data, &result)
	return result
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

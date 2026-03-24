package surface_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Diagnostic tests to isolate why async operations inside bus.on handlers fail.
// Each test increases async complexity to find the exact failure point.

// Level 1: bus.on handler with await Promise.resolve (microtask only, no Schedule)
func TestDiag_BusOn_AwaitPromiseResolve(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "diag-promise-resolve.ts",
		Code: `
			bus.on("test", async (msg) => {
				const val = await Promise.resolve("micro");
				msg.reply({ result: val });
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("deploy timeout")
	}
	time.Sleep(100 * time.Millisecond)

	pr2, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.diag-promise-resolve.test",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	replyCh := make(chan messages.Message, 1)
	unsub2, _ := rt.SubscribeRaw(ctx, pr2.ReplyTo, func(msg messages.Message) {
		if msg.Metadata["done"] == "true" {
			replyCh <- msg
		}
	})
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]string
		json.Unmarshal(msg.Payload, &result)
		assert.Equal(t, "micro", result["result"])
		t.Log("PASS: await Promise.resolve works inside bus.on")
	case <-ctx.Done():
		t.Fatal("FAIL: await Promise.resolve inside bus.on timed out")
	}
}

// Level 2: bus.on handler with await setTimeout (uses Go Schedule via bridge.Go)
func TestDiag_BusOn_AwaitSetTimeout(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "diag-settimeout.ts",
		Code: `
			bus.on("test", async (msg) => {
				await new Promise(resolve => setTimeout(resolve, 50));
				msg.reply({ result: "delayed" });
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("deploy timeout")
	}
	time.Sleep(100 * time.Millisecond)

	pr2, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.diag-settimeout.test",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	replyCh := make(chan messages.Message, 1)
	unsub2, _ := rt.SubscribeRaw(ctx, pr2.ReplyTo, func(msg messages.Message) {
		if msg.Metadata["done"] == "true" {
			replyCh <- msg
		}
	})
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]string
		json.Unmarshal(msg.Payload, &result)
		assert.Equal(t, "delayed", result["result"])
		t.Log("PASS: await setTimeout works inside bus.on")
	case <-ctx.Done():
		t.Fatal("FAIL: await setTimeout inside bus.on timed out — Schedule delivery broken in Await-inside-ProcessJobs")
	}
}

// Level 3: bus.on handler with await tools.call (uses bridgeRequestAsync → Go Schedule)
func TestDiag_BusOn_AwaitToolsCall(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "diag-tools-call.ts",
		Code: `
			bus.on("test", async (msg) => {
				const result = await tools.call("echo", { message: "from-bus-handler" });
				msg.reply({ result: result });
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("deploy timeout")
	}
	time.Sleep(100 * time.Millisecond)

	pr2, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.diag-tools-call.test",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	replyCh := make(chan messages.Message, 1)
	unsub2, _ := rt.SubscribeRaw(ctx, pr2.ReplyTo, func(msg messages.Message) {
		if msg.Metadata["done"] == "true" {
			replyCh <- msg
		}
	})
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]any
		json.Unmarshal(msg.Payload, &result)
		inner, _ := result["result"].(map[string]any)
		assert.Equal(t, "from-bus-handler", inner["echoed"])
		t.Log("PASS: await tools.call works inside bus.on")
	case <-ctx.Done():
		t.Fatal("FAIL: await tools.call inside bus.on timed out")
	}
}

// Level 4: bus.on handler with await fetch (real HTTP call via Go goroutine)
func TestDiag_BusOn_AwaitFetch(t *testing.T) {
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a simple HTTP endpoint that responds quickly
	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "diag-fetch.ts",
		Code: `
			bus.on("test", async (msg) => {
				console.log("[diag-fetch] handler called");
				try {
					console.log("[diag-fetch] calling fetch...");
					const resp = await fetch("https://httpbin.org/get");
					console.log("[diag-fetch] fetch returned, status: " + resp.status);
					const text = await resp.text();
					console.log("[diag-fetch] body length: " + text.length);
					msg.reply({ status: resp.status, bodyLen: text.length });
				} catch (e) {
					console.error("[diag-fetch] error: " + (e.message || e));
					msg.reply({ error: e.message || String(e) });
				}
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("deploy timeout")
	}
	time.Sleep(100 * time.Millisecond)

	pr2, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.diag-fetch.test",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	replyCh := make(chan messages.Message, 1)
	unsub2, _ := rt.SubscribeRaw(ctx, pr2.ReplyTo, func(msg messages.Message) {
		if msg.Metadata["done"] == "true" {
			replyCh <- msg
		}
	})
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]any
		json.Unmarshal(msg.Payload, &result)
		if errMsg, ok := result["error"]; ok {
			t.Fatalf("FAIL: fetch inside bus.on returned error: %v", errMsg)
		}
		assert.Equal(t, float64(200), result["status"])
		t.Logf("PASS: await fetch works inside bus.on (body: %v bytes)", result["bodyLen"])
	case <-ctx.Done():
		t.Fatal("FAIL: await fetch inside bus.on timed out — fetch Promise never resolves")
	}
}

// Level 5: bus.on handler with await generateText (real AI SDK + HTTP)
func TestDiag_BusOn_AwaitGenerateText(t *testing.T) {
	testutil.LoadEnv(t)
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	rt := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "diag-generatetext.ts",
		Code: `
			bus.on("test", async (msg) => {
				console.log("[diag-gen] handler called");
				try {
					console.log("[diag-gen] calling generateText...");
					const result = await generateText({
						model: model("openai", "gpt-4o-mini"),
						prompt: "Say hi",
						maxTokens: 5,
					});
					console.log("[diag-gen] got result: " + result.text);
					msg.reply({ text: result.text });
				} catch (e) {
					console.error("[diag-gen] error: " + (e.message || e));
					msg.reply({ error: e.message || String(e) });
				}
			});
		`,
	})
	require.NoError(t, err)
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("deploy timeout")
	}
	time.Sleep(100 * time.Millisecond)

	pr2, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.diag-generatetext.test",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	replyCh := make(chan messages.Message, 1)
	unsub2, _ := rt.SubscribeRaw(ctx, pr2.ReplyTo, func(msg messages.Message) {
		if msg.Metadata["done"] == "true" {
			replyCh <- msg
		}
	})
	defer unsub2()

	select {
	case msg := <-replyCh:
		var result map[string]any
		json.Unmarshal(msg.Payload, &result)
		if errMsg, ok := result["error"]; ok {
			t.Fatalf("FAIL: generateText inside bus.on returned error: %v", errMsg)
		}
		assert.NotEmpty(t, result["text"])
		t.Logf("PASS: await generateText works inside bus.on: %v", result["text"])
	case <-ctx.Done():
		t.Fatal("FAIL: await generateText inside bus.on timed out")
	}
}

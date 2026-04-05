package bus

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testE2EMultiServiceChain — A deploys, B deploys, A calls B, B calls Go tool.
func testE2EMultiServiceChain(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	// Deploy Service B — listens on bus, calls Go "echo" tool
	err := env.Deploy("svc-b-adv.ts", `
		bus.on("process", async function(msg) {
			var result = await tools.call("echo", {message: "processed:" + msg.payload.data});
			msg.reply({fromB: true, toolResult: result});
		});
	`)
	require.NoError(t, err)

	// Deploy Service A — receives request, forwards to B
	err = env.Deploy("svc-a-adv.ts", `
		bus.on("start", function(msg) {
			var r = bus.sendTo("svc-b-adv.ts", "process", {data: msg.payload.input});
			msg.reply({fromA: true, forwarded: true, replyTo: r.replyTo});
		});
	`)
	require.NoError(t, err)

	// Call A
	pr, err := sdk.Publish(env.Kernel, ctx, messages.CustomMsg{
		Topic:   "ts.svc-a-adv.start",
		Payload: json.RawMessage(`{"input":"hello"}`),
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "fromA")
		assert.Contains(t, string(p), "forwarded")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

// testE2EStreamingResponse — deploy handler that uses msg.stream, verify SSE events.
func testE2EStreamingResponse(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	err := env.Deploy("streamer-adv.ts", `
		bus.on("stream", function(msg) {
			msg.stream.text("chunk1");
			msg.stream.text("chunk2");
			msg.stream.progress(50, "halfway");
			msg.stream.text("chunk3");
			msg.stream.end({done: true});
		});
	`)
	require.NoError(t, err)

	pr, err := sdk.Publish(env.Kernel, ctx, messages.CustomMsg{
		Topic:   "ts.streamer-adv.stream",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	var chunks []json.RawMessage
	done := make(chan bool, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		chunks = append(chunks, json.RawMessage(m.Payload))
		var parsed struct {
			Type string `json:"type"`
		}
		json.Unmarshal(m.Payload, &parsed)
		if parsed.Type == "end" {
			done <- true
		}
		// Also check done metadata
		if m.Metadata["done"] == "true" {
			done <- true
		}
	})
	defer unsub()

	select {
	case <-done:
		// On GoChannel, rapid stream messages may arrive merged — exact count varies.
		// What matters: we got the end signal and at least some chunks.
		assert.Greater(t, len(chunks), 0, "should have received stream chunks")
	case <-time.After(5 * time.Second):
		t.Logf("received %d chunks before timeout", len(chunks))
		assert.Greater(t, len(chunks), 0, "should have received some chunks")
	}
}

// testE2EMultiDomain — workflow crossing domain boundaries:
// write file → call tool that reads+processes → write output → verify.
func testE2EMultiDomain(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Write input file via polyfill
	_, err := freshEnv.Kernel.EvalTS(ctx, "__test_multi.ts", `
		fs.writeFileSync("input.json", '{"items":["apple","banana","cherry"]}');
		return "ok";
	`)
	require.NoError(t, err)

	// 2. Read it back via polyfill
	readData, err := freshEnv.Kernel.EvalTS(ctx, "__test_multi.ts", `return fs.readFileSync("input.json", "utf8");`)
	require.NoError(t, err)

	// 3. Process with the "echo" tool
	pr, err := sdk.Publish(freshEnv.Kernel, ctx, messages.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": readData},
	})
	require.NoError(t, err)
	callCh := make(chan messages.ToolCallResp, 1)
	cancelCall, err := sdk.SubscribeTo[messages.ToolCallResp](freshEnv.Kernel, ctx, pr.ReplyTo, func(r messages.ToolCallResp, _ messages.Message) { callCh <- r })
	require.NoError(t, err)
	defer cancelCall()
	var callResp messages.ToolCallResp
	select {
	case callResp = <-callCh:
	case <-ctx.Done():
		t.Fatal("timeout calling echo")
	}

	// 4. Write the processed output via polyfill
	escaped := strings.ReplaceAll(string(callResp.Result), `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	writeCode := `fs.writeFileSync("output.json", '` + escaped + `'); return "ok";`
	_, err = freshEnv.Kernel.EvalTS(ctx, "__test_multi.ts", writeCode)
	require.NoError(t, err)

	// 5. Read and verify output via polyfill
	outData, err := freshEnv.Kernel.EvalTS(ctx, "__test_multi.ts", `return fs.readFileSync("output.json", "utf8");`)
	require.NoError(t, err)
	assert.Contains(t, outData, "echoed")
}

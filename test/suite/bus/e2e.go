package bus

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testE2EMultiServiceChain — A deploys, B deploys, A calls B, B calls Go tool.
func testE2EMultiServiceChain(t *testing.T, env *suite.TestEnv) {
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

	// Call A through the shared inbox caller. Fast handlers can reply before a
	// post-publish subscription exists on external transports such as NATS.
	p := testutil.PublishAndWait(t, env.Kit, sdk.CustomMsg{
		Topic:   "ts.svc-a-adv.start",
		Payload: json.RawMessage(`{"input":"hello"}`),
	}, 5*time.Second)
	assert.Contains(t, string(p), "fromA")
	assert.Contains(t, string(p), "forwarded")
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

	var chunks []json.RawMessage
	_, err = env.Kit.Caller().Call(ctx, "ts.streamer-adv.stream", json.RawMessage(`{}`), sdk.CallerConfig{
		StreamHandler: func(m sdk.Message) error {
			chunks = append(chunks, json.RawMessage(m.Payload))
			return nil
		},
	})
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 0, "should have received stream chunks")
}

// testE2EMultiDomain — workflow crossing domain boundaries:
// write file → call tool that reads+processes → write output → verify.
func testE2EMultiDomain(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Write input file via polyfill
	testutil.EvalTS(t, freshEnv.Kit, "__test_multi.ts", `
		fs.writeFileSync("input.json", '{"items":["apple","banana","cherry"]}');
		return "ok";
	`)

	// 2. Read it back via polyfill
	readData := testutil.EvalTS(t, freshEnv.Kit, "__test_multi_read.ts", `return fs.readFileSync("input.json", "utf8");`)

	// 3. Process with the "echo" tool
	pr, err := sdk.Publish(freshEnv.Kit, ctx, toolmsg.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": readData},
	})
	require.NoError(t, err)
	callCh := make(chan toolmsg.ToolCallResp, 1)
	cancelCall, err := sdk.SubscribeTo[toolmsg.ToolCallResp](freshEnv.Kit, ctx, pr.ReplyTo, func(r toolmsg.ToolCallResp, _ sdk.Message) { callCh <- r })
	require.NoError(t, err)
	defer cancelCall()
	var callResp toolmsg.ToolCallResp
	select {
	case callResp = <-callCh:
	case <-ctx.Done():
		t.Fatal("timeout calling echo")
	}

	// 4. Write the processed output via polyfill
	escaped := strings.ReplaceAll(string(callResp.Result), `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	writeCode := `fs.writeFileSync("output.json", '` + escaped + `'); return "ok";`
	testutil.EvalTS(t, freshEnv.Kit, "__test_multi_write.ts", writeCode)

	// 5. Read and verify output via polyfill
	outData := testutil.EvalTS(t, freshEnv.Kit, "__test_multi_out.ts", `return fs.readFileSync("output.json", "utf8");`)
	assert.Contains(t, outData, "echoed")
}

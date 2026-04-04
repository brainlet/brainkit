package bus

import (
	"context"
	"encoding/json"
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

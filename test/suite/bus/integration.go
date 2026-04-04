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

func testTwoServiceInteraction(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	err := env.Deploy("service-b-int.ts", `
		bus.on("process", (msg) => {
			msg.reply({ processed: msg.payload.data + "-done" });
		});
	`)
	require.NoError(t, err)

	err = env.Deploy("service-a-int.ts", `
		bus.on("ask", async (msg) => {
			var resp = await bus.sendTo("service-b-int.ts", "process", { data: msg.payload.data });
			msg.reply({ forwarded: resp.processed || resp });
		});
	`)
	require.NoError(t, err)
	time.Sleep(300 * time.Millisecond)

	sendPR, err := sdk.SendToService(env.Kernel, ctx, "service-a-int.ts", "ask", map[string]string{"data": "hello"})
	require.NoError(t, err)

	replyCh := make(chan json.RawMessage, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		replyCh <- json.RawMessage(msg.Payload)
	})
	defer unsub()

	select {
	case raw := <-replyCh:
		assert.NotEmpty(t, raw, "should receive response from two-service chain")
		t.Logf("two-service response: %s", string(raw))
	case <-time.After(10 * time.Second):
		t.Fatal("timeout — two-service interaction failed")
	}
}

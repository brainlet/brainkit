package bus

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tsCallDeployAndTrigger deploys a .ts that has a bus.on("trigger", ...)
// handler using bus.call internally, then sends a trigger and returns the
// reply payload map.
func tsCallDeployAndTrigger(t *testing.T, env *suite.TestEnv, source, handlerCode string) map[string]any {
	t.Helper()
	testutil.Deploy(t, env.Kit, source, handlerCode)
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pr, err := sdk.SendToService(env.Kit, ctx, source, "trigger", map[string]any{})
	require.NoError(t, err)

	replyCh := make(chan sdk.Message, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(msg sdk.Message) {
		if msg.Metadata["done"] == "true" {
			select {
			case replyCh <- msg:
			default:
			}
		}
	})
	defer unsub()

	select {
	case msg := <-replyCh:
		data := suite.ResponseData(msg.Payload)
		var m map[string]any
		if len(data) > 0 {
			_ = json.Unmarshal(data, &m)
		}
		return m
	case <-ctx.Done():
		t.Fatal("timeout waiting for trigger reply")
		return nil
	}
}

// testTSBusCallHappyPath — .ts handler B's bus.on uses bus.call to reach
// another .ts handler A; the reply payload bubbles back to the test.
func testTSBusCallHappyPath(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-call-server.ts", `
		bus.on("echo", (msg) => {
			msg.reply({ echoed: msg.payload.text + "!" });
		});
	`)
	time.Sleep(100 * time.Millisecond)

	reply := tsCallDeployAndTrigger(t, env, "ts-call-client.ts", `
		bus.on("trigger", async (msg) => {
			const r = await bus.call("ts.ts-call-server.echo", { text: "hi" }, { timeoutMs: 5000 });
			msg.reply({ echoed: r.echoed });
		});
	`)
	assert.Equal(t, "hi!", reply["echoed"])
}

// testTSBusCallRequiresTimeout — bus.call without timeoutMs rejects.
func testTSBusCallRequiresTimeout(t *testing.T, env *suite.TestEnv) {
	reply := tsCallDeployAndTrigger(t, env, "ts-call-notimeout.ts", `
		bus.on("trigger", async (msg) => {
			try {
				await bus.call("ts.nobody.nothing", { x: 1 });
				msg.reply({ code: "NO_THROW" });
			} catch (e) {
				msg.reply({ code: e.code || "NO_CODE", message: e.message || "" });
			}
		});
	`)
	assert.Equal(t, "VALIDATION_ERROR", reply["code"])
	assert.Contains(t, reply["message"], "timeoutMs")
}

// testTSBusCallPropagatesBrainkitError — remote handler throws BrainkitError;
// caller sees the typed code in the JS BrainkitError.
func testTSBusCallPropagatesBrainkitError(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-call-thrower.ts", `
		bus.on("boom", (msg) => {
			throw new BrainkitError("nope", "NOT_FOUND", { resource: "thing", name: "gone" });
		});
	`)
	time.Sleep(100 * time.Millisecond)

	reply := tsCallDeployAndTrigger(t, env, "ts-call-catcher.ts", `
		bus.on("trigger", async (msg) => {
			try {
				await bus.call("ts.ts-call-thrower.boom", {}, { timeoutMs: 5000 });
				msg.reply({ code: "NO_THROW" });
			} catch (e) {
				msg.reply({ code: e.code || "NO_CODE", message: e.message || "" });
			}
		});
	`)
	assert.Equal(t, "NOT_FOUND", reply["code"])
}

// testTSBusCallTimesOut — bus.call with a short deadline + silent handler
// surfaces as CALL_TIMEOUT.
func testTSBusCallTimesOut(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-call-silent.ts", `
		bus.on("slow", (msg) => { /* never reply */ });
	`)
	time.Sleep(100 * time.Millisecond)

	reply := tsCallDeployAndTrigger(t, env, "ts-call-impatient.ts", `
		bus.on("trigger", async (msg) => {
			try {
				await bus.call("ts.ts-call-silent.slow", {}, { timeoutMs: 150 });
				msg.reply({ code: "NO_THROW" });
			} catch (e) {
				msg.reply({ code: e.code || "NO_CODE" });
			}
		});
	`)
	assert.Equal(t, "CALL_TIMEOUT", reply["code"])
}

// testGoBusCallToTS — Go brainkit.Call → .ts handler that replies. Verifies
// envelope round-trip end-to-end.
func testGoBusCallToTS(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-goclient-server.ts", `
		bus.on("add", (msg) => {
			msg.reply({ sum: msg.payload.a + msg.payload.b });
		});
	`)
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := brainkit.Call[sdk.CustomMsg, map[string]any](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.ts-goclient-server.add",
		Payload: []byte(`{"a":3,"b":4}`),
	})
	require.NoError(t, err)
	assert.Equal(t, float64(7), resp["sum"])
}

// testGoBusCallTSHandlerThrowsTypedError — Go brainkit.Call → .ts handler
// that throws BrainkitError; caller gets a typed Go *NotFoundError via
// envelope unwrap.
func testGoBusCallTSHandlerThrowsTypedError(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-goclient-thrower.ts", `
		bus.on("fail", (msg) => {
			throw new BrainkitError("no such thing", "NOT_FOUND", { resource: "item", name: "x" });
		});
	`)
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := brainkit.Call[sdk.CustomMsg, map[string]any](env.Kit, ctx, sdk.CustomMsg{
		Topic:   "ts.ts-goclient-thrower.fail",
		Payload: []byte(`{}`),
	})
	require.Error(t, err)

	var nf *sdkerrors.NotFoundError
	assert.True(t, errors.As(err, &nf), "want *NotFoundError, got %T: %v", err, err)
}

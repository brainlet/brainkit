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
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tsCallDeployAndTrigger deploys a .ts that has a bus.on("trigger", ...)
// handler, then sends a trigger and returns the reply payload map.
func tsCallDeployAndTrigger(t *testing.T, env *suite.TestEnv, source, handlerCode string) map[string]any {
	t.Helper()
	testutil.Deploy(t, env.Kit, source, handlerCode)
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	payload, err := json.Marshal(map[string]any{})
	require.NoError(t, err)
	data, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic:   protocol.ResolveServiceTopic(source, "trigger"),
		Payload: payload,
	})
	require.NoError(t, err)
	var m map[string]any
	if len(data) > 0 {
		_ = json.Unmarshal(data, &m)
	}
	return m
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

// testTSBusCallStreamHappyPath — .ts bus.callStream consumes intermediate
// chunks and resolves with the terminal reply on the same shared caller path.
func testTSBusCallStreamHappyPath(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-callstream-server.ts", `
		bus.on("numbers", (msg) => {
			msg.send({ n: 1 });
			msg.send({ n: 2 });
			msg.send({ n: 3 });
			msg.reply({ done: true, total: 3 });
		});
	`)
	time.Sleep(100 * time.Millisecond)

	reply := tsCallDeployAndTrigger(t, env, "ts-callstream-client.ts", `
		bus.on("trigger", async (msg) => {
			let count = 0;
			let sum = 0;
			let sawCorrelation = false;
			const final = await bus.callStream("ts.ts-callstream-server.numbers", {}, {
				timeoutMs: 5000,
				bufferSize: 8,
				bufferPolicy: "block",
				onChunk: async (chunk, streamMsg) => {
					await Promise.resolve();
					count++;
					sum += chunk.n;
					sawCorrelation = sawCorrelation || !!streamMsg.correlationId;
				},
			});
			msg.reply({ final: final.done, total: final.total, count, sum, sawCorrelation });
		});
	`)
	assert.Equal(t, true, reply["final"])
	assert.Equal(t, float64(3), reply["total"])
	assert.Equal(t, float64(3), reply["count"])
	assert.Equal(t, float64(6), reply["sum"])
	assert.Equal(t, true, reply["sawCorrelation"])
}

// testTSBusCallServiceStreamHappyPath — .ts callServiceStream resolves a
// service mailbox and still receives done=false chunks before the terminal.
func testTSBusCallServiceStreamHappyPath(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-service-stream-server.ts", `
		bus.on("numbers", (msg) => {
			msg.send({ n: 4 });
			msg.send({ n: 5 });
			msg.reply({ ok: true });
		});
	`)
	time.Sleep(100 * time.Millisecond)

	reply := tsCallDeployAndTrigger(t, env, "ts-service-stream-client.ts", `
		bus.on("trigger", async (msg) => {
			let count = 0;
			let sum = 0;
			const final = await bus.callServiceStream("ts-service-stream-server.ts", "numbers", {}, {
				timeoutMs: 5000,
				onChunk: (chunk) => {
					count++;
					sum += chunk.n;
				},
			});
			msg.reply({ finalOk: final.ok, count, sum });
		});
	`)
	assert.Equal(t, true, reply["finalOk"])
	assert.Equal(t, float64(2), reply["count"])
	assert.Equal(t, float64(9), reply["sum"])
}

// testTSBusCallAndCallStreamStayAsyncUnderConcurrency proves that JS callers
// do not serialize every request/reply or stream response behind one blocking
// response path. The server increments an in-flight counter before awaiting a
// timer; if awaited handlers were serialized, maxActive would stay at 1.
func testTSBusCallAndCallStreamStayAsyncUnderConcurrency(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-async-calls-server.ts", `
		let active = 0;
		let maxActive = 0;
		function sleep(ms) { return new Promise((resolve) => setTimeout(resolve, ms)); }
		async function withActive(fn) {
			active++;
			if (active > maxActive) maxActive = active;
			try {
				return await fn();
			} finally {
				active--;
			}
		}
		bus.on("work", async (msg) => {
			await withActive(async () => {
				await sleep(120);
			});
			msg.reply({ n: msg.payload.n, maxActive });
		});
		bus.on("stream", async (msg) => {
			await withActive(async () => {
				msg.send({ n: msg.payload.n, phase: "start" });
				await sleep(120);
				msg.send({ n: msg.payload.n, phase: "end" });
			});
			msg.reply({ n: msg.payload.n, maxActive });
		});
	`)
	time.Sleep(100 * time.Millisecond)

	reply := tsCallDeployAndTrigger(t, env, "ts-async-calls-client.ts", `
		bus.on("trigger", async (msg) => {
			const started = Date.now();
			const calls = Array.from({ length: 8 }, (_, n) => {
				return bus.call("ts.ts-async-calls-server.work", { n }, { timeoutMs: 5000 });
			});
			let chunks = 0;
			const streams = Array.from({ length: 4 }, (_, n) => {
				return bus.callStream("ts.ts-async-calls-server.stream", { n: n + 100 }, {
					timeoutMs: 5000,
					bufferSize: 8,
					bufferPolicy: "block",
					onChunk: async () => {
						await Promise.resolve();
						chunks++;
					},
				});
			});
			const results = await Promise.all(calls.concat(streams));
			const maxActive = results.reduce((max, item) => Math.max(max, item.maxActive || 0), 0);
			const sum = results.reduce((total, item) => total + item.n, 0);
			msg.reply({ elapsedMs: Date.now() - started, maxActive, chunks, sum });
		});
	`)
	assert.GreaterOrEqual(t, int(reply["maxActive"].(float64)), 2)
	assert.Less(t, int(reply["elapsedMs"].(float64)), 1000)
	assert.Equal(t, float64(8), reply["chunks"])
	assert.Equal(t, float64(434), reply["sum"])
}

// testTSBusCallStreamOnChunkErrorRejects — local stream consumer failures
// abort the call and preserve typed BrainkitError details.
func testTSBusCallStreamOnChunkErrorRejects(t *testing.T, env *suite.TestEnv) {
	testutil.Deploy(t, env.Kit, "ts-callstream-error-server.ts", `
		bus.on("numbers", (msg) => {
			msg.send({ n: 1 });
			msg.reply({ ok: true });
		});
	`)
	time.Sleep(100 * time.Millisecond)

	reply := tsCallDeployAndTrigger(t, env, "ts-callstream-error-client.ts", `
		bus.on("trigger", async (msg) => {
			try {
				await bus.callStream("ts.ts-callstream-error-server.numbers", {}, {
					timeoutMs: 5000,
					onChunk: () => {
						throw new BrainkitError("bad stream chunk", "VALIDATION_ERROR", { field: "chunk" });
					},
				});
				msg.reply({ code: "NO_THROW" });
			} catch (e) {
				msg.reply({ code: e.code || "NO_CODE", field: e.details && e.details.field || "" });
			}
		});
	`)
	assert.Equal(t, "VALIDATION_ERROR", reply["code"])
	assert.Equal(t, "chunk", reply["field"])
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

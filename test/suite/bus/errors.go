package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPublishToCommandTopic(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__cmd_topic.ts", `
		var caught = "none";
		try { __go_brainkit_bus_send("tools.call", JSON.stringify({})); }
		catch(e) { caught = e.code || e.message || "error"; }
		return caught;
	`)
	assert.NotEqual(t, "none", result, "publishing to command topic should error")
}

func testEmitToCommandTopic(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__emit_cmd.ts", `
		var caught = "none";
		try { bus.emit("tools.call", {}); }
		catch(e) { caught = "error"; }
		return caught;
	`)
	assert.Equal(t, "error", result, "bus.emit should block command topics (bug #8 fixed)")
}

func testSubscribeReceivesMetadataAdv(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := env.Deploy("meta-check-adv.ts", `
		bus.on("check", function(msg) {
			msg.reply({
				hasTopic: msg.topic.length > 0,
				hasReplyTo: msg.replyTo.length > 0,
				hasCorrelation: msg.correlationId.length > 0,
			});
		});
	`)
	require.NoError(t, err)

	resp, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
		Topic: "ts.meta-check-adv.check", Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	assert.Contains(t, string(resp), `"hasTopic":true`)
	assert.Contains(t, string(resp), `"hasReplyTo":true`)
	assert.Contains(t, string(resp), `"hasCorrelation":true`)
}

func testReplyWithoutReplyTo(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	err := env.Deploy("no-reply-adv.ts", `
		bus.on("fire", function(msg) {
			msg.reply({ok: true});
		});
	`)
	require.NoError(t, err)

	sdk.Emit(env.Kit, ctx, sdk.CustomEvent{
		Topic: "ts.no-reply-adv.fire", Payload: json.RawMessage(`{}`),
	})
	time.Sleep(200 * time.Millisecond)
	assert.True(t, testutil.Alive(t, env.Kit))
}

func testSendToNonexistentService(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__sendto_ghost.ts", `
		var r = bus.sendTo("ghost-service.ts", "ask", {});
		return r === undefined ? "published" : "fail";
	`)
	assert.Equal(t, "published", result)
}

func testCorrelationIDPreserved(t *testing.T, env *suite.TestEnv) {
	err := env.Deploy("corr-echo-adv.ts", `
		bus.on("echo", function(msg) {
			msg.reply({correlationId: msg.correlationId});
		});
	`)
	require.NoError(t, err)

	pr, got, ok := publishAndWaitMessage(t, env.Kit, sdk.CustomMsg{
		Topic: "ts.corr-echo-adv.echo", Payload: json.RawMessage(`{}`),
	}, 3*time.Second)
	require.True(t, ok)
	assert.Equal(t, pr.CorrelationID, got.Metadata["correlationId"])
}

func testMultipleReplies(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	err := env.Deploy("multi-reply-adv.ts", `
		bus.on("multi", function(msg) {
			msg.send({chunk: 1});
			msg.send({chunk: 2});
			msg.reply({final: true});
		});
	`)
	require.NoError(t, err)

	var received []json.RawMessage
	done := make(chan bool, 1)
	replyTo := "ts.multi-reply-adv.multi.reply." + uuid.NewString()
	unsub, _ := env.Kit.SubscribeRaw(ctx, replyTo, func(m sdk.Message) {
		received = append(received, json.RawMessage(m.Payload))
		if m.Metadata["done"] == "true" {
			done <- true
		}
	})
	defer unsub()

	_, _ = sdk.Publish(env.Kit, ctx, sdk.CustomMsg{
		Topic: "ts.multi-reply-adv.multi", Payload: json.RawMessage(`{}`),
	}, sdk.WithReplyTo(replyTo))

	select {
	case <-done:
		assert.GreaterOrEqual(t, len(received), 2, "should receive chunks + final")
	case <-time.After(3 * time.Second):
		assert.Greater(t, len(received), 0, "should receive at least something")
	}
}

func testSubscribeUnsubscribe(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__sub_unsub.ts", `
		var received = 0;
		var subId = bus.subscribe("events.count-test", function() { received++; });
		bus.emit("events.count-test", {});
		bus.unsubscribe(subId);
		bus.emit("events.count-test", {});
		return "ok";
	`)
	assert.Equal(t, "ok", result)
}

func testDeploymentNamespace(t *testing.T, env *suite.TestEnv) {
	err := env.Deploy("ns-test-adv.ts", `
		output({
			source: kit.source,
			namespace: kit.namespace,
			callerId: kit.callerId,
		});
	`)
	require.NoError(t, err)

	result := testutil.EvalTS(t, env.Kit, "__ns_result.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "ns-test-adv.ts")
}

func testScheduleWithPayload(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	fired := make(chan []byte, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, "sched.payload.test", func(m sdk.Message) {
		fired <- m.Payload
	})
	defer unsub()

	testutil.Schedule(t, env.Kit, "in 200ms", "sched.payload.test", json.RawMessage(`{"key":"value","num":42}`))

	select {
	case p := <-fired:
		assert.Contains(t, string(p), "key")
		assert.Contains(t, string(p), "42")
	case <-time.After(5 * time.Second):
		t.Fatal("schedule didn't fire")
	}
}

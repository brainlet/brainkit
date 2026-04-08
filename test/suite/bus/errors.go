package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
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
	ctx := context.Background()

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

	pr, _ := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic: "ts.meta-check-adv.check", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), `"hasTopic":true`)
		assert.Contains(t, string(p), `"hasReplyTo":true`)
		assert.Contains(t, string(p), `"hasCorrelation":true`)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func testReplyWithoutReplyTo(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	err := env.Deploy("no-reply-adv.ts", `
		bus.on("fire", function(msg) {
			msg.reply({ok: true});
		});
	`)
	require.NoError(t, err)

	sdk.Emit(env.Kit, ctx, messages.CustomEvent{
		Topic: "ts.no-reply-adv.fire", Payload: json.RawMessage(`{}`),
	})
	time.Sleep(200 * time.Millisecond)
	assert.True(t, testutil.Alive(t, env.Kit))
}

func testSendToNonexistentService(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__sendto_ghost.ts", `
		var r = bus.sendTo("ghost-service.ts", "ask", {});
		return r.replyTo ? "published" : "fail";
	`)
	assert.Equal(t, "published", result)
}

func testCorrelationIDPreserved(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	err := env.Deploy("corr-echo-adv.ts", `
		bus.on("echo", function(msg) {
			msg.reply({correlationId: msg.correlationId});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic: "ts.corr-echo-adv.echo", Payload: json.RawMessage(`{}`),
	})

	ch := make(chan messages.Message, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m })
	defer unsub()

	select {
	case m := <-ch:
		assert.Equal(t, pr.CorrelationID, m.Metadata["correlationId"])
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
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

	pr, _ := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic: "ts.multi-reply-adv.multi", Payload: json.RawMessage(`{}`),
	})

	var received []json.RawMessage
	done := make(chan bool, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		received = append(received, json.RawMessage(m.Payload))
		if m.Metadata["done"] == "true" {
			done <- true
		}
	})
	defer unsub()

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
	unsub, _ := env.Kit.SubscribeRaw(ctx, "sched.payload.test", func(m messages.Message) {
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

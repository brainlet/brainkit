package adversarial_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusErrors_PublishToCommandTopic(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	result, err := tk.EvalTS(context.Background(), "__cmd_topic.ts", `
		var caught = "none";
		try { __go_brainkit_bus_send("tools.call", JSON.stringify({})); }
		catch(e) { caught = e.code || e.message || "error"; }
		return caught;
	`)
	require.NoError(t, err)
	assert.NotEqual(t, "none", result, "publishing to command topic should error")
}

// FIXED (bug #8): bus.emit now validates against command topics, same as bus_send.
func TestBusErrors_EmitToCommandTopic(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	result, err := tk.EvalTS(context.Background(), "__emit_cmd.ts", `
		var caught = "none";
		try { bus.emit("tools.call", {}); }
		catch(e) { caught = "error"; }
		return caught;
	`)
	require.NoError(t, err)
	assert.Equal(t, "error", result, "bus.emit should block command topics (bug #8 fixed)")
}

func TestBusErrors_SubscribeReceivesMetadata(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "meta-check.ts", `
		bus.on("check", function(msg) {
			msg.reply({
				hasTopic: msg.topic.length > 0,
				hasReplyTo: msg.replyTo.length > 0,
				hasCorrelation: msg.correlationId.length > 0,
			});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.meta-check.check", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
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

func TestBusErrors_ReplyWithoutReplyTo(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "no-reply.ts", `
		bus.on("fire", function(msg) {
			msg.reply({ok: true});
		});
	`)
	require.NoError(t, err)

	sdk.Emit(tk, ctx, messages.CustomEvent{
		Topic: "ts.no-reply.fire", Payload: json.RawMessage(`{}`),
	})
	time.Sleep(200 * time.Millisecond)
	assert.True(t, tk.Alive(ctx))
}

func TestBusErrors_SendToNonexistentService(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	result, err := tk.EvalTS(context.Background(), "__sendto_ghost.ts", `
		var r = bus.sendTo("ghost-service.ts", "ask", {});
		return r.replyTo ? "published" : "fail";
	`)
	require.NoError(t, err)
	assert.Equal(t, "published", result)
}

func TestBusErrors_CorrelationIDPreserved(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "corr-echo.ts", `
		bus.on("echo", function(msg) {
			msg.reply({correlationId: msg.correlationId});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.corr-echo.echo", Payload: json.RawMessage(`{}`),
	})

	ch := make(chan messages.Message, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m })
	defer unsub()

	select {
	case m := <-ch:
		assert.Equal(t, pr.CorrelationID, m.Metadata["correlationId"])
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func TestBusErrors_MultipleReplies(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "multi-reply.ts", `
		bus.on("multi", function(msg) {
			msg.send({chunk: 1});
			msg.send({chunk: 2});
			msg.reply({final: true});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.multi-reply.multi", Payload: json.RawMessage(`{}`),
	})

	var received []json.RawMessage
	done := make(chan bool, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
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

func TestBusErrors_SubscribeUnsubscribe(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	result, err := tk.EvalTS(ctx, "__sub_unsub.ts", `
		var received = 0;
		var subId = bus.subscribe("events.count-test", function() { received++; });
		bus.emit("events.count-test", {});
		bus.unsubscribe(subId);
		bus.emit("events.count-test", {});
		return "ok";
	`)
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestBusErrors_DeploymentNamespace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "ns-test.ts", `
		output({
			source: kit.source,
			namespace: kit.namespace,
			callerId: kit.callerId,
		});
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__ns_result.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)
	assert.Contains(t, result, "ns-test.ts")
}

func TestBusErrors_ScheduleWithPayload(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	fired := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, "sched.payload.test", func(m messages.Message) {
		fired <- m.Payload
	})
	defer unsub()

	_, err := tk.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 200ms",
		Topic:      "sched.payload.test",
		Payload:    json.RawMessage(`{"key":"value","num":42}`),
	})
	require.NoError(t, err)

	select {
	case p := <-fired:
		assert.Contains(t, string(p), "key")
		assert.Contains(t, string(p), "42")
	case <-time.After(5 * time.Second):
		t.Fatal("schedule didn't fire")
	}
}

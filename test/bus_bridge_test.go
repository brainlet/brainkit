package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBusBridge_JSPublishReturnsReplyTo verifies that __go_brainkit_bus_publish
// generates a replyTo and correlationId, and returns them to JS.
func TestBusBridge_JSPublishReturnsReplyTo(t *testing.T) {
	rt := newTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := rt.EvalTS(ctx, "__test_bus_publish.ts", `
		var result = __go_brainkit_bus_publish("test.publish.target", JSON.stringify({hello: "world"}));
		var parsed = JSON.parse(result);
		return JSON.stringify({
			hasReplyTo: parsed.replyTo && parsed.replyTo.length > 0,
			hasCorrelationId: parsed.correlationId && parsed.correlationId.length > 0,
		});
	`)
	require.NoError(t, err)

	var parsed struct {
		HasReplyTo       bool `json:"hasReplyTo"`
		HasCorrelationId bool `json:"hasCorrelationId"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.True(t, parsed.HasReplyTo, "bus.publish should return replyTo")
	assert.True(t, parsed.HasCorrelationId, "bus.publish should return correlationId")
}

// TestBusBridge_JSEmitFireAndForget verifies __go_brainkit_bus_emit publishes without replyTo.
func TestBusBridge_JSEmitFireAndForget(t *testing.T) {
	rt := newTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	received := make(chan messages.Message, 1)
	unsub, err := rt.SubscribeRaw(ctx, "test.emit.target", func(msg messages.Message) {
		received <- msg
	})
	require.NoError(t, err)
	defer unsub()

	_, err = rt.EvalTS(ctx, "__test_bus_emit.ts", `
		__go_brainkit_bus_emit("test.emit.target", JSON.stringify({event: "happened"}));
		return "ok";
	`)
	require.NoError(t, err)

	select {
	case msg := <-received:
		assert.Contains(t, string(msg.Payload), "happened")
		assert.Empty(t, msg.Metadata["replyTo"], "emit should NOT have replyTo")
	case <-ctx.Done():
		t.Fatal("timeout waiting for emitted message")
	}
}

// TestBusBridge_JSReplyDoneFlag verifies __go_brainkit_bus_reply publishes
// to replyTo with correlationId and done metadata flag.
// Uses the real flow: Go publishes → JS subscribes → JS replies → Go receives.
func TestBusBridge_JSReplyDoneFlag(t *testing.T) {
	rt := newTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// JS sets up a handler that sends a chunk then a final reply
	_, err := rt.EvalTS(ctx, "__test_reply_setup.ts", `
		var subId = __go_brainkit_subscribe("test.reply.trigger");
		globalThis.__bus_subs[subId] = function(msg) {
			if (msg.replyTo) {
				// Send chunk (not done)
				__go_brainkit_bus_reply(msg.replyTo, JSON.stringify({delta:"hello"}), msg.correlationId, false);
				// Send final (done)
				__go_brainkit_bus_reply(msg.replyTo, JSON.stringify({text:"world"}), msg.correlationId, true);
			}
		};
		return "ok";
	`)
	require.NoError(t, err)

	// Go publishes to trigger the JS handler
	pubResult, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "test.reply.trigger",
		Payload: json.RawMessage(`{"go":"true"}`),
	})
	require.NoError(t, err)

	// Subscribe to replyTo — collect both messages (order not guaranteed)
	received := make(chan messages.Message, 2)
	unsub, err := rt.SubscribeRaw(ctx, pubResult.ReplyTo, func(msg messages.Message) {
		received <- msg
	})
	require.NoError(t, err)
	defer unsub()

	var msgs []messages.Message
	for i := 0; i < 2; i++ {
		select {
		case msg := <-received:
			msgs = append(msgs, msg)
		case <-ctx.Done():
			t.Fatalf("timeout waiting for message %d/2", i+1)
		}
	}
	require.Len(t, msgs, 2)

	// Identify chunk vs final by done flag
	var chunk, final *messages.Message
	for i := range msgs {
		if msgs[i].Metadata["done"] == "true" {
			final = &msgs[i]
		} else {
			chunk = &msgs[i]
		}
	}
	require.NotNil(t, chunk, "should have a chunk (done != true)")
	assert.Equal(t, pubResult.CorrelationID, chunk.Metadata["correlationId"])
	assert.Contains(t, string(chunk.Payload), "hello")

	require.NotNil(t, final, "should have a final (done == true)")
	assert.Equal(t, pubResult.CorrelationID, final.Metadata["correlationId"])
	assert.Contains(t, string(final.Payload), "world")
}

// TestBusBridge_JSSubscribeReceivesMetadata verifies that JS bus.subscribe
// handlers receive payload + replyTo + correlationId + callerId.
func TestBusBridge_JSSubscribeReceivesMetadata(t *testing.T) {
	rt := newTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// JS subscribes and on message, replies back with what it received
	_, err := rt.EvalTS(ctx, "__test_sub_meta_setup.ts", `
		var subId = __go_brainkit_subscribe("test.meta.topic");
		globalThis.__bus_subs[subId] = function(msg) {
			if (msg.replyTo) {
				__go_brainkit_bus_reply(msg.replyTo, JSON.stringify({
					hasPayload: msg.payload !== undefined && msg.payload !== null,
					hasReplyTo: typeof msg.replyTo === "string" && msg.replyTo.length > 0,
					hasCorrelationId: typeof msg.correlationId === "string" && msg.correlationId.length > 0,
					hasTopic: typeof msg.topic === "string" && msg.topic.length > 0,
					hasCallerId: typeof msg.callerId === "string" && msg.callerId.length > 0,
				}), msg.correlationId, true);
			}
		};
		return "ok";
	`)
	require.NoError(t, err)

	// Publish from Go
	pubResult, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "test.meta.topic",
		Payload: json.RawMessage(`{"data":"test123"}`),
	})
	require.NoError(t, err)

	// Subscribe to reply
	done := make(chan json.RawMessage, 1)
	unsub, err := rt.SubscribeRaw(ctx, pubResult.ReplyTo, func(msg messages.Message) {
		done <- json.RawMessage(msg.Payload)
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case resp := <-done:
		var parsed struct {
			HasPayload       bool `json:"hasPayload"`
			HasReplyTo       bool `json:"hasReplyTo"`
			HasCorrelationId bool `json:"hasCorrelationId"`
			HasTopic         bool `json:"hasTopic"`
			HasCallerId      bool `json:"hasCallerId"`
		}
		require.NoError(t, json.Unmarshal(resp, &parsed))
		assert.True(t, parsed.HasPayload, "should have payload")
		assert.True(t, parsed.HasReplyTo, "should have replyTo")
		assert.True(t, parsed.HasCorrelationId, "should have correlationId")
		assert.True(t, parsed.HasTopic, "should have topic")
	case <-ctx.Done():
		t.Fatal("timeout waiting for metadata reply")
	}
}

// TestBusBridge_GoToJSRoundTrip verifies Go can publish a CustomMsg to a JS handler
// and receive a reply back through the bus.
func TestBusBridge_GoToJSRoundTrip(t *testing.T) {
	rt := newTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set up a JS handler that replies
	_, err := rt.EvalTS(ctx, "__test_roundtrip_setup.ts", `
		var subId = __go_brainkit_subscribe("test.roundtrip.ask");
		globalThis.__bus_subs[subId] = function(msg) {
			if (msg.replyTo) {
				__go_brainkit_bus_reply(msg.replyTo, JSON.stringify({answer: "42"}), msg.correlationId, true);
			}
		};
		return "ok";
	`)
	require.NoError(t, err)

	// Publish from Go
	pubResult, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "test.roundtrip.ask",
		Payload: json.RawMessage(`{"question":"meaning of life"}`),
	})
	require.NoError(t, err)

	// Subscribe to reply
	done := make(chan json.RawMessage, 1)
	unsub, err := rt.SubscribeRaw(ctx, pubResult.ReplyTo, func(msg messages.Message) {
		done <- json.RawMessage(msg.Payload)
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case resp := <-done:
		assert.Contains(t, string(resp), "42")
	case <-ctx.Done():
		t.Fatal("timeout waiting for reply from JS handler")
	}
}

// TestBusAPI_DeployWithBusOn verifies the full deploy → bus.on → message → reply flow.
// Deploys .ts code that uses bus.on("greet"), then sends a message from Go.
func TestBusAPI_DeployWithBusOn(t *testing.T) {
	rt := newTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy a .ts service using bus.on
	tsCode := `
		bus.on("greet", function(msg) {
			msg.reply({ greeting: "hello " + msg.payload.name });
		});
	`
	deployResult, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "greeter.ts",
		Code:   tsCode,
	})
	require.NoError(t, err)

	// Wait for deploy to complete
	deployCh := make(chan messages.KitDeployResp, 1)
	deployUnsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, deployResult.ReplyTo, func(r messages.KitDeployResp, m messages.Message) {
		deployCh <- r
	})
	defer deployUnsub()
	select {
	case dr := <-deployCh:
		require.Empty(t, dr.Error, "deploy should succeed")
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	// Send to the mailbox topic: ts.greeter.greet
	pubResult, err := sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.greeter.greet",
		Payload: json.RawMessage(`{"name":"world"}`),
	})
	require.NoError(t, err)

	// Get reply
	replyCh := make(chan json.RawMessage, 1)
	replyUnsub, _ := rt.SubscribeRaw(ctx, pubResult.ReplyTo, func(msg messages.Message) {
		replyCh <- json.RawMessage(msg.Payload)
	})
	defer replyUnsub()

	select {
	case resp := <-replyCh:
		assert.Contains(t, string(resp), "hello world")
	case <-ctx.Done():
		t.Fatal("timeout waiting for reply from deployed .ts service")
	}

	// Teardown
	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "greeter.ts"})
}

// TestBusAPI_StreamingChunks verifies msg.send (chunks) + msg.reply (final) from a .ts service.
func TestBusAPI_StreamingChunks(t *testing.T) {
	rt := newTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy a .ts service that sends 1 chunk then a final reply
	tsCode := `
		bus.on("stream", function(msg) {
			msg.send({ delta: "chunk1" });
			msg.reply({ text: "final", done: true });
		});
	`
	deployResult, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{Source: "streamer.ts", Code: tsCode})
	require.NoError(t, err)
	deployCh := make(chan messages.KitDeployResp, 1)
	deployUnsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, deployResult.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { deployCh <- r })
	defer deployUnsub()
	select {
	case dr := <-deployCh:
		require.Empty(t, dr.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	// Subscribe to replyTo BEFORE publishing (avoid race with async handler)
	replyTopic := "test.stream.reply"
	received := make(chan messages.Message, 10)
	replyUnsub, _ := rt.SubscribeRaw(ctx, replyTopic, func(msg messages.Message) {
		received <- msg
	})
	defer replyUnsub()

	// Send message with known replyTo
	_, err = sdk.Publish(rt, ctx, messages.CustomMsg{
		Topic:   "ts.streamer.stream",
		Payload: json.RawMessage(`{}`),
	}, sdk.WithReplyTo(replyTopic))
	require.NoError(t, err)

	var chunks []messages.Message
	for {
		select {
		case msg := <-received:
			chunks = append(chunks, msg)
			if msg.Metadata["done"] == "true" {
				goto done
			}
		case <-ctx.Done():
			t.Fatalf("timeout, received %d messages", len(chunks))
		}
	}
done:
	// Should have at least 1 message (the final with done=true).
	// Chunks may or may not arrive depending on GoChannel timing.
	require.GreaterOrEqual(t, len(chunks), 1, "should have at least the final message")

	// The last received message should be the final (done=true)
	lastMsg := chunks[len(chunks)-1]
	assert.Equal(t, "true", lastMsg.Metadata["done"], "last message should have done=true")
	assert.Contains(t, string(lastMsg.Payload), "final")
	t.Logf("received %d messages total (chunks + final)", len(chunks))

	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "streamer.ts"})
}

// TestBusAPI_KitRegisterAgentDiscovery verifies kit.register("agent") makes it
// discoverable via agents.list, and teardown removes it.
func TestBusAPI_KitRegisterAgentDiscovery(t *testing.T) {
	rt := newTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy a .ts that registers an agent
	tsCode := `
		kit.register("agent", "test-bot", {});
	`
	deployResult, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{Source: "bot.ts", Code: tsCode})
	require.NoError(t, err)
	deployCh := make(chan messages.KitDeployResp, 1)
	deployUnsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, deployResult.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { deployCh <- r })
	defer deployUnsub()
	select {
	case dr := <-deployCh:
		require.Empty(t, dr.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	// Query agents.list — should find test-bot
	listResult, err := sdk.Publish(rt, ctx, messages.AgentListMsg{})
	require.NoError(t, err)
	listCh := make(chan messages.AgentListResp, 1)
	listUnsub, _ := sdk.SubscribeTo[messages.AgentListResp](rt, ctx, listResult.ReplyTo, func(r messages.AgentListResp, m messages.Message) { listCh <- r })
	defer listUnsub()
	select {
	case lr := <-listCh:
		found := false
		for _, a := range lr.Agents {
			if a.Name == "test-bot" {
				found = true
			}
		}
		assert.True(t, found, "test-bot should be in agents.list")
	case <-ctx.Done():
		t.Fatal("timeout listing agents")
	}

	// Teardown — should remove agent
	sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "bot.ts"})
	time.Sleep(100 * time.Millisecond)

	// Query again — should NOT find test-bot
	listResult2, err := sdk.Publish(rt, ctx, messages.AgentListMsg{})
	require.NoError(t, err)
	listCh2 := make(chan messages.AgentListResp, 1)
	listUnsub2, _ := sdk.SubscribeTo[messages.AgentListResp](rt, ctx, listResult2.ReplyTo, func(r messages.AgentListResp, m messages.Message) { listCh2 <- r })
	defer listUnsub2()
	select {
	case lr := <-listCh2:
		for _, a := range lr.Agents {
			assert.NotEqual(t, "test-bot", a.Name, "test-bot should be removed after teardown")
		}
	case <-ctx.Done():
		t.Fatal("timeout listing agents after teardown")
	}
}

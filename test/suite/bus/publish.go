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

// testJSPublishReturnsReplyTo verifies that __go_brainkit_bus_publish
// generates a replyTo and correlationId, and returns them to JS.
func testJSPublishReturnsReplyTo(t *testing.T, env *suite.TestEnv) {
	result := testutil.EvalTS(t, env.Kit, "__test_bus_publish.ts", `
		var result = __go_brainkit_bus_publish("test.publish.target", JSON.stringify({hello: "world"}));
		var parsed = JSON.parse(result);
		return JSON.stringify({
			hasReplyTo: parsed.replyTo && parsed.replyTo.length > 0,
			hasCorrelationId: parsed.correlationId && parsed.correlationId.length > 0,
		});
	`)

	var parsed struct {
		HasReplyTo       bool `json:"hasReplyTo"`
		HasCorrelationId bool `json:"hasCorrelationId"`
	}
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.True(t, parsed.HasReplyTo, "bus.publish should return replyTo")
	assert.True(t, parsed.HasCorrelationId, "bus.publish should return correlationId")
}

// testJSEmitFireAndForget verifies __go_brainkit_bus_emit publishes without replyTo.
func testJSEmitFireAndForget(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	received := make(chan messages.Message, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, "test.emit.target", func(msg messages.Message) {
		received <- msg
	})
	require.NoError(t, err)
	defer unsub()

	testutil.EvalTS(t, env.Kit, "__test_bus_emit.ts", `
		__go_brainkit_bus_emit("test.emit.target", JSON.stringify({event: "happened"}));
		return "ok";
	`)

	select {
	case msg := <-received:
		assert.Contains(t, string(msg.Payload), "happened")
		assert.Empty(t, msg.Metadata["replyTo"], "emit should NOT have replyTo")
	case <-ctx.Done():
		t.Fatal("timeout waiting for emitted message")
	}
}

// testJSReplyDoneFlag verifies __go_brainkit_bus_reply publishes
// to replyTo with correlationId and done metadata flag.
func testJSReplyDoneFlag(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testutil.EvalTS(t, env.Kit, "__test_reply_setup.ts", `
		var subId = __go_brainkit_subscribe("test.reply.trigger");
		globalThis.__bus_subs[subId] = function(msg) {
			if (msg.replyTo) {
				__go_brainkit_bus_reply(msg.replyTo, JSON.stringify({delta:"hello"}), msg.correlationId, false);
				__go_brainkit_bus_reply(msg.replyTo, JSON.stringify({text:"world"}), msg.correlationId, true);
			}
		};
		return "ok";
	`)

	pubResult, err := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic:   "test.reply.trigger",
		Payload: json.RawMessage(`{"go":"true"}`),
	})
	require.NoError(t, err)

	received := make(chan messages.Message, 2)
	unsub, err := env.Kit.SubscribeRaw(ctx, pubResult.ReplyTo, func(msg messages.Message) {
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

// testJSSubscribeReceivesMetadata verifies that JS bus.subscribe
// handlers receive payload + replyTo + correlationId + callerId.
func testJSSubscribeReceivesMetadata(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testutil.EvalTS(t, env.Kit, "__test_sub_meta_setup.ts", `
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

	pubResult, err := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic:   "test.meta.topic",
		Payload: json.RawMessage(`{"data":"test123"}`),
	})
	require.NoError(t, err)

	done := make(chan json.RawMessage, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pubResult.ReplyTo, func(msg messages.Message) {
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

// testGoToJSRoundTrip verifies Go can publish a CustomMsg to a JS handler
// and receive a reply back through the bus.
func testGoToJSRoundTrip(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testutil.EvalTS(t, env.Kit, "__test_roundtrip_setup.ts", `
		var subId = __go_brainkit_subscribe("test.roundtrip.ask");
		globalThis.__bus_subs[subId] = function(msg) {
			if (msg.replyTo) {
				__go_brainkit_bus_reply(msg.replyTo, JSON.stringify({answer: "42"}), msg.correlationId, true);
			}
		};
		return "ok";
	`)

	pubResult, err := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic:   "test.roundtrip.ask",
		Payload: json.RawMessage(`{"question":"meaning of life"}`),
	})
	require.NoError(t, err)

	done := make(chan json.RawMessage, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pubResult.ReplyTo, func(msg messages.Message) {
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

// testDeployWithBusOn verifies the full deploy -> bus.on -> message -> reply flow.
func testDeployWithBusOn(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tsCode := `
		bus.on("greet", function(msg) {
			msg.reply({ greeting: "hello " + msg.payload.name });
		});
	`
	deployResult, err := sdk.Publish(env.Kit, ctx, messages.KitDeployMsg{
		Source: "greeter.ts",
		Code:   tsCode,
	})
	require.NoError(t, err)

	deployCh := make(chan messages.KitDeployResp, 1)
	deployUnsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kit, ctx, deployResult.ReplyTo, func(r messages.KitDeployResp, m messages.Message) {
		deployCh <- r
	})
	defer deployUnsub()
	select {
	case dr := <-deployCh:
		require.Empty(t, dr.Error, "deploy should succeed")
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	pubResult, err := sdk.SendToService(env.Kit, ctx, "greeter.ts", "greet", json.RawMessage(`{"name":"world"}`))
	require.NoError(t, err)

	replyCh := make(chan json.RawMessage, 1)
	replyUnsub, _ := env.Kit.SubscribeRaw(ctx, pubResult.ReplyTo, func(msg messages.Message) {
		replyCh <- json.RawMessage(msg.Payload)
	})
	defer replyUnsub()

	select {
	case resp := <-replyCh:
		assert.Contains(t, string(resp), "hello world")
	case <-ctx.Done():
		t.Fatal("timeout waiting for reply from deployed .ts service")
	}

	sdk.Publish(env.Kit, ctx, messages.KitTeardownMsg{Source: "greeter.ts"})
}

// testStreamingChunks verifies msg.send (chunks) + msg.reply (final) from a .ts service.
func testStreamingChunks(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tsCode := `
		bus.on("stream", function(msg) {
			msg.send({ delta: "chunk1" });
			msg.reply({ text: "final", done: true });
		});
	`
	deployResult, err := sdk.Publish(env.Kit, ctx, messages.KitDeployMsg{Source: "streamer.ts", Code: tsCode})
	require.NoError(t, err)
	deployCh := make(chan messages.KitDeployResp, 1)
	deployUnsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kit, ctx, deployResult.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { deployCh <- r })
	defer deployUnsub()
	select {
	case dr := <-deployCh:
		require.Empty(t, dr.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	replyTopic := "test.stream.reply"
	received := make(chan messages.Message, 10)
	replyUnsub, _ := env.Kit.SubscribeRaw(ctx, replyTopic, func(msg messages.Message) {
		received <- msg
	})
	defer replyUnsub()

	_, err = sdk.SendToService(env.Kit, ctx, "streamer.ts", "stream", json.RawMessage(`{}`), sdk.WithReplyTo(replyTopic))
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
	require.GreaterOrEqual(t, len(chunks), 1, "should have at least the final message")
	lastMsg := chunks[len(chunks)-1]
	assert.Equal(t, "true", lastMsg.Metadata["done"], "last message should have done=true")
	assert.Contains(t, string(lastMsg.Payload), "final")

	sdk.Publish(env.Kit, ctx, messages.KitTeardownMsg{Source: "streamer.ts"})
}

// testKitRegisterAgentDiscovery verifies kit.register("agent") makes it
// discoverable via agents.list, and teardown removes it.
func testKitRegisterAgentDiscovery(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tsCode := `
		kit.register("agent", "test-bot", {});
	`
	deployResult, err := sdk.Publish(env.Kit, ctx, messages.KitDeployMsg{Source: "bot.ts", Code: tsCode})
	require.NoError(t, err)
	deployCh := make(chan messages.KitDeployResp, 1)
	deployUnsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kit, ctx, deployResult.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { deployCh <- r })
	defer deployUnsub()
	select {
	case dr := <-deployCh:
		require.Empty(t, dr.Error)
	case <-ctx.Done():
		t.Fatal("timeout deploying")
	}

	listResult, err := sdk.Publish(env.Kit, ctx, messages.AgentListMsg{})
	require.NoError(t, err)
	listCh := make(chan messages.AgentListResp, 1)
	listUnsub, _ := sdk.SubscribeTo[messages.AgentListResp](env.Kit, ctx, listResult.ReplyTo, func(r messages.AgentListResp, m messages.Message) { listCh <- r })
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

	sdk.Publish(env.Kit, ctx, messages.KitTeardownMsg{Source: "bot.ts"})
	time.Sleep(100 * time.Millisecond)

	listResult2, err := sdk.Publish(env.Kit, ctx, messages.AgentListMsg{})
	require.NoError(t, err)
	listCh2 := make(chan messages.AgentListResp, 1)
	listUnsub2, _ := sdk.SubscribeTo[messages.AgentListResp](env.Kit, ctx, listResult2.ReplyTo, func(r messages.AgentListResp, m messages.Message) { listCh2 <- r })
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

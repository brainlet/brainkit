package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConcurrentPublish50 — 50 concurrent publishes, verifying delivery.
// Ported from adversarial/backend_advanced_test.go:TestBackendAdvanced_ConcurrentPublish.
func testConcurrentPublish50(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	var received atomic.Int64
	unsub, err := env.Kit.SubscribeRaw(ctx, "incoming.concurrent.suite", func(m messages.Message) {
		received.Add(1)
	})
	require.NoError(t, err)
	defer unsub()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			env.Kit.PublishRaw(ctx, "incoming.concurrent.suite", json.RawMessage(fmt.Sprintf(`{"n":%d}`, n)))
		}(i)
	}
	wg.Wait()

	time.Sleep(500 * time.Millisecond)
	count := received.Load()
	assert.Greater(t, count, int64(0), "should receive messages from concurrent publish")
	t.Logf("received %d/50 messages", count)
}

// testLargePayload100KB — 100KB message delivery via deploy handler.
// Ported from adversarial/backend_matrix_test.go:TestBackendMatrix_LargePayload.
func testLargePayload100KB(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "big-msg-suite.ts", `
		bus.on("big", function(msg) {
			var size = JSON.stringify(msg.payload).length;
			msg.reply({size: size});
		});
	`)

	big := make([]byte, 100000)
	for i := range big {
		big[i] = 'x'
	}
	payload, _ := json.Marshal(map[string]string{"data": string(big)})

	pr, err := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic:   "ts.big-msg-suite.big",
		Payload: payload,
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "size")
	case <-ctx.Done():
		t.Fatal("timeout with 100KB payload")
	}
}

// testDottedTopicNames — topics with dots (sanitizer-sensitive).
// Ported from adversarial/backend_matrix_test.go:TestBackendMatrix_DottedTopicNames.
func testDottedTopicNames(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "my.dotted.agent.suite.ts", `
		bus.on("ask", function(msg) { msg.reply({dotted: true}); });
	`)

	pr, err := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic:   "ts.my.dotted.agent.suite.ask",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "dotted")
	case <-ctx.Done():
		t.Fatal("timeout with dotted topic")
	}
}

// testDeployHandlerCall — deploy .ts handler, publish, get reply via tool call.
// Ported from adversarial/backend_advanced_test.go:TestBackendAdvanced_DeployHandlerCall.
func testDeployHandlerCall(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "backend-handler-suite.ts", `
		bus.on("ask", async function(msg) {
			var r = await tools.call("echo", {message: "via-suite"});
			msg.reply(r);
		});
	`)

	pr, err := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic: "ts.backend-handler-suite.ask", Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "via-suite")
	case <-ctx.Done():
		t.Fatal("timeout on deploy+handler+call")
	}
}

// testPublishReply — deploy handler, publish, get reply.
// Ported from adversarial/backend_matrix_test.go:TestBackendMatrix_PublishReply.
func testPublishReply(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "backend-reply-suite.ts", `
		bus.on("ping", function(msg) { msg.reply({backend: "works"}); });
	`)

	pr, err := sdk.Publish(env.Kit, ctx, messages.CustomMsg{
		Topic:   "ts.backend-reply-suite.ping",
		Payload: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "works")
	case <-ctx.Done():
		t.Fatal("timeout on publish+reply")
	}
}

// testErrorCodeOnBus — error codes survive transport (NOT_FOUND for nonexistent tool).
// Ported from adversarial/backend_matrix_test.go:TestBackendMatrix_ErrorCodeOnBus.
func testErrorCodeOnBus(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolCallMsg{Name: "ghost-backend-tool-suite"})
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		code := suite.ResponseCode(payload)
		assert.Equal(t, "NOT_FOUND", code, "error code should survive transport")
	case <-ctx.Done():
		t.Fatal("timeout on error code test")
	}
}

// testTransportCompliancePublishSubscribe — basic pub/sub contract on memory transport.
// Ported from transport/compliance_test.go:TestTransport_Compliance/PublishSubscribe.
func testTransportCompliancePublishSubscribe(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	received := make(chan []byte, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, "test.compliance.topic", func(msg messages.Message) {
		received <- msg.Payload
	})
	require.NoError(t, err)
	defer unsub()

	env.Kit.PublishRaw(ctx, "test.compliance.topic", []byte(`{"hello":"world"}`))

	select {
	case payload := <-received:
		assert.JSONEq(t, `{"hello":"world"}`, string(payload))
	case <-ctx.Done():
		t.Fatal("timeout waiting for message")
	}
}

// testTransportComplianceCorrelationID — publish returns a non-empty correlationID.
// Ported from transport/compliance_test.go:TestTransport_Compliance/CorrelationID.
func testTransportComplianceCorrelationID(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	received := make(chan string, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, "corr.topic.suite", func(msg messages.Message) {
		received <- msg.Metadata["correlationId"]
	})
	require.NoError(t, err)
	defer unsub()

	env.Kit.PublishRaw(ctx, "corr.topic.suite", []byte(`{}`))

	select {
	case gotCorrID := <-received:
		assert.NotEmpty(t, gotCorrID, "correlationId should be present in metadata")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testTransportComplianceDottedTopics — dotted topics work on memory transport.
// Ported from transport/compliance_test.go:TestTransport_Compliance/DottedTopics.
func testTransportComplianceDottedTopics(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	received := make(chan bool, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, "ai.generate.suite", func(msg messages.Message) {
		received <- true
	})
	require.NoError(t, err)
	defer unsub()

	env.Kit.PublishRaw(ctx, "ai.generate.suite", []byte(`{"model":"test"}`))

	select {
	case <-received:
		// OK — dotted topic works
	case <-ctx.Done():
		t.Fatal("timeout — dotted topic failed")
	}
}

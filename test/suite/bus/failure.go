package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

func testSyncThrowErrorResponse(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "thrower.ts",
		Code:   `bus.on("fail", (msg) => { throw new Error("sync boom"); });`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	sendPR, _ := sdk.SendToService(env.Kernel, ctx, "thrower.ts", "fail", map[string]bool{"x": true})
	errCh := make(chan string, 1)
	replyUnsub, _ := env.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		if e, ok := resp["error"].(string); ok {
			errCh <- e
		}
	})
	defer replyUnsub()

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "sync boom")
	case <-ctx.Done():
		t.Fatal("timeout — caller should get error response, not silent timeout")
	}
}

func testAsyncRejectionErrorResponse(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(env.Kernel, ctx, messages.KitDeployMsg{
		Source: "async-fail.ts",
		Code:   `bus.on("fail", async (msg) => { throw new Error("async boom"); });`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](env.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	sendPR, _ := sdk.SendToService(env.Kernel, ctx, "async-fail.ts", "fail", map[string]bool{"x": true})
	errCh := make(chan string, 1)
	replyUnsub, _ := env.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		if e, ok := resp["error"].(string); ok {
			errCh <- e
		}
	})
	defer replyUnsub()

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "async boom")
	case <-ctx.Done():
		t.Fatal("timeout — caller should get error response for async rejection")
	}
}

// testHandlerFailedEventEmitted needs its own kernel to avoid picking up
// stale bus.handler.failed events from previous failure tests on the shared kernel.
func testHandlerFailedEventEmitted(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to bus.handler.failed BEFORE deploying+sending
	eventCh := make(chan messages.HandlerFailedEvent, 1)
	eventUnsub, _ := sdk.SubscribeTo[messages.HandlerFailedEvent](freshEnv.Kernel, ctx, "bus.handler.failed", func(evt messages.HandlerFailedEvent, _ messages.Message) {
		eventCh <- evt
	})
	defer eventUnsub()

	pr, _ := sdk.Publish(freshEnv.Kernel, ctx, messages.KitDeployMsg{
		Source: "event-emitter-fail.ts",
		Code:   `bus.on("fail", (msg) => { throw new Error("event test"); });`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](freshEnv.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	sdk.SendToService(freshEnv.Kernel, ctx, "event-emitter-fail.ts", "fail", map[string]bool{"x": true})

	select {
	case evt := <-eventCh:
		assert.Contains(t, evt.Error, "event test")
		assert.False(t, evt.WillRetry)
	case <-ctx.Done():
		t.Fatal("timeout — bus.handler.failed event should be emitted")
	}
}

// testRetryPolicyRetries creates its own kernel with RetryPolicies configured.
// Cannot use the shared env because retry policies are kernel-level config.
func testRetryPolicyRetries(t *testing.T, _ *suite.TestEnv) {
	retryEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.retry-test.*": {
			MaxRetries:    2,
			InitialDelay:  100 * time.Millisecond,
			BackoffFactor: 2.0,
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(retryEnv.Kernel, ctx, messages.KitDeployMsg{
		Source: "retry-test.ts",
		Code: `
			var _attempts = 0;
			bus.on("try", (msg) => {
				_attempts++;
				if (_attempts <= 2) {
					throw new Error("attempt " + _attempts + " failed");
				}
				msg.reply({ attempts: _attempts, ok: true });
			});
		`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](retryEnv.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	sendPR, _ := sdk.SendToService(retryEnv.Kernel, ctx, "retry-test.ts", "try", map[string]bool{"x": true})
	replyCh := make(chan map[string]any, 1)
	replyUnsub, _ := retryEnv.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer replyUnsub()

	select {
	case resp := <-replyCh:
		assert.Equal(t, true, resp["ok"])
	case <-ctx.Done():
		t.Fatal("timeout — retries should eventually succeed")
	}
}

// testRetryExhaustedDeadLetter creates its own kernel with dead letter config.
func testRetryExhaustedDeadLetter(t *testing.T, _ *suite.TestEnv) {
	dlEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.dl-test.*": {
			MaxRetries:      1,
			InitialDelay:    50 * time.Millisecond,
			DeadLetterTopic: "dead-letter",
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(dlEnv.Kernel, ctx, messages.KitDeployMsg{
		Source: "dl-test.ts",
		Code:   `bus.on("fail", (msg) => { throw new Error("always fails"); });`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](dlEnv.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	dlCh := make(chan json.RawMessage, 1)
	dlUnsub, _ := dlEnv.Kernel.SubscribeRaw(ctx, "dead-letter", func(msg messages.Message) {
		dlCh <- json.RawMessage(msg.Payload)
	})
	defer dlUnsub()

	errCh := make(chan string, 1)
	sendPR, _ := sdk.SendToService(dlEnv.Kernel, ctx, "dl-test.ts", "fail", map[string]bool{"x": true})
	replyUnsub, _ := dlEnv.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		if e, ok := resp["error"].(string); ok {
			errCh <- e
		}
	})
	defer replyUnsub()

	select {
	case dl := <-dlCh:
		var parsed map[string]any
		json.Unmarshal(dl, &parsed)
		assert.Contains(t, parsed["error"], "always fails")
	case <-ctx.Done():
		t.Fatal("timeout — dead letter should be published")
	}

	select {
	case errMsg := <-errCh:
		assert.Contains(t, errMsg, "1 retries")
	case <-time.After(2 * time.Second):
		t.Fatal("error response should be sent after exhaustion")
	}
}

// testExhaustedEventEmitted creates its own kernel with retry config.
func testExhaustedEventEmitted(t *testing.T, _ *suite.TestEnv) {
	exEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.exhaust-evt.*": {
			MaxRetries:   1,
			InitialDelay: 50 * time.Millisecond,
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(exEnv.Kernel, ctx, messages.KitDeployMsg{
		Source: "exhaust-evt.ts",
		Code:   `bus.on("fail", (msg) => { throw new Error("exhaust event"); });`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](exEnv.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	exhaustedCh := make(chan messages.HandlerExhaustedEvent, 1)
	exUnsub, _ := sdk.SubscribeTo[messages.HandlerExhaustedEvent](exEnv.Kernel, ctx, "bus.handler.exhausted", func(evt messages.HandlerExhaustedEvent, _ messages.Message) {
		exhaustedCh <- evt
	})
	defer exUnsub()

	sdk.SendToService(exEnv.Kernel, ctx, "exhaust-evt.ts", "fail", map[string]bool{"x": true})

	select {
	case evt := <-exhaustedCh:
		assert.Contains(t, evt.Error, "exhaust event")
		assert.Equal(t, 1, evt.RetryCount)
	case <-ctx.Done():
		t.Fatal("timeout — bus.handler.exhausted event should be emitted")
	}
}

// testRetryPreservesReplyTo creates its own kernel with retry config.
func testRetryPreservesReplyTo(t *testing.T, _ *suite.TestEnv) {
	rpEnv := suite.Full(t, suite.WithRetryPolicies(map[string]brainkit.RetryPolicy{
		"ts.replyto-test.*": {
			MaxRetries:   1,
			InitialDelay: 100 * time.Millisecond,
		},
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(rpEnv.Kernel, ctx, messages.KitDeployMsg{
		Source: "replyto-test.ts",
		Code: `
			var _count = 0;
			bus.on("try", (msg) => {
				_count++;
				if (_count === 1) throw new Error("first fail");
				msg.reply({ ok: true, attempt: _count });
			});
		`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rpEnv.Kernel, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	sendPR, _ := sdk.SendToService(rpEnv.Kernel, ctx, "replyto-test.ts", "try", map[string]bool{"x": true})
	replyCh := make(chan map[string]any, 1)
	replyUnsub, _ := rpEnv.Kernel.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer replyUnsub()

	select {
	case resp := <-replyCh:
		assert.Equal(t, true, resp["ok"])
		assert.Equal(t, float64(2), resp["attempt"])
	case <-ctx.Done():
		t.Fatal("timeout — original caller should receive the success reply after retry")
	}
}
